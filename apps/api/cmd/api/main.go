package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/httpapi"
	"github.com/kadebhug/seatd_v2/internal/httpkit"
	"github.com/kadebhug/seatd_v2/internal/observability"
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

	cfg, err := app.LoadConfig("api", 8080, app.BuildInfo{Version: version, Commit: commit, Date: date})
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

	mux := httpkit.NewStatusMux(cfg, logger, metrics)
	mux.Handle("/v1/", httpapi.NewHandler(cfg, logger, pool, metrics))
	if err := httpkit.Run(ctx, cfg, logger, mux, metrics); err != nil {
		logger.ErrorContext(ctx, "api stopped with error", "error", err)
		os.Exit(1)
	}
}
