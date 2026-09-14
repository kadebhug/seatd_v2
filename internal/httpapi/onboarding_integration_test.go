//go:build integration

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kadebhug/seatd_v2/internal/app"
)

func TestOwnerOnboardingRouteRemoved(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	handler := NewHandler(app.Config{Environment: app.EnvTest}, nil, pool)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/owner", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}
