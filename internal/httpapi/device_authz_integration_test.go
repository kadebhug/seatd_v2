//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

func TestDeviceManagementAuthorizesActorPermissions(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createDeviceAuthzFixture(t, ctx, pool)
	handler := NewHandler(app.Config{Environment: app.EnvTest}, nil, pool)

	t.Run("actor without permission", func(t *testing.T) {
		assertDeviceRequestForbidden(t, handler, http.MethodGet, "/v1/devices", fixture, fixture.Location.ID, fixture.WaiterRef, "")

		body := `{"locationId":"` + fixture.Location.ID.String() + `","deviceType":"display"}`
		assertDeviceRequestForbidden(t, handler, http.MethodPost, "/v1/devices/pairing-codes", fixture, fixture.Location.ID, fixture.WaiterRef, body)

		device := createDeviceForAuthzTest(t, ctx, pool, fixture, fixture.Location.ID)
		assertDeviceRequestForbidden(t, handler, http.MethodPost, "/v1/devices/"+device.ID.String()+"/revoke", fixture, fixture.Location.ID, fixture.WaiterRef, "")
	})

	t.Run("actor with location permission", func(t *testing.T) {
		assertDeviceRequestOK(t, handler, http.MethodGet, "/v1/devices", fixture, fixture.Location.ID, fixture.ManagerRef, "")

		body := `{"locationId":"` + fixture.Location.ID.String() + `","deviceType":"display"}`
		assertDeviceRequestStatus(t, handler, http.MethodPost, "/v1/devices/pairing-codes", fixture, fixture.Location.ID, fixture.ManagerRef, body, http.StatusCreated)

		device := createDeviceForAuthzTest(t, ctx, pool, fixture, fixture.Location.ID)
		assertDeviceRequestOK(t, handler, http.MethodPost, "/v1/devices/"+device.ID.String()+"/revoke", fixture, fixture.Location.ID, fixture.ManagerRef, "")
	})

	t.Run("actor with organisation permission", func(t *testing.T) {
		assertDeviceRequestOK(t, handler, http.MethodGet, "/v1/devices", fixture, uuid.Nil, fixture.OwnerRef, "")

		body := `{"locationId":"` + fixture.OtherLocation.ID.String() + `","deviceType":"display"}`
		assertDeviceRequestStatus(t, handler, http.MethodPost, "/v1/devices/pairing-codes", fixture, uuid.Nil, fixture.OwnerRef, body, http.StatusCreated)

		device := createDeviceForAuthzTest(t, ctx, pool, fixture, fixture.OtherLocation.ID)
		assertDeviceRequestOK(t, handler, http.MethodPost, "/v1/devices/"+device.ID.String()+"/revoke", fixture, uuid.Nil, fixture.OwnerRef, "")
	})

	t.Run("cross location attempt", func(t *testing.T) {
		device := createDeviceForAuthzTest(t, ctx, pool, fixture, fixture.OtherLocation.ID)
		assertDeviceRequestForbidden(t, handler, http.MethodPost, "/v1/devices/"+device.ID.String()+"/revoke", fixture, fixture.Location.ID, fixture.ManagerRef, "")

		body := `{"locationId":"` + fixture.OtherLocation.ID.String() + `","deviceType":"display"}`
		assertDeviceRequestForbidden(t, handler, http.MethodPost, "/v1/devices/pairing-codes", fixture, fixture.Location.ID, fixture.ManagerRef, body)
	})
}

type deviceAuthzFixture struct {
	Organisation  db.Organisation
	Location      db.Location
	OtherLocation db.Location
	OwnerRef      string
	ManagerRef    string
	WaiterRef     string
}

func setupHTTPAPIDatabase(t *testing.T, ctx context.Context) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("SEATD_INTEGRATION_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set SEATD_INTEGRATION_DATABASE_URL to run integration tests")
	}

	pool, err := store.OpenPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open database pool: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"); err != nil {
		t.Fatalf("reset schema: %v", err)
	}
	for _, path := range httpAPIMigrationPaths(t) {
		sql, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read migration %s: %v", path, err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatalf("apply migration %s: %v", path, err)
		}
	}

	return pool
}

func httpAPIMigrationPaths(t *testing.T) []string {
	t.Helper()

	root := httpAPIRepoRoot(t)
	entries, err := filepath.Glob(filepath.Join(root, "infra/database/migrations/*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	return entries
}

func httpAPIRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repository root from %s", dir)
		}
		dir = parent
	}
}

func createDeviceAuthzFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) deviceAuthzFixture {
	t.Helper()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	ownerRef := "user:owner-" + suffix
	managerRef := "user:manager-" + suffix
	waiterRef := "user:waiter-" + suffix

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin fixture transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Fatalf("rollback fixture transaction: %v", err)
		}
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		t.Fatalf("set admin context: %v", err)
	}
	q := db.New(tx)
	org, err := q.CreateOrganisation(ctx, db.CreateOrganisationParams{
		Slug:               "device-authz-" + suffix,
		Name:               "Device Authz",
		LegacyRestaurantID: pgtype.Text{String: "legacy-device-authz-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	location, err := q.CreateLocation(ctx, db.CreateLocationParams{
		OrganisationID:     org.ID,
		Slug:               "main-" + suffix,
		Name:               "Main",
		Timezone:           "Africa/Johannesburg",
		LegacyRestaurantID: pgtype.Text{String: "legacy-main-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create location: %v", err)
	}
	otherLocation, err := q.CreateLocation(ctx, db.CreateLocationParams{
		OrganisationID:     org.ID,
		Slug:               "annex-" + suffix,
		Name:               "Annex",
		Timezone:           "Africa/Johannesburg",
		LegacyRestaurantID: pgtype.Text{String: "legacy-annex-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create other location: %v", err)
	}
	if _, err := q.CreateOrganisationMembership(ctx, db.CreateOrganisationMembershipParams{
		OrganisationID: org.ID,
		MemberRef:      ownerRef,
		Role:           identity.RoleOrganisationOwner,
	}); err != nil {
		t.Fatalf("create owner membership: %v", err)
	}
	if _, err := q.CreateLocationMembership(ctx, db.CreateLocationMembershipParams{
		OrganisationID: org.ID,
		LocationID:     location.ID,
		MemberRef:      managerRef,
		Role:           identity.RoleLocationManager,
	}); err != nil {
		t.Fatalf("create manager membership: %v", err)
	}
	if _, err := q.CreateLocationMembership(ctx, db.CreateLocationMembershipParams{
		OrganisationID: org.ID,
		LocationID:     location.ID,
		MemberRef:      waiterRef,
		Role:           identity.RoleWaiter,
	}); err != nil {
		t.Fatalf("create waiter membership: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit fixture transaction: %v", err)
	}

	return deviceAuthzFixture{
		Organisation:  org,
		Location:      location,
		OtherLocation: otherLocation,
		OwnerRef:      ownerRef,
		ManagerRef:    managerRef,
		WaiterRef:     waiterRef,
	}
}

func createDeviceForAuthzTest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fixture deviceAuthzFixture, locationID uuid.UUID) db.Device {
	t.Helper()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin device transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Fatalf("rollback device transaction: %v", err)
		}
	}()
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		t.Fatalf("set admin context: %v", err)
	}
	device, err := db.New(tx).RegisterDevice(ctx, db.RegisterDeviceParams{
		OrganisationID:           fixture.Organisation.ID,
		LocationID:               uuid.NullUUID{UUID: locationID, Valid: true},
		DeviceType:               identity.DeviceTypeDisplay,
		Platform:                 "test",
		AppVersion:               "0.1.0",
		Name:                     pgtype.Text{String: "Display " + locationID.String()[:8], Valid: true},
		Capabilities:             []byte(`{}`),
		Configuration:            []byte(`{}`),
		HeartbeatIntervalSeconds: 60,
	})
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit device transaction: %v", err)
	}
	return device
}

func assertDeviceRequestOK(t *testing.T, handler http.Handler, method string, path string, fixture deviceAuthzFixture, locationID uuid.UUID, actorRef string, body string) {
	t.Helper()
	assertDeviceRequestStatus(t, handler, method, path, fixture, locationID, actorRef, body, http.StatusOK)
}

func assertDeviceRequestForbidden(t *testing.T, handler http.Handler, method string, path string, fixture deviceAuthzFixture, locationID uuid.UUID, actorRef string, body string) {
	t.Helper()
	rec := assertDeviceRequestStatus(t, handler, method, path, fixture, locationID, actorRef, body, http.StatusForbidden)
	var response errorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Error.Code != "forbidden" {
		t.Fatalf("error code = %q, want forbidden; body: %s", response.Error.Code, rec.Body.String())
	}
}

func assertDeviceRequestStatus(t *testing.T, handler http.Handler, method string, path string, fixture deviceAuthzFixture, locationID uuid.UUID, actorRef string, body string, status int) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader(`{}`)
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set(headerOrganisationID, fixture.Organisation.ID.String())
	if locationID != uuid.Nil {
		req.Header.Set(headerLocationID, locationID.String())
	}
	req.Header.Set(headerActorRef, actorRef)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != status {
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, path, rec.Code, status, rec.Body.String())
	}
	return rec
}
