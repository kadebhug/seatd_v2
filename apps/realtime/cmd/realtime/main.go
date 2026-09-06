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

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/events"
	"github.com/kadebhug/seatd_v2/internal/httpkit"
	"github.com/kadebhug/seatd_v2/internal/observability"
	"github.com/kadebhug/seatd_v2/internal/outbox"
	"github.com/kadebhug/seatd_v2/internal/realtime"
	"github.com/kadebhug/seatd_v2/internal/store"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := app.LoadConfig("realtime", 8082, app.BuildInfo{Version: version, Commit: commit, Date: date})
	if err != nil {
		slog.Error("configuration invalid", "error", err)
		os.Exit(1)
	}

	logger := app.NewLogger(cfg)
	shutdownTracing, err := observability.InitTracing(ctx, cfg, logger)
	if err != nil {
		logger.ErrorContext(ctx, "tracing configuration invalid", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			logger.WarnContext(ctx, "tracing shutdown failed", "error", err)
		}
	}()
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

	wsConfig, err := loadRealtimeConfig()
	if err != nil {
		logger.ErrorContext(ctx, "realtime configuration invalid", "error", err)
		os.Exit(1)
	}
	hub := realtime.NewHub(logger)
	metrics.RegisterRealtimeHub(hub)
	worker := outbox.NewWorker(pool, logger, map[string][]outbox.Consumer{
		realtime.DestinationOperations: {
			realtimeConsumer(hub),
		},
	}, outbox.Config{
		InstanceID:   "realtime-" + cfg.Version.Commit,
		BatchSize:    50,
		Concurrency:  4,
		PollInterval: time.Second,
		Metrics:      metrics.NewOutboxMetrics(),
	})
	metrics.RegisterOutboxStats(func(ctx context.Context) (int64, time.Duration, error) {
		stats, err := worker.Stats(ctx)
		return stats.PendingCount, stats.OldestPendingAge, err
	})
	go func() {
		if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			logger.ErrorContext(ctx, "realtime outbox dispatcher stopped with error", "error", err)
		}
	}()

	mux := httpkit.NewStatusMux(cfg, logger, metrics)
	mux.Handle("/v1/", realtime.NewHandler(cfg, logger, pool, hub, wsConfig))
	if err := httpkit.Run(ctx, cfg, logger, mux, metrics); err != nil {
		logger.ErrorContext(ctx, "realtime gateway stopped with error", "error", err)
		os.Exit(1)
	}
}

func realtimeConsumer(hub *realtime.Hub) outbox.Consumer {
	return outbox.ConsumerFunc{
		ConsumerName: "realtime-gateway",
		Fn: func(ctx context.Context, event events.Envelope) error {
			hub.Publish(ctx, event)
			return nil
		},
	}
}

func loadRealtimeConfig() (realtime.WebSocketConfig, error) {
	sendBuffer, err := envInt("SEATD_REALTIME_SEND_BUFFER", 64)
	if err != nil {
		return realtime.WebSocketConfig{}, err
	}
	readLimit, err := envInt64("SEATD_REALTIME_READ_LIMIT", 4096)
	if err != nil {
		return realtime.WebSocketConfig{}, err
	}
	heartbeat, err := envDuration("SEATD_REALTIME_HEARTBEAT", 25*time.Second)
	if err != nil {
		return realtime.WebSocketConfig{}, err
	}
	connectionLimit, err := envInt("SEATD_REALTIME_CONNECTION_LIMIT", 5000)
	if err != nil {
		return realtime.WebSocketConfig{}, err
	}
	return realtime.WebSocketConfig{
		SendBuffer:      sendBuffer,
		ReadLimit:       readLimit,
		Heartbeat:       heartbeat,
		ConnectionLimit: connectionLimit,
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

func envInt64(key string, fallback int64) (int64, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return parsed, nil
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
