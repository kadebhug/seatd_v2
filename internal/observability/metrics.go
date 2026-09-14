package observability

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/realtime"
)

type ServiceMetrics struct {
	cfg      app.Config
	logger   *slog.Logger
	registry *prometheus.Registry

	requests *prometheus.CounterVec
	errors   *prometheus.CounterVec
	duration *prometheus.HistogramVec

	guestAbuse       *prometheus.CounterVec
	deviceHeartbeats *prometheus.CounterVec
	webhooks         *prometheus.CounterVec
	reconciliations  *prometheus.CounterVec
	reconcileDiscrep *prometheus.GaugeVec
	analyticsRebuild *prometheus.CounterVec
	outbox           *OutboxMetrics
}

func NewServiceMetrics(cfg app.Config, logger *slog.Logger) *ServiceMetrics {
	if logger == nil {
		logger = slog.Default()
	}
	registry := prometheus.NewRegistry()
	labels := prometheus.Labels{
		"service":       cfg.Name,
		"environment":   cfg.Environment,
		"build_version": cfg.Version.Version,
		"commit":        cfg.Version.Commit,
	}
	registerer := prometheus.WrapRegistererWith(labels, registry)

	m := &ServiceMetrics{
		cfg:      cfg,
		logger:   logger,
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "http_requests_total",
			Help:      "Total HTTP requests served.",
		}, []string{"method", "route", "status"}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "http_request_errors_total",
			Help:      "Total HTTP requests with 5xx responses.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "seatd",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP request duration in seconds.",
			Buckets:   []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route", "status"}),
		guestAbuse: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "guest_qr_abuse_events_total",
			Help:      "Guest QR abuse or rate-limit decisions.",
		}, []string{"decision"}),
		deviceHeartbeats: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "device_heartbeats_total",
			Help:      "Device heartbeat attempts by result, type, and platform.",
		}, []string{"result", "device_type", "platform"}),
		webhooks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "integration_webhooks_total",
			Help:      "Integration webhook attempts by vendor and outcome.",
		}, []string{"vendor", "outcome"}),
		reconciliations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "integration_reconciliations_total",
			Help:      "Integration reconciliation runs by outcome.",
		}, []string{"outcome"}),
		reconcileDiscrep: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "seatd",
			Name:      "integration_reconciliation_discrepancies",
			Help:      "Discrepancies found in the most recent observed reconciliation run.",
		}, []string{}),
		analyticsRebuild: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "analytics_rebuilds_total",
			Help:      "Manual analytics rebuild attempts by outcome.",
		}, []string{"outcome"}),
	}

	registerer.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		m.requests,
		m.errors,
		m.duration,
		m.guestAbuse,
		m.deviceHeartbeats,
		m.webhooks,
		m.reconciliations,
		m.reconcileDiscrep,
		m.analyticsRebuild,
	)
	return m
}

func (m *ServiceMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *ServiceMetrics) Middleware(next http.Handler) http.Handler {
	if m == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)

		status := strconv.Itoa(recorder.status)
		route := routePattern(r)
		m.requests.WithLabelValues(r.Method, route, status).Inc()
		m.duration.WithLabelValues(r.Method, route, status).Observe(time.Since(started).Seconds())
		if recorder.status >= 500 {
			m.errors.WithLabelValues(r.Method, route, status).Inc()
		}
	})
}

func (m *ServiceMetrics) RegisterPGXPool(pool *pgxpool.Pool) {
	if m == nil || pool == nil {
		return
	}
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{
		"service":       m.cfg.Name,
		"environment":   m.cfg.Environment,
		"build_version": m.cfg.Version.Version,
		"commit":        m.cfg.Version.Commit,
	}, m.registry)
	registerer.MustRegister(
		poolGauge("acquired_conns", "Acquired PostgreSQL pool connections.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.AcquiredConns())
		}),
		poolGauge("idle_conns", "Idle PostgreSQL pool connections.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.IdleConns())
		}),
		poolGauge("total_conns", "Total PostgreSQL pool connections.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.TotalConns())
		}),
		poolGauge("max_conns", "Configured maximum PostgreSQL pool connections.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.MaxConns())
		}),
		poolGauge("acquire_count", "Cumulative PostgreSQL pool acquire count.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.AcquireCount())
		}),
		poolGauge("acquire_duration_seconds", "Cumulative PostgreSQL pool acquire duration in seconds.", pool, func(s *pgxpool.Stat) float64 {
			return s.AcquireDuration().Seconds()
		}),
		poolGauge("canceled_acquire_count", "Cumulative canceled PostgreSQL pool acquire count.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.CanceledAcquireCount())
		}),
		poolGauge("empty_acquire_count", "Cumulative empty PostgreSQL pool acquire count.", pool, func(s *pgxpool.Stat) float64 {
			return float64(s.EmptyAcquireCount())
		}),
	)
}

func (m *ServiceMetrics) RegisterOperationalDB(pool *pgxpool.Pool) {
	if m == nil || pool == nil {
		return
	}
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{
		"service":       m.cfg.Name,
		"environment":   m.cfg.Environment,
		"build_version": m.cfg.Version.Version,
		"commit":        m.cfg.Version.Commit,
	}, m.registry)
	registerer.MustRegister(
		dbGauge("devices_stale_total", "Trusted devices whose heartbeat is stale.", m, pool, `
SELECT count(*)
FROM devices
WHERE trust_state = 'trusted'
  AND last_heartbeat_at IS NOT NULL
  AND last_heartbeat_at < now() - make_interval(secs => greatest(heartbeat_interval_seconds * 3, 180))
`),
		dbGauge("devices_never_heartbeat_total", "Trusted devices that have never sent a heartbeat.", m, pool, `
SELECT count(*)
FROM devices
WHERE trust_state = 'trusted'
  AND last_heartbeat_at IS NULL
`),
		dbGauge("integration_webhooks_failed_total", "Failed integration webhook inbox records.", m, pool, `
SELECT count(*)
FROM integration_webhook_inbox
WHERE processing_state = 'failed'
`),
		dbGauge("integration_discrepancies_open_total", "Open integration reconciliation discrepancies.", m, pool, `
SELECT count(*)
FROM integration_discrepancies
WHERE resolution_state <> 'resolved'
`),
		dbGauge("analytics_rebuild_runs_failed_total", "Analytics rebuild runs that failed in the last 24 hours.", m, pool, `
SELECT count(*)
FROM analytics_rebuild_runs
WHERE status = 'failed' AND started_at > now() - interval '24 hours'
`),
		dbFloatGauge("analytics_projector_lag_seconds", "Seconds of lag between the latest applied event and now for the analytics projector.", m, pool, `
SELECT lag_seconds FROM analytics_projector_checkpoints WHERE projector_name = 'analytics-projector'
`),
		dbFloatGauge("analytics_rebuild_stuck_seconds", "Seconds since the current analytics rebuild started, if one is running.", m, pool, `
SELECT COALESCE((
    SELECT EXTRACT(EPOCH FROM (now() - rebuild_started_at))
    FROM analytics_projector_checkpoints
    WHERE projector_name = 'analytics-projector' AND rebuild_status = 'running'
), 0)
`),
	)
}

func (m *ServiceMetrics) RegisterRealtimeHub(hub *realtime.Hub) {
	if m == nil || hub == nil {
		return
	}
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{
		"service":       m.cfg.Name,
		"environment":   m.cfg.Environment,
		"build_version": m.cfg.Version.Version,
		"commit":        m.cfg.Version.Commit,
	}, m.registry)
	registerer.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "seatd",
			Name:      "realtime_active_connections",
			Help:      "Current active realtime WebSocket connections.",
		}, func() float64 {
			return float64(hub.MetricsSnapshot().ActiveConnections)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "realtime_connections_accepted_total",
			Help:      "Realtime WebSocket connections accepted.",
		}, func() float64 {
			return float64(hub.MetricsSnapshot().ConnectionsAccepted)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "realtime_connections_closed_total",
			Help:      "Realtime WebSocket connections closed.",
		}, func() float64 {
			return float64(hub.MetricsSnapshot().ConnectionsClosed)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "realtime_messages_published_total",
			Help:      "Realtime messages published.",
		}, func() float64 {
			return float64(hub.MetricsSnapshot().MessagesPublished)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "realtime_messages_dropped_total",
			Help:      "Realtime messages dropped.",
		}, func() float64 {
			return float64(hub.MetricsSnapshot().MessagesDropped)
		}),
		prometheus.NewCounterFunc(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "realtime_backpressure_closes_total",
			Help:      "Realtime connections closed because client send buffers were full.",
		}, func() float64 {
			return float64(hub.MetricsSnapshot().BackpressureCloses)
		}),
	)
}

func (m *ServiceMetrics) ObserveGuestAbuse(decision string) {
	if m != nil {
		m.guestAbuse.WithLabelValues(cleanLabel(decision)).Inc()
	}
}

func (m *ServiceMetrics) ObserveDeviceHeartbeat(result, deviceType, platform string) {
	if m != nil {
		m.deviceHeartbeats.WithLabelValues(cleanLabel(result), cleanLabel(deviceType), cleanLabel(platform)).Inc()
	}
}

func (m *ServiceMetrics) ObserveIntegrationWebhook(vendor, outcome string) {
	if m != nil {
		m.webhooks.WithLabelValues(cleanLabel(vendor), cleanLabel(outcome)).Inc()
	}
}

func (m *ServiceMetrics) ObserveIntegrationReconciliation(outcome string, _ string, discrepancies int32) {
	if m == nil {
		return
	}
	m.reconciliations.WithLabelValues(cleanLabel(outcome)).Inc()
	m.reconcileDiscrep.WithLabelValues().Set(float64(discrepancies))
}

func (m *ServiceMetrics) ObserveAnalyticsRebuild(outcome string) {
	if m != nil {
		m.analyticsRebuild.WithLabelValues(cleanLabel(outcome)).Inc()
	}
}

func (m *ServiceMetrics) NewOutboxMetrics() *OutboxMetrics {
	if m == nil {
		return nil
	}
	out := &OutboxMetrics{
		claimed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "outbox_records_claimed_total",
			Help:      "Outbox records claimed for processing.",
		}, []string{"destination", "topic"}),
		processed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "outbox_records_processed_total",
			Help:      "Outbox records processed successfully.",
		}, []string{"destination", "topic"}),
		retried: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "outbox_records_retried_total",
			Help:      "Outbox records scheduled for retry.",
		}, []string{"destination", "topic"}),
		failed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "outbox_records_failed_total",
			Help:      "Outbox records permanently failed.",
		}, []string{"destination", "topic"}),
		consumerFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "seatd",
			Name:      "outbox_consumer_failures_total",
			Help:      "Outbox consumer failures.",
		}, []string{"consumer"}),
	}
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{
		"service":       m.cfg.Name,
		"environment":   m.cfg.Environment,
		"build_version": m.cfg.Version.Version,
		"commit":        m.cfg.Version.Commit,
	}, m.registry)
	registerer.MustRegister(out.claimed, out.processed, out.retried, out.failed, out.consumerFailed)
	m.outbox = out
	return out
}

func (m *ServiceMetrics) RegisterOutboxStats(fn func(context.Context) (pending int64, oldestAge time.Duration, err error)) {
	if m == nil || fn == nil {
		return
	}
	registerer := prometheus.WrapRegistererWith(prometheus.Labels{
		"service":       m.cfg.Name,
		"environment":   m.cfg.Environment,
		"build_version": m.cfg.Version.Version,
		"commit":        m.cfg.Version.Commit,
	}, m.registry)
	registerer.MustRegister(
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "seatd",
			Name:      "outbox_pending_records",
			Help:      "Current pending outbox records.",
		}, func() float64 {
			pending, _, err := fn(context.Background())
			if err != nil {
				m.logger.Warn("collecting outbox pending metric failed", "error", err)
				return 0
			}
			return float64(pending)
		}),
		prometheus.NewGaugeFunc(prometheus.GaugeOpts{
			Namespace: "seatd",
			Name:      "outbox_oldest_pending_age_seconds",
			Help:      "Age of the oldest pending outbox record in seconds.",
		}, func() float64 {
			_, age, err := fn(context.Background())
			if err != nil {
				m.logger.Warn("collecting outbox oldest pending age metric failed", "error", err)
				return 0
			}
			return age.Seconds()
		}),
	)
}

type OutboxMetrics struct {
	claimed        *prometheus.CounterVec
	processed      *prometheus.CounterVec
	retried        *prometheus.CounterVec
	failed         *prometheus.CounterVec
	consumerFailed *prometheus.CounterVec
}

func (m *OutboxMetrics) RecordClaimed(destination, topic string) {
	if m != nil {
		m.claimed.WithLabelValues(cleanLabel(destination), cleanLabel(topic)).Inc()
	}
}

func (m *OutboxMetrics) RecordProcessed(destination, topic string) {
	if m != nil {
		m.processed.WithLabelValues(cleanLabel(destination), cleanLabel(topic)).Inc()
	}
}

func (m *OutboxMetrics) RecordRetried(destination, topic string) {
	if m != nil {
		m.retried.WithLabelValues(cleanLabel(destination), cleanLabel(topic)).Inc()
	}
}

func (m *OutboxMetrics) RecordFailed(destination, topic string) {
	if m != nil {
		m.failed.WithLabelValues(cleanLabel(destination), cleanLabel(topic)).Inc()
	}
}

func (m *OutboxMetrics) RecordConsumerFailed(consumer string) {
	if m != nil {
		m.consumerFailed.WithLabelValues(cleanLabel(consumer)).Inc()
	}
}

func routePattern(r *http.Request) string {
	if r.Pattern != "" {
		return r.Pattern
	}
	if r.URL != nil && r.URL.Path != "" {
		return "unmatched"
	}
	return "unknown"
}

func poolGauge(name, help string, pool *pgxpool.Pool, value func(*pgxpool.Stat) float64) prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "seatd",
		Subsystem: "db_pool",
		Name:      name,
		Help:      help,
	}, func() float64 {
		return value(pool.Stat())
	})
}

func dbGauge(name, help string, m *ServiceMetrics, pool *pgxpool.Pool, sql string) prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "seatd",
		Name:      name,
		Help:      help,
	}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		tx, err := pool.Begin(ctx)
		if err != nil {
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
			_ = tx.Rollback(ctx)
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		var count int64
		if err := tx.QueryRow(ctx, sql).Scan(&count); err != nil {
			_ = tx.Rollback(ctx)
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		if err := tx.Commit(ctx); err != nil {
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		return float64(count)
	})
}

func dbFloatGauge(name, help string, m *ServiceMetrics, pool *pgxpool.Pool, sql string) prometheus.Collector {
	return prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace: "seatd",
		Name:      name,
		Help:      help,
	}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		tx, err := pool.Begin(ctx)
		if err != nil {
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
			_ = tx.Rollback(ctx)
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		var value float64
		if err := tx.QueryRow(ctx, sql).Scan(&value); err != nil {
			_ = tx.Rollback(ctx)
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		if err := tx.Commit(ctx); err != nil {
			m.logger.Warn("collecting database metric failed", "metric", name, "error", err)
			return 0
		}
		return value
	})
}

func cleanLabel(value string) string {
	if value == "" {
		return "unknown"
	}
	if len(value) > 80 {
		return fmt.Sprintf("%.80s", value)
	}
	return value
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}
