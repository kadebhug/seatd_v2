package httpkit

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/observability"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Status struct {
	Service     string        `json:"service"`
	Environment string        `json:"environment"`
	Status      string        `json:"status"`
	Version     app.BuildInfo `json:"version"`
}

func NewStatusMux(cfg app.Config, logger *slog.Logger, metrics ...*observability.ServiceMetrics) *http.ServeMux {
	mux := http.NewServeMux()
	status := Status{
		Service:     cfg.Name,
		Environment: cfg.Environment,
		Status:      "ok",
		Version:     cfg.Version,
	}

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(r.Context(), logger, w, status)
	})
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(r.Context(), logger, w, cfg.Version)
	})
	if len(metrics) > 0 && metrics[0] != nil {
		mux.Handle("GET /metrics", metrics[0].Handler())
	}

	return mux
}

func Run(ctx context.Context, cfg app.Config, logger *slog.Logger, handler http.Handler, metrics ...*observability.ServiceMetrics) error {
	if len(metrics) > 0 && metrics[0] != nil {
		handler = metrics[0].Middleware(handler)
	}
	handler = requestLogger(logger, handler)
	handler = otelhttp.NewHandler(handler, cfg.Name, otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
		if r.Pattern != "" {
			return r.Method + " " + r.Pattern
		}
		if r.URL != nil && r.URL.Path != "" {
			return r.Method + " unmatched"
		}
		return r.Method + " unknown"
	}))
	server := &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		logger.InfoContext(ctx, "http server starting", "address", cfg.Address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logger.InfoContext(ctx, "http server stopped")
		return nil
	case err := <-errc:
		return err
	}
}

func writeJSON(ctx context.Context, logger *slog.Logger, w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		logger.ErrorContext(ctx, "encoding response failed", "error", err)
	}
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		if r.Pattern != "" {
			span := trace.SpanFromContext(r.Context())
			span.SetName(r.Method + " " + r.Pattern)
			span.SetAttributes(attribute.String("http.route", r.Pattern))
		}
		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration_ms", time.Since(started).Milliseconds(),
		}
		attrs = append(attrs, observability.TraceLogAttrs(r.Context())...)
		logger.InfoContext(r.Context(), "http request", attrs...)
	})
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
