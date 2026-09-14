//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

func TestPlatformAdminRoutesAuthorizeAndAudit(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createPlatformFixture(t, ctx, pool)
	handler := NewHandler(app.Config{Environment: app.EnvTest}, nil, pool)

	t.Run("non admin forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/platform/tenants", nil)
		req.Header.Set(headerActorRef, fixture.OwnerRef)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assertAPIError(t, rec, http.StatusForbidden, "forbidden")
	})

	t.Run("admin search audits", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/platform/tenants?q=Platform&limit=10", nil)
		req.Header.Set(headerActorRef, fixture.PlatformRef)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response struct {
			Tenants []platformTenantSummaryDTO `json:"tenants"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(response.Tenants) != 1 || response.Tenants[0].ID != fixture.Organisation.ID.String() {
			t.Fatalf("tenants = %#v, want fixture organisation", response.Tenants)
		}
		assertPlatformAuditCount(t, ctx, pool, uuid.Nil, fixture.PlatformRef, platformActionTenantSearch, 1)
	})

	t.Run("admin detail audits", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/platform/tenants/"+fixture.Organisation.ID.String(), nil)
		req.Header.Set(headerActorRef, fixture.PlatformRef)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		var response platformTenantDetailDTO
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if response.Tenant.ID != fixture.Organisation.ID.String() {
			t.Fatalf("tenant id = %q, want %q", response.Tenant.ID, fixture.Organisation.ID.String())
		}
		if response.Diagnostics.DisabledLocationCount != 1 {
			t.Fatalf("disabled location count = %d, want 1", response.Diagnostics.DisabledLocationCount)
		}
		assertPlatformAuditCount(t, ctx, pool, fixture.Organisation.ID, fixture.PlatformRef, platformActionTenantRead, 1)
	})
}

func TestPlatformAdminSuspendReactivate(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createPlatformFixture(t, ctx, pool)
	handler := NewHandler(app.Config{Environment: app.EnvTest}, nil, pool)

	suspendReq := func(actorRef, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/"+fixture.Organisation.ID.String()+"/suspend", strings.NewReader(body))
		req.Header.Set(headerActorRef, actorRef)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}
	reactivateReq := func(actorRef, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "/v1/platform/tenants/"+fixture.Organisation.ID.String()+"/reactivate", strings.NewReader(body))
		req.Header.Set(headerActorRef, actorRef)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec
	}

	t.Run("non admin forbidden", func(t *testing.T) {
		rec := suspendReq(fixture.OwnerRef, `{"reason":"abuse"}`)
		assertAPIError(t, rec, http.StatusForbidden, "forbidden")
	})

	t.Run("platform admin without write permission forbidden", func(t *testing.T) {
		rec := suspendReq(fixture.ReadOnlyAdminRef, `{"reason":"abuse"}`)
		assertAPIError(t, rec, http.StatusForbidden, "forbidden")
	})

	t.Run("missing reason rejected", func(t *testing.T) {
		rec := suspendReq(fixture.PlatformRef, `{}`)
		assertAPIError(t, rec, http.StatusBadRequest, "validation_failed")
	})

	t.Run("platform admin suspends and reactivates with audit", func(t *testing.T) {
		suspendRec := suspendReq(fixture.PlatformRef, `{"reason":"suspicious billing activity"}`)
		if suspendRec.Code != http.StatusOK {
			t.Fatalf("suspend status = %d, want %d; body: %s", suspendRec.Code, http.StatusOK, suspendRec.Body.String())
		}
		var suspended platformTenantDetailDTO
		if err := json.Unmarshal(suspendRec.Body.Bytes(), &suspended); err != nil {
			t.Fatalf("decode suspend response: %v", err)
		}
		if suspended.Tenant.Status != "disabled" {
			t.Fatalf("tenant status = %q, want %q", suspended.Tenant.Status, "disabled")
		}
		assertPlatformAuditCount(t, ctx, pool, fixture.Organisation.ID, fixture.PlatformRef, platformActionTenantSuspend, 1)

		reactivateRec := reactivateReq(fixture.PlatformRef, `{"reason":"billing dispute resolved"}`)
		if reactivateRec.Code != http.StatusOK {
			t.Fatalf("reactivate status = %d, want %d; body: %s", reactivateRec.Code, http.StatusOK, reactivateRec.Body.String())
		}
		var reactivated platformTenantDetailDTO
		if err := json.Unmarshal(reactivateRec.Body.Bytes(), &reactivated); err != nil {
			t.Fatalf("decode reactivate response: %v", err)
		}
		if reactivated.Tenant.Status != "active" {
			t.Fatalf("tenant status = %q, want %q", reactivated.Tenant.Status, "active")
		}
		assertPlatformAuditCount(t, ctx, pool, fixture.Organisation.ID, fixture.PlatformRef, platformActionTenantReactivate, 1)
	})
}

type platformFixture struct {
	Organisation     db.Organisation
	PlatformRef      string
	OwnerRef         string
	ReadOnlyAdminRef string
}

func createPlatformFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) platformFixture {
	t.Helper()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	platformProfileID := uuid.New()
	ownerProfileID := uuid.New()
	readOnlyAdminProfileID := uuid.New()
	platformRef := "user:" + platformProfileID.String()
	ownerRef := "user:" + ownerProfileID.String()
	readOnlyAdminRef := "user:" + readOnlyAdminProfileID.String()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin fixture transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		t.Fatalf("set admin context: %v", err)
	}
	q := db.New(tx)
	if _, err := q.CreateUserProfile(ctx, db.CreateUserProfileParams{
		DisplayName: "Platform Admin",
		Email:       pgtype.Text{String: "platform-" + suffix + "@example.test", Valid: true},
	}); err != nil {
		t.Fatalf("create platform profile: %v", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE user_profiles
SET id = $1
WHERE email = $2
`, platformProfileID, "platform-"+suffix+"@example.test"); err != nil {
		t.Fatalf("set platform profile id: %v", err)
	}
	if _, err := q.CreateUserProfile(ctx, db.CreateUserProfileParams{
		DisplayName: "Owner",
		Email:       pgtype.Text{String: "owner-" + suffix + "@example.test", Valid: true},
	}); err != nil {
		t.Fatalf("create owner profile: %v", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE user_profiles
SET id = $1
WHERE email = $2
`, ownerProfileID, "owner-"+suffix+"@example.test"); err != nil {
		t.Fatalf("set owner profile id: %v", err)
	}
	if _, err := q.CreateUserProfile(ctx, db.CreateUserProfileParams{
		DisplayName: "Read Only Admin",
		Email:       pgtype.Text{String: "platform-ro-" + suffix + "@example.test", Valid: true},
	}); err != nil {
		t.Fatalf("create read-only admin profile: %v", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE user_profiles
SET id = $1
WHERE email = $2
`, readOnlyAdminProfileID, "platform-ro-"+suffix+"@example.test"); err != nil {
		t.Fatalf("set read-only admin profile id: %v", err)
	}
	org, err := q.CreateOrganisation(ctx, db.CreateOrganisationParams{
		Slug:               "platform-" + suffix,
		Name:               "Platform Tenant",
		LegacyRestaurantID: pgtype.Text{String: "legacy-platform-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	activeLocation, err := q.CreateLocation(ctx, db.CreateLocationParams{
		OrganisationID:     org.ID,
		Slug:               "main-" + suffix,
		Name:               "Main",
		Timezone:           "Africa/Johannesburg",
		LegacyRestaurantID: pgtype.Text{String: "legacy-platform-main-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create active location: %v", err)
	}
	disabledLocation, err := q.CreateLocation(ctx, db.CreateLocationParams{
		OrganisationID:     org.ID,
		Slug:               "closed-" + suffix,
		Name:               "Closed",
		Timezone:           "Africa/Johannesburg",
		LegacyRestaurantID: pgtype.Text{String: "legacy-platform-closed-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create disabled location: %v", err)
	}
	if _, err := tx.Exec(ctx, "UPDATE locations SET status = 'disabled' WHERE id = $1", disabledLocation.ID); err != nil {
		t.Fatalf("disable location: %v", err)
	}
	if _, err := q.CreatePlatformMembership(ctx, db.CreatePlatformMembershipParams{
		UserProfileID:     platformProfileID,
		Role:              identity.RolePlatformAdmin,
		GrantedByActorRef: "test",
	}); err != nil {
		t.Fatalf("create platform membership: %v", err)
	}
	if _, err := q.CreateOrganisationMembership(ctx, db.CreateOrganisationMembershipParams{
		OrganisationID: org.ID,
		UserProfileID:  uuid.NullUUID{UUID: ownerProfileID, Valid: true},
		MemberRef:      ownerRef,
		Role:           identity.RoleOrganisationOwner,
	}); err != nil {
		t.Fatalf("create owner membership: %v", err)
	}
	if _, err := q.CreatePlatformMembership(ctx, db.CreatePlatformMembershipParams{
		UserProfileID:     readOnlyAdminProfileID,
		Role:              identity.RoleSupport,
		GrantedByActorRef: "test",
	}); err != nil {
		t.Fatalf("create read-only platform admin membership: %v", err)
	}
	if _, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
		OrganisationID:           org.ID,
		LocationID:               uuid.NullUUID{UUID: activeLocation.ID, Valid: true},
		DeviceType:               identity.DeviceTypeDisplay,
		Platform:                 "test",
		AppVersion:               "0.1.0",
		Name:                     pgtype.Text{String: "Display", Valid: true},
		Capabilities:             []byte(`{}`),
		Configuration:            []byte(`{}`),
		HeartbeatIntervalSeconds: 60,
	}); err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit fixture transaction: %v", err)
	}
	return platformFixture{
		Organisation:     org,
		PlatformRef:      platformRef,
		OwnerRef:         ownerRef,
		ReadOnlyAdminRef: readOnlyAdminRef,
	}
}

func assertPlatformAuditCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, organisationID uuid.UUID, actorRef string, action string, want int) {
	t.Helper()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin audit assertion transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		t.Fatalf("set admin context: %v", err)
	}
	var got int
	if organisationID == uuid.Nil {
		err = tx.QueryRow(ctx, `
SELECT count(*)
FROM audit_events
WHERE organisation_id IS NULL
  AND actor_ref = $1
  AND action = $2
`, actorRef, action).Scan(&got)
	} else {
		err = tx.QueryRow(ctx, `
SELECT count(*)
FROM audit_events
WHERE organisation_id = $1
  AND actor_ref = $2
  AND action = $3
`, organisationID, actorRef, action).Scan(&got)
	}
	if err != nil {
		t.Fatalf("count platform audits: %v", err)
	}
	if got != want {
		t.Fatalf("audit count = %d, want %d", got, want)
	}
}
