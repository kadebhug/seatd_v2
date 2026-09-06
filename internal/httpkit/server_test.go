package httpkit

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadebhug/seatd_v2/internal/app"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestNewStatusMux(t *testing.T) {
	cfg := app.Config{
		Name:        "api",
		Environment: app.EnvTest,
		Version:     app.BuildInfo{Version: "dev", Commit: "test", Date: "test"},
	}
	mux := NewStatusMux(cfg, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestRequestLoggerAddsTraceIDs(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(previousProvider)

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	var logs bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logs, nil))
	ctx, span := otel.Tracer("test").Start(context.Background(), "request")
	defer span.End()

	handler := requestLogger(logger, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := logs.String()
	if !strings.Contains(body, `"trace_id"`) {
		t.Fatalf("log did not include trace_id: %s", body)
	}
	if !strings.Contains(body, `"span_id"`) {
		t.Fatalf("log did not include span_id: %s", body)
	}
}
