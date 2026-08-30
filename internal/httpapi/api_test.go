package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seatd/seatd/internal/app"
)

func TestCommandRequiresDevelopmentHeaders(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/tables/55555555-5555-5555-5555-555555555551/occupy", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
}

func TestCommandRejectsInvalidPathUUID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/tables/not-a-uuid/occupy", strings.NewReader(`{}`))
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	req.Header.Set(headerLocationID, "22222222-2222-2222-2222-222222222222")
	req.Header.Set(headerActorRef, "user:test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
}

func TestCommandRejectsMissingCommandID(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/tables/55555555-5555-5555-5555-555555555551/occupy", strings.NewReader(`{"expectedVersion":1}`))
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	req.Header.Set(headerLocationID, "22222222-2222-2222-2222-222222222222")
	req.Header.Set(headerActorRef, "user:test")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
}

func TestOwnerSnapshotRequiresActor(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/owner/snapshot", nil)
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
}

func TestDeviceHeartbeatRequiresBearerCredential(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/heartbeat", strings.NewReader(`{"appVersion":"0.1.0"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestCreateFloorRejectsUnknownBodyFields(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/floors", strings.NewReader(`{"slug":"main","name":"Main","sortOrder":0,"canvas":{},"extra":true}`))
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	req.Header.Set(headerLocationID, "22222222-2222-2222-2222-222222222222")
	req.Header.Set(headerActorRef, "user:owner-demo")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
}

func assertAPIError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, status, rec.Body.String())
	}
	var response errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != code {
		t.Fatalf("error code = %q, want %q", response.Error.Code, code)
	}
}
