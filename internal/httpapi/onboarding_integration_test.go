//go:build integration

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

func TestOwnerOnboardingCreatesTenantSkeleton(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	session := createEmptyWebSession(t, ctx, pool)
	handler := NewHandler(app.Config{}, nil, pool)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/owner", strings.NewReader(`{
		"organisationName": "Onboarded Group",
		"organisationSlug": "onboarded-group",
		"locationName": "Main Room",
		"locationSlug": "main-room",
		"timezone": "Africa/Johannesburg",
		"floor": {
			"name": "Main floor",
			"slug": "main-floor",
			"canvas": { "width": 1200, "height": 800, "unit": "px" },
			"zones": [{ "name": "Dining room", "sortOrder": 0 }],
			"tables": [{
				"label": "1",
				"capacityLabel": "4",
				"shape": "rectangle",
				"zoneName": "Dining room",
				"geometry": { "x": 120, "y": 120, "width": 120, "height": 84 }
			}]
		},
		"servicePeriods": [{
			"name": "Dinner",
			"daysOfWeek": [1, 2, 3, 4, 5, 6],
			"startTime": "17:00",
			"endTime": "22:00"
		}],
		"staff": [{ "name": "Sam Server", "email": "sam@example.test", "role": "waiter" }]
	}`))
	req.Header.Set("Authorization", "Bearer "+session.Secret)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var response ownerOnboardingResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Organisation.ID == "" || response.Location.ID == "" {
		t.Fatalf("response missing tenant ids: %#v", response)
	}
	if len(response.Session.Memberships) != 1 || response.Session.Memberships[0].Role != identity.RoleOrganisationOwner {
		t.Fatalf("memberships = %#v, want organisation owner", response.Session.Memberships)
	}
	assertOnboardingRows(t, ctx, pool, uuid.MustParse(response.Organisation.ID), uuid.MustParse(response.Location.ID))
}

func TestOwnerOnboardingRejectsExistingMembership(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	session := createEmptyWebSession(t, ctx, pool)
	handler := NewHandler(app.Config{}, nil, pool)
	body := []byte(`{
		"organisationName": "First Group",
		"organisationSlug": "first-group",
		"locationName": "Main",
		"locationSlug": "main",
		"timezone": "UTC"
	}`)

	first := httptest.NewRequest(http.MethodPost, "/v1/onboarding/owner", bytes.NewReader(body))
	first.Header.Set("Authorization", "Bearer "+session.Secret)
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("first status = %d, want %d; body: %s", firstRec.Code, http.StatusCreated, firstRec.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/v1/onboarding/owner", bytes.NewReader(body))
	second.Header.Set("Authorization", "Bearer "+session.Secret)
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)

	assertAPIError(t, secondRec, http.StatusConflict, "already_exists")
}

func createEmptyWebSession(t *testing.T, ctx context.Context, pool *pgxpool.Pool) identity.WebSession {
	t.Helper()

	service := identity.NewService(pool)
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	profile, err := service.ResolveExternalIdentity(ctx, identity.ResolveExternalIdentityParams{
		DisplayName: "Owner " + suffix,
		Email:       "owner-" + suffix + "@example.test",
		External: identity.ExternalIdentity{
			Issuer:  "https://issuer.example.test",
			Subject: "owner-" + suffix,
		},
	})
	if err != nil {
		t.Fatalf("create user profile: %v", err)
	}
	session, err := service.CreateWebSession(ctx, identity.CreateWebSessionParams{
		UserProfileID: profile.User.ID,
		TTL:           time.Hour,
	})
	if err != nil {
		t.Fatalf("create web session: %v", err)
	}
	session.User = profile.User
	return session
}

func assertOnboardingRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, organisationID, locationID uuid.UUID) {
	t.Helper()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin assertion transaction: %v", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		t.Fatalf("set admin context: %v", err)
	}
	q := db.New(tx)
	if _, err := q.GetOrganisation(ctx, organisationID); err != nil {
		t.Fatalf("get organisation: %v", err)
	}
	if _, err := q.GetLocation(ctx, db.GetLocationParams{ID: locationID, OrganisationID: organisationID}); err != nil {
		t.Fatalf("get location: %v", err)
	}
	floors, err := q.ListAllFloorsByLocation(ctx, db.ListAllFloorsByLocationParams{
		OrganisationID: organisationID,
		LocationID:     locationID,
	})
	if err != nil {
		t.Fatalf("list floors: %v", err)
	}
	if len(floors) != 1 {
		t.Fatalf("floor count = %d, want 1", len(floors))
	}
	tables, err := q.ListTableStatesByLocationIncludingArchived(ctx, db.ListTableStatesByLocationIncludingArchivedParams{
		OrganisationID: organisationID,
		LocationID:     locationID,
	})
	if err != nil {
		t.Fatalf("list tables: %v", err)
	}
	if len(tables) != 1 || tables[0].TableOccupancy.TableID != tables[0].Table.ID {
		t.Fatalf("table states = %#v, want table with occupancy", tables)
	}
	periods, err := q.ListServicePeriodsByLocation(ctx, db.ListServicePeriodsByLocationParams{
		OrganisationID: organisationID,
		LocationID:     locationID,
	})
	if err != nil {
		t.Fatalf("list service periods: %v", err)
	}
	if len(periods) != 1 {
		t.Fatalf("service period count = %d, want 1", len(periods))
	}
}
