package httpkit

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seatd/seatd/internal/app"
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
