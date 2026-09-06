package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/analytics"
	"github.com/kadebhug/seatd_v2/internal/domain/events"
	"github.com/kadebhug/seatd_v2/internal/domain/integrations"
	"github.com/kadebhug/seatd_v2/internal/domain/operations"
	"github.com/kadebhug/seatd_v2/internal/httpkit"
	"github.com/kadebhug/seatd_v2/internal/observability"
	"github.com/kadebhug/seatd_v2/internal/outbox"
	"github.com/kadebhug/seatd_v2/internal/store"
	"golang.org/x/sync/errgroup"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := app.LoadConfig("worker", 8081, app.BuildInfo{Version: version, Commit: commit, Date: date})
	if err != nil {
		slog.Error("configuration invalid", "error", err)
		os.Exit(1)
	}

	logger := app.NewLogger(cfg)
	if cfg.DatabaseURL == "" {
		logger.ErrorContext(ctx, "database url is required", "variable", "SEATD_DATABASE_URL")
		os.Exit(1)
	}
	pool, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.ErrorContext(ctx, "database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	metrics := observability.NewServiceMetrics(cfg, logger)
	metrics.RegisterPGXPool(pool)
	metrics.RegisterOperationalDB(pool)

	outboxConfig, err := loadOutboxConfig()
	if err != nil {
		logger.ErrorContext(ctx, "outbox configuration invalid", "error", err)
		os.Exit(1)
	}
	outboxConfig.Metrics = metrics.NewOutboxMetrics()
	worker := outbox.NewWorker(pool, logger, consumers(pool, logger), outboxConfig)
	metrics.RegisterOutboxStats(func(ctx context.Context) (int64, time.Duration, error) {
		stats, err := worker.Stats(ctx)
		return stats.PendingCount, stats.OldestPendingAge, err
	})
	integrationInterval, err := envDuration("SEATD_INTEGRATION_RECONCILE_INTERVAL", 5*time.Minute)
	if err != nil {
		logger.ErrorContext(ctx, "integration configuration invalid", "error", err)
		os.Exit(1)
	}
	integrationService := integrations.NewService(pool, operations.NewService(pool))
	logger.InfoContext(ctx, "worker starting")
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return worker.Run(groupCtx)
	})
	group.Go(func() error {
		return runIntegrationReconciler(groupCtx, logger, integrationService, integrationInterval, metrics)
	})
	group.Go(func() error {
		mux := httpkit.NewStatusMux(cfg, logger, metrics)
		return httpkit.Run(groupCtx, cfg, logger, mux, metrics)
	})
	if err := group.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.ErrorContext(ctx, "worker stopped with error", "error", err)
		os.Exit(1)
	}
	logger.InfoContext(ctx, "worker stopped")
}

func runIntegrationReconciler(ctx context.Context, logger *slog.Logger, service *integrations.Service, interval time.Duration, metrics *observability.ServiceMetrics) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		processed, err := service.ReconcileConnected(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			if metrics != nil {
				metrics.ObserveIntegrationReconciliation("failed", "", 0)
			}
			logger.WarnContext(ctx, "integration reconciliation failed", "error", err)
		} else if processed > 0 {
			if metrics != nil {
				metrics.ObserveIntegrationReconciliation("completed", "", 0)
			}
			logger.InfoContext(ctx, "integration reconciliation completed", "connections", processed)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func consumers(pool *pgxpool.Pool, logger *slog.Logger) map[string][]outbox.Consumer {
	return map[string][]outbox.Consumer{
		"analytics.operations": {
			analytics.NewProjector(pool),
		},
		"audit.operations": {
			logConsumer(logger, "audit-reporter"),
		},
	}
}

func logConsumer(logger *slog.Logger, name string) outbox.Consumer {
	return outbox.ConsumerFunc{
		ConsumerName: name,
		Fn: func(ctx context.Context, event events.Envelope) error {
			logger.InfoContext(ctx, "outbox event delivered",
				"consumer", name,
				"event_id", event.ID,
				"event_type", event.Type,
				"organisation_id", event.OrganisationID,
				"location_id", event.LocationID,
			)
			return nil
		},
	}
}

func loadOutboxConfig() (outbox.Config, error) {
	batchSize, err := envInt32("SEATD_OUTBOX_BATCH_SIZE", 25)
	if err != nil {
		return outbox.Config{}, err
	}
	concurrency, err := envInt("SEATD_OUTBOX_CONCURRENCY", 4)
	if err != nil {
		return outbox.Config{}, err
	}
	maxAttempts, err := envInt32("SEATD_OUTBOX_MAX_ATTEMPTS", 8)
	if err != nil {
		return outbox.Config{}, err
	}
	pollInterval, err := envDuration("SEATD_OUTBOX_POLL_INTERVAL", time.Second)
	if err != nil {
		return outbox.Config{}, err
	}
	backoffBase, err := envDuration("SEATD_OUTBOX_BACKOFF_BASE", time.Second)
	if err != nil {
		return outbox.Config{}, err
	}
	backoffMax, err := envDuration("SEATD_OUTBOX_BACKOFF_MAX", time.Minute)
	if err != nil {
		return outbox.Config{}, err
	}
	claimTimeout, err := envDuration("SEATD_OUTBOX_CLAIM_TIMEOUT", 5*time.Minute)
	if err != nil {
		return outbox.Config{}, err
	}
	return outbox.Config{
		BatchSize:    batchSize,
		Concurrency:  concurrency,
		MaxAttempts:  maxAttempts,
		PollInterval: pollInterval,
		BackoffBase:  backoffBase,
		BackoffMax:   backoffMax,
		ClaimTimeout: claimTimeout,
	}, nil
}

func envInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return parsed, nil
}

func envInt32(key string, fallback int32) (int32, error) {
	value, err := envInt(key, int(fallback))
	if err != nil {
		return 0, err
	}
	if value > int(^uint32(0)>>1) {
		return 0, fmt.Errorf("%s is too large", key)
	}
	return int32(value), nil
}

func envDuration(key string, fallback time.Duration) (time.Duration, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}
