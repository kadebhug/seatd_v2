//go:build integration

package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/operations"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

func TestGuestQRHTTPFlowUsesRealExportURLAndReflectsStaffResolution(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createGuestQRFixture(t, ctx, pool)
	handler := NewHandler(app.Config{GuestWebOrigin: "https://guest.seatd.test"}, nil, pool)

	occupy := assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/occupy", fixture, `{"commandId":"`+uuid.NewString()+`","expectedVersion":1}`)
	var occupied tableCommandResponse
	decodeResponse(t, occupy, &occupied)

	exportRec := assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/qr-capabilities", fixture, `{"label":"front door","expiresAt":""}`)
	if exportRec.Code != http.StatusCreated {
		t.Fatalf("export status = %d, want %d; body: %s", exportRec.Code, http.StatusCreated, exportRec.Body.String())
	}
	var export qrExportDTO
	decodeResponse(t, exportRec, &export)
	if export.Token == "" {
		t.Fatalf("export token is empty")
	}
	if export.PublicURL != "https://guest.seatd.test/qr/"+export.Token {
		t.Fatalf("public URL = %q, want exported guest URL with token", export.PublicURL)
	}

	contextRec := assertPublicRequest(t, handler, http.MethodGet, "/v1/guest/qr/"+export.Token, "", http.StatusOK)
	var guest guestContextDTO
	decodeResponse(t, contextRec, &guest)
	if guest.LocationName != fixture.Location.Name || guest.TableLabel != fixture.Table.Label {
		t.Fatalf("guest context = %+v, want exported table context", guest)
	}
	if guest.Occupancy.Status != operations.TableStatusOccupied {
		t.Fatalf("guest occupancy = %q, want occupied", guest.Occupancy.Status)
	}

	commandID := uuid.NewString()
	createBody := `{"commandId":"` + commandID + `","actionKey":"call_waiter"}`
	createRec := assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", createBody, http.StatusCreated)
	var created assistDTO
	decodeResponse(t, createRec, &created)
	if created.Status != operations.AssistStatusPending || created.ActionKey == nil || *created.ActionKey != "call_waiter" {
		t.Fatalf("created assist = %+v, want pending call_waiter", created)
	}

	replayRec := assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", createBody, http.StatusCreated)
	if replayRec.Header().Get("X-Seatd-Idempotency-Replayed") != "true" {
		t.Fatalf("idempotency replay header = %q, want true", replayRec.Header().Get("X-Seatd-Idempotency-Replayed"))
	}
	var replayed assistDTO
	decodeResponse(t, replayRec, &replayed)
	if replayed.ID != created.ID {
		t.Fatalf("replayed assist id = %q, want %q", replayed.ID, created.ID)
	}

	conflictBody := `{"commandId":"` + commandID + `","actionKey":"request_bill"}`
	assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", conflictBody, http.StatusConflict)

	duplicateRec := assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", `{"commandId":"`+uuid.NewString()+`","actionKey":"call_waiter"}`, http.StatusCreated)
	var duplicate assistDTO
	decodeResponse(t, duplicateRec, &duplicate)
	if duplicate.ID != created.ID {
		t.Fatalf("duplicate active assist id = %q, want %q", duplicate.ID, created.ID)
	}

	resolveRec := assertTenantCommand(t, handler, http.MethodPost, "/v1/assists/"+created.ID+"/resolve", fixture, `{"commandId":"`+uuid.NewString()+`","expectedVersion":`+itoa32(created.Version)+`}`)
	var resolved assistDTO
	decodeResponse(t, resolveRec, &resolved)
	if resolved.Status != operations.AssistStatusResolved {
		t.Fatalf("resolved assist status = %q, want resolved", resolved.Status)
	}

	pollRec := assertPublicRequest(t, handler, http.MethodGet, "/v1/guest/qr/"+export.Token+"/requests/"+created.ID, "", http.StatusOK)
	var polled assistDTO
	decodeResponse(t, pollRec, &polled)
	if polled.Status != operations.AssistStatusResolved {
		t.Fatalf("polled assist status = %q, want resolved", polled.Status)
	}

	var lastUsedValid bool
	if err := pool.QueryRow(ctx, `SELECT last_used_at IS NOT NULL FROM table_qr_capabilities WHERE id = $1`, export.ID).Scan(&lastUsedValid); err != nil {
		t.Fatalf("read capability last_used_at: %v", err)
	}
	if !lastUsedValid {
		t.Fatalf("guest lookup did not update capability last_used_at")
	}
}

func TestGuestQRTokensFailClosedAndRecordLookupDenials(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createGuestQRFixture(t, ctx, pool)
	handler := NewHandler(app.Config{GuestWebOrigin: "https://guest.seatd.test"}, nil, pool)

	expiredRec := assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/qr-capabilities", fixture, `{"label":"expired","expiresAt":""}`)
	var expired qrExportDTO
	decodeResponse(t, expiredRec, &expired)
	_, err := pool.Exec(ctx, `
UPDATE table_qr_capabilities
SET issued_at = now() - interval '2 hours',
    expires_at = now() - interval '1 hour'
WHERE id = $1
`, expired.ID)
	if err != nil {
		t.Fatalf("expire qr capability: %v", err)
	}
	assertPublicRequest(t, handler, http.MethodGet, "/v1/guest/qr/"+expired.Token, "", http.StatusNotFound)

	exportRec := assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/qr-capabilities", fixture, `{"label":"revoked","expiresAt":""}`)
	var revoked qrExportDTO
	decodeResponse(t, exportRec, &revoked)
	assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/qr-capabilities/"+revoked.ID+"/revoke", fixture, `{}`)
	assertPublicRequest(t, handler, http.MethodGet, "/v1/guest/qr/"+revoked.Token, "", http.StatusNotFound)

	var denied int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM guest_abuse_events WHERE decision = 'lookup_denied'`).Scan(&denied); err != nil {
		t.Fatalf("count lookup denials: %v", err)
	}
	if denied != 2 {
		t.Fatalf("lookup_denied events = %d, want 2", denied)
	}
}

func TestGuestQRRateLimitsAndCooldownsRecordAbuse(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createGuestQRFixture(t, ctx, pool)
	handler := NewHandler(app.Config{}, nil, pool)

	assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/occupy", fixture, `{"commandId":"`+uuid.NewString()+`","expectedVersion":1}`)
	exportRec := assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/qr-capabilities", fixture, `{"label":"rate limit","expiresAt":""}`)
	var export qrExportDTO
	decodeResponse(t, exportRec, &export)

	var first assistDTO
	for i := 0; i < 6; i++ {
		rec := assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", `{"commandId":"`+uuid.NewString()+`","actionKey":"call_waiter"}`, http.StatusCreated)
		var current assistDTO
		decodeResponse(t, rec, &current)
		if i == 0 {
			first = current
		} else if current.ID != first.ID {
			t.Fatalf("rate-limit warmup request %d created assist %q, want duplicate %q", i, current.ID, first.ID)
		}
	}
	assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", `{"commandId":"`+uuid.NewString()+`","actionKey":"call_waiter"}`, http.StatusTooManyRequests)
	assertGuestAbuseDecisionCount(t, pool, "rate_limited", 1)

	cooldownHandler := NewHandler(app.Config{}, nil, pool)
	resolveRec := assertTenantCommand(t, cooldownHandler, http.MethodPost, "/v1/assists/"+first.ID+"/resolve", fixture, `{"commandId":"`+uuid.NewString()+`","expectedVersion":`+itoa32(first.Version)+`}`)
	var resolved assistDTO
	decodeResponse(t, resolveRec, &resolved)
	if resolved.Status != operations.AssistStatusResolved {
		t.Fatalf("resolved status = %q, want resolved", resolved.Status)
	}
	assertPublicRequest(t, cooldownHandler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", `{"commandId":"`+uuid.NewString()+`","actionKey":"call_waiter"}`, http.StatusTooManyRequests)
	assertGuestAbuseDecisionCount(t, pool, "cooldown", 1)
}

func TestGuestCanCancelPendingRequest(t *testing.T) {
	ctx := context.Background()
	pool := setupHTTPAPIDatabase(t, ctx)
	fixture := createGuestQRFixture(t, ctx, pool)
	handler := NewHandler(app.Config{}, nil, pool)

	assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/occupy", fixture, `{"commandId":"`+uuid.NewString()+`","expectedVersion":1}`)
	exportRec := assertTenantCommand(t, handler, http.MethodPost, "/v1/tables/"+fixture.Table.ID.String()+"/qr-capabilities", fixture, `{"label":"cancel","expiresAt":""}`)
	var export qrExportDTO
	decodeResponse(t, exportRec, &export)
	createRec := assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests", `{"commandId":"`+uuid.NewString()+`","actionKey":"request_bill"}`, http.StatusCreated)
	var created assistDTO
	decodeResponse(t, createRec, &created)

	cancelRec := assertPublicRequest(t, handler, http.MethodPost, "/v1/guest/qr/"+export.Token+"/requests/"+created.ID+"/cancel", `{"commandId":"`+uuid.NewString()+`"}`, http.StatusOK)
	var cancelled assistDTO
	decodeResponse(t, cancelRec, &cancelled)
	if cancelled.Status != operations.AssistStatusCancelled {
		t.Fatalf("cancelled status = %q, want cancelled", cancelled.Status)
	}

	contextRec := assertPublicRequest(t, handler, http.MethodGet, "/v1/guest/qr/"+export.Token, "", http.StatusOK)
	var guest guestContextDTO
	decodeResponse(t, contextRec, &guest)
	if guest.ActiveRequest != nil {
		t.Fatalf("guest context active request = %+v, want none after cancellation", guest.ActiveRequest)
	}
}

type guestQRFixture struct {
	Organisation db.Organisation
	Location     db.Location
	Floor        db.Floor
	Zone         db.Zone
	Table        db.Table
	ActorRef     string
}

func createGuestQRFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) guestQRFixture {
	t.Helper()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	actorRef := "user:guest-qr-" + suffix
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
		Slug:               "guest-qr-" + suffix,
		Name:               "Guest QR Group",
		LegacyRestaurantID: pgtype.Text{String: "legacy-guest-qr-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create organisation: %v", err)
	}
	location, err := q.CreateLocation(ctx, db.CreateLocationParams{
		OrganisationID:     org.ID,
		Slug:               "main-" + suffix,
		Name:               "Main Street",
		Timezone:           "Africa/Johannesburg",
		LegacyRestaurantID: pgtype.Text{String: "legacy-main-" + suffix, Valid: true},
	})
	if err != nil {
		t.Fatalf("create location: %v", err)
	}
	floor, err := q.CreateFloor(ctx, db.CreateFloorParams{
		OrganisationID: org.ID,
		LocationID:     location.ID,
		Slug:           "floor-" + suffix,
		Name:           "Ground Floor",
		Canvas:         []byte(`{"width":1200,"height":800,"unit":"px"}`),
	})
	if err != nil {
		t.Fatalf("create floor: %v", err)
	}
	zone, err := q.CreateZone(ctx, db.CreateZoneParams{
		OrganisationID: org.ID,
		LocationID:     location.ID,
		FloorID:        floor.ID,
		Name:           "Patio " + suffix,
	})
	if err != nil {
		t.Fatalf("create zone: %v", err)
	}
	table, err := q.CreateTable(ctx, db.CreateTableParams{
		OrganisationID: org.ID,
		LocationID:     location.ID,
		FloorID:        floor.ID,
		ZoneID:         zone.ID,
		Label:          "T-" + suffix[:8],
		CapacityLabel:  "4",
		Shape:          "rectangle",
		Geometry:       []byte(`{"x":120,"y":160,"width":120,"height":80,"rotation":0}`),
	})
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	if _, err := q.CreateTableOccupancy(ctx, db.CreateTableOccupancyParams{
		TableID:        table.ID,
		OrganisationID: org.ID,
		LocationID:     location.ID,
	}); err != nil {
		t.Fatalf("create occupancy: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit fixture transaction: %v", err)
	}

	return guestQRFixture{
		Organisation: org,
		Location:     location,
		Floor:        floor,
		Zone:         zone,
		Table:        table,
		ActorRef:     actorRef,
	}
}

func assertTenantCommand(t *testing.T, handler http.Handler, method string, path string, fixture guestQRFixture, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(headerOrganisationID, fixture.Organisation.ID.String())
	req.Header.Set(headerLocationID, fixture.Location.ID.String())
	req.Header.Set(headerActorRef, fixture.ActorRef)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("%s %s status = %d, want 2xx; body: %s", method, path, rec.Code, rec.Body.String())
	}
	return rec
}

func assertPublicRequest(t *testing.T, handler http.Handler, method string, path string, body string, status int) *httptest.ResponseRecorder {
	t.Helper()

	reader := strings.NewReader(body)
	if body == "" {
		reader = strings.NewReader(`{}`)
	}
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != status {
		t.Fatalf("%s %s status = %d, want %d; body: %s", method, path, rec.Code, status, rec.Body.String())
	}
	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, out any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
		t.Fatalf("decode response: %v; body: %s", err, rec.Body.String())
	}
}

func assertGuestAbuseDecisionCount(t *testing.T, pool *pgxpool.Pool, decision string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM guest_abuse_events WHERE decision = $1`, decision).Scan(&got); err != nil {
		t.Fatalf("count %s events: %v", decision, err)
	}
	if got != want {
		t.Fatalf("%s events = %d, want %d", decision, got, want)
	}
}

func itoa32(value int32) string {
	if value == 0 {
		return "0"
	}
	var out [11]byte
	i := len(out)
	n := value
	for n > 0 {
		i--
		out[i] = byte('0' + n%10)
		n /= 10
	}
	return string(out[i:])
}
