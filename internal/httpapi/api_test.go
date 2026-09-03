package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/configuration"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
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
	assertAPIErrorMessage(t, rec, "id must be a UUID")
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

func TestProductionRequestContextRejectsForgedIdentityHeaders(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{Environment: app.EnvProduction}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/tables/not-a-uuid/occupy", strings.NewReader(`{}`))
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	req.Header.Set(headerLocationID, "22222222-2222-2222-2222-222222222222")
	req.Header.Set(headerActorRef, "user:owner-demo")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
	assertAPIErrorMessage(t, rec, "client-supplied identity headers are not accepted")
}

func TestProductionRequestContextAllowsTrustedIdentityHeaders(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{
		Environment:            app.EnvProduction,
		InternalAPISecret:      "expected-secret",
		TrustedIdentityHeaders: true,
	}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/tables/not-a-uuid/occupy", strings.NewReader(`{}`))
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	req.Header.Set(headerLocationID, "22222222-2222-2222-2222-222222222222")
	req.Header.Set(headerActorRef, "user:owner-demo")
	req.Header.Set(headerInternalSecret, "expected-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
	assertAPIErrorMessage(t, rec, "id must be a UUID")
}

func TestProductionTrustedIdentityHeadersRequireInternalSecret(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{
		Environment:            app.EnvProduction,
		InternalAPISecret:      "expected-secret",
		TrustedIdentityHeaders: true,
	}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/owner/snapshot", nil)
	req.Header.Set(headerOrganisationID, "11111111-1111-1111-1111-111111111111")
	req.Header.Set(headerLocationID, "22222222-2222-2222-2222-222222222222")
	req.Header.Set(headerActorRef, "user:owner-demo")
	req.Header.Set(headerInternalSecret, "wrong-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestSessionAllowsScopeRequiresOrganisationMembershipForOrganisationScope(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	session := identity.WebSession{
		Memberships: []identity.Membership{
			{
				OrganisationID: orgID,
				LocationID:     locationID,
				Role:           identity.RoleLocationManager,
			},
		},
	}

	if sessionAllowsScope(session, orgID, uuid.Nil) {
		t.Fatal("sessionAllowsScope() = true for organisation scope with only a location membership")
	}
	if !sessionAllowsScope(session, orgID, locationID) {
		t.Fatal("sessionAllowsScope() = false for matching location membership")
	}
}

func TestSessionPermissionsExcludeLocationRolesFromOrganisationScope(t *testing.T) {
	t.Parallel()

	orgID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	session := identity.WebSession{
		Memberships: []identity.Membership{
			{
				OrganisationID: orgID,
				LocationID:     locationID,
				Role:           identity.RoleLocationManager,
			},
		},
	}

	if hasPermission(sessionPermissions(session, orgID, uuid.Nil), identity.PermissionDeviceManage) {
		t.Fatal("organisation-scope permissions include a location-scoped role")
	}
	if !hasPermission(sessionPermissions(session, orgID, locationID), identity.PermissionDeviceManage) {
		t.Fatal("location-scope permissions exclude the matching location role")
	}
}

func TestRequireRequestPermissionAllowsResolvedPermission(t *testing.T) {
	t.Parallel()

	err := requireRequestPermission(t.Context(), nil, requestContext{
		Permissions: []string{identity.PermissionOperationsRead},
	}, tenantAuthzLocation, identity.PermissionOperationsRead)
	if err != nil {
		t.Fatalf("requireRequestPermission() error = %v, want nil", err)
	}
}

func TestRequireRequestPermissionRejectsMissingActor(t *testing.T) {
	t.Parallel()

	err := requireRequestPermission(t.Context(), nil, requestContext{
		Permissions: []string{identity.PermissionLayoutRead},
	}, tenantAuthzLocation, identity.PermissionOperationsRead)
	if !errors.Is(err, errForbidden) {
		t.Fatalf("requireRequestPermission() error = %v, want errForbidden", err)
	}
}

func TestRoutePermissionForTenantReadSurfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{
			name:   "membership list",
			method: http.MethodGet,
			path:   "/v1/memberships",
			want:   identity.PermissionOrganisationManage,
		},
		{
			name:   "platform tenant search",
			method: http.MethodGet,
			path:   "/v1/platform/tenants",
			want:   identity.PermissionPlatformAdmin,
		},
		{
			name:   "platform tenant read",
			method: http.MethodGet,
			path:   "/v1/platform/tenants/11111111-1111-1111-1111-111111111111",
			want:   identity.PermissionPlatformAdmin,
		},
		{
			name:   "device list",
			method: http.MethodGet,
			path:   "/v1/devices",
			want:   identity.PermissionDeviceManage,
		},
		{
			name:   "location state",
			method: http.MethodGet,
			path:   "/v1/location-state",
			want:   identity.PermissionOperationsRead,
		},
		{
			name:   "active assists",
			method: http.MethodGet,
			path:   "/v1/assists",
			want:   identity.PermissionOperationsRead,
		},
		{
			name:   "audit timeline",
			method: http.MethodGet,
			path:   "/v1/audit/timeline",
			want:   identity.PermissionAuditRead,
		},
		{
			name:   "floor list",
			method: http.MethodGet,
			path:   "/v1/floors",
			want:   identity.PermissionLayoutRead,
		},
		{
			name:   "table detail",
			method: http.MethodGet,
			path:   "/v1/tables/55555555-5555-5555-5555-555555555551",
			want:   identity.PermissionLayoutRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if got := routePermission(req); got != tt.want {
				t.Fatalf("routePermission() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProductionOIDCSessionCreationRequiresInternalSecret(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{Environment: app.EnvProduction, InternalAPISecret: "expected-secret"}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/oidc/session", strings.NewReader(`{}`))
	req.Header.Set(headerInternalSecret, "wrong-secret")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestDeviceHeartbeatRequiresBearerCredential(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/devices/heartbeat", strings.NewReader(`{"appVersion":"0.1.0"}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusUnauthorized, "unauthorized")
}

func TestIntegrationWebhookRequiresConnectionHeader(t *testing.T) {
	t.Parallel()

	handler := NewHandler(app.Config{}, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/v1/integrations/reference_pos/webhooks", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
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

func TestWriteConfigurationErrorDuplicateZoneName(t *testing.T) {
	t.Parallel()

	api := &API{}
	rec := httptest.NewRecorder()
	api.writeConfigurationError(rec, configuration.ConflictError{
		Constraint: "zones_floor_id_name_key",
	})

	assertAPIError(t, rec, http.StatusConflict, "already_exists")
	var response errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	want := "a zone with this name already exists on this floor"
	if response.Error.Message != want {
		t.Fatalf("message = %q, want %q", response.Error.Message, want)
	}
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

func assertAPIErrorMessage(t *testing.T, rec *httptest.ResponseRecorder, message string) {
	t.Helper()
	var response errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Message != message {
		t.Fatalf("error message = %q, want %q", response.Error.Message, message)
	}
}
