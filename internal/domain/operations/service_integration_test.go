//go:build integration

package operations_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/seatd/seatd/internal/domain/operations"
	"github.com/seatd/seatd/internal/store"
	"github.com/seatd/seatd/internal/store/db"
)

func TestServiceTableAndAssistTransitions(t *testing.T) {
	ctx := context.Background()
	pool := setupDatabase(t, ctx)
	fixture := createFixture(t, ctx, pool)
	service := operations.NewService(pool)

	partySize := int32(4)
	occupied, err := service.OccupyTable(ctx, operations.OccupyTableParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		TableID:         fixture.Table.ID,
		ExpectedVersion: 1,
		PartySize:       &partySize,
		ActorRef:        "user:waiter-1",
		Source:          "waiter",
	})
	if err != nil {
		t.Fatalf("occupy table: %v", err)
	}
	if occupied.Occupancy.Status != operations.TableStatusOccupied {
		t.Fatalf("occupancy status = %q, want occupied", occupied.Occupancy.Status)
	}
	if occupied.Occupancy.Version != 2 {
		t.Fatalf("occupancy version = %d, want 2", occupied.Occupancy.Version)
	}
	if !occupied.Occupancy.CurrentSessionID.Valid || occupied.Occupancy.CurrentSessionID.UUID != occupied.Session.ID {
		t.Fatalf("current session not linked to opened session")
	}

	assist, err := service.RequestAssist(ctx, operations.RequestAssistParams{
		OrganisationID: fixture.Organisation.ID,
		LocationID:     fixture.Location.ID,
		TableID:        fixture.Table.ID,
		RequestedBy:    "guest",
		Source:         "guest_qr",
		Note:           "bill",
	})
	if err != nil {
		t.Fatalf("request assist: %v", err)
	}
	if assist.Status != operations.AssistStatusPending {
		t.Fatalf("assist status = %q, want pending", assist.Status)
	}
	if !assist.TableSessionID.Valid || assist.TableSessionID.UUID != occupied.Session.ID {
		t.Fatalf("assist not linked to active session")
	}

	acknowledged, err := service.AcknowledgeAssist(ctx, operations.ChangeAssistParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		AssistID:        assist.ID,
		ExpectedVersion: assist.Version,
		ActorRef:        "user:waiter-2",
	})
	if err != nil {
		t.Fatalf("acknowledge assist: %v", err)
	}

	resolved, err := service.ResolveAssist(ctx, operations.ChangeAssistParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		AssistID:        assist.ID,
		ExpectedVersion: acknowledged.Version,
		ActorRef:        "user:waiter-2",
	})
	if err != nil {
		t.Fatalf("resolve assist: %v", err)
	}
	if resolved.Status != operations.AssistStatusResolved {
		t.Fatalf("assist status = %q, want resolved", resolved.Status)
	}

	var state db.GetTableStateRow
	withTenantQueries(t, ctx, pool, fixture.Organisation.ID, fixture.Location.ID, func(q *db.Queries) {
		var err error
		state, err = q.GetTableState(ctx, db.GetTableStateParams{
			ID:             fixture.Table.ID,
			OrganisationID: fixture.Organisation.ID,
			LocationID:     fixture.Location.ID,
		})
		if err != nil {
			t.Fatalf("get table state: %v", err)
		}
	})
	if state.TableOccupancy.Status != operations.TableStatusOccupied {
		t.Fatalf("resolving assist changed occupancy to %q", state.TableOccupancy.Status)
	}

	cleared, err := service.ClearTable(ctx, operations.ClearTableParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		TableID:         fixture.Table.ID,
		ExpectedVersion: state.TableOccupancy.Version,
		ActorRef:        "user:waiter-1",
	})
	if err != nil {
		t.Fatalf("clear table: %v", err)
	}
	if cleared.Occupancy.Status != operations.TableStatusAvailable {
		t.Fatalf("occupancy status = %q, want available", cleared.Occupancy.Status)
	}
	if !cleared.Session.EndedAt.Valid {
		t.Fatalf("cleared session has no ended_at")
	}

	withTenantTx(t, ctx, pool, fixture.Organisation.ID, fixture.Location.ID, func(tx pgx.Tx, _ *db.Queries) {
		rows, err := tx.Query(ctx, `
SELECT event_type, schema_version, entity_type, entity_id, entity_version, actor_ref, event_data
FROM operational_events
WHERE organisation_id = $1 AND location_id = $2
ORDER BY occurred_at, id
`, fixture.Organisation.ID, fixture.Location.ID)
		if err != nil {
			t.Fatalf("query operational events: %v", err)
		}
		defer rows.Close()

		type eventRow struct {
			eventType     string
			schemaVersion int32
			entityType    string
			entityID      uuid.UUID
			entityVersion int32
			actorRef      string
			data          map[string]any
		}
		var events []eventRow
		for rows.Next() {
			var row eventRow
			var data []byte
			if err := rows.Scan(&row.eventType, &row.schemaVersion, &row.entityType, &row.entityID, &row.entityVersion, &row.actorRef, &data); err != nil {
				t.Fatalf("scan operational event: %v", err)
			}
			if err := json.Unmarshal(data, &row.data); err != nil {
				t.Fatalf("decode operational event data: %v", err)
			}
			events = append(events, row)
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("iterate operational events: %v", err)
		}
		wantTypes := []string{"table.occupied", "assist.requested", "assist.acknowledged", "assist.resolved", "table.cleared"}
		if len(events) != len(wantTypes) {
			t.Fatalf("event count = %d, want %d", len(events), len(wantTypes))
		}
		for i, want := range wantTypes {
			if events[i].eventType != want {
				t.Fatalf("event[%d] type = %q, want %q", i, events[i].eventType, want)
			}
			if events[i].schemaVersion != 1 {
				t.Fatalf("event[%d] schema version = %d, want 1", i, events[i].schemaVersion)
			}
			if events[i].entityType == "" || events[i].entityID == uuid.Nil || events[i].entityVersion < 1 {
				t.Fatalf("event[%d] has incomplete entity envelope: %+v", i, events[i])
			}
		}

		var outboxCount int
		if err := tx.QueryRow(ctx, `
SELECT count(*)
FROM outbox_records
WHERE organisation_id = $1 AND location_id = $2 AND status = 'pending'
`, fixture.Organisation.ID, fixture.Location.ID).Scan(&outboxCount); err != nil {
			t.Fatalf("count outbox records: %v", err)
		}
		if outboxCount != len(wantTypes) {
			t.Fatalf("outbox count = %d, want %d", outboxCount, len(wantTypes))
		}
	})
}

func TestServiceRejectsInvalidStateChanges(t *testing.T) {
	ctx := context.Background()
	pool := setupDatabase(t, ctx)
	fixture := createFixture(t, ctx, pool)
	service := operations.NewService(pool)

	_, err := service.OccupyTable(ctx, operations.OccupyTableParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		TableID:         fixture.Table.ID,
		ExpectedVersion: 99,
		Source:          "waiter",
	})
	if !errors.Is(err, operations.ErrVersionConflict) {
		t.Fatalf("stale occupy error = %v, want version conflict", err)
	}

	occupied, err := service.OccupyTable(ctx, operations.OccupyTableParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		TableID:         fixture.Table.ID,
		ExpectedVersion: 1,
		Source:          "waiter",
	})
	if err != nil {
		t.Fatalf("occupy table: %v", err)
	}

	_, err = service.OccupyTable(ctx, operations.OccupyTableParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		TableID:         fixture.Table.ID,
		ExpectedVersion: occupied.Occupancy.Version,
		Source:          "waiter",
	})
	if !errors.Is(err, operations.ErrAlreadyOccupied) {
		t.Fatalf("double occupy error = %v, want already occupied", err)
	}

	assist, err := service.RequestAssist(ctx, operations.RequestAssistParams{
		OrganisationID: fixture.Organisation.ID,
		LocationID:     fixture.Location.ID,
		TableID:        fixture.Table.ID,
		Source:         "guest_qr",
	})
	if err != nil {
		t.Fatalf("request assist: %v", err)
	}
	cancelled, err := service.CancelAssist(ctx, operations.ChangeAssistParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		AssistID:        assist.ID,
		ExpectedVersion: assist.Version,
	})
	if err != nil {
		t.Fatalf("cancel assist: %v", err)
	}
	_, err = service.ResolveAssist(ctx, operations.ChangeAssistParams{
		OrganisationID:  fixture.Organisation.ID,
		LocationID:      fixture.Location.ID,
		AssistID:        assist.ID,
		ExpectedVersion: cancelled.Version,
	})
	if !errors.Is(err, operations.ErrInvalidAssistTransition) {
		t.Fatalf("resolve cancelled assist error = %v, want invalid transition", err)
	}

	_, err = pool.Exec(ctx, "UPDATE tables SET is_active = false, deleted_at = now() WHERE id = $1", fixture.Table.ID)
	if err != nil {
		t.Fatalf("disable table: %v", err)
	}
	_, err = service.RequestAssist(ctx, operations.RequestAssistParams{
		OrganisationID: fixture.Organisation.ID,
		LocationID:     fixture.Location.ID,
		TableID:        fixture.Table.ID,
		Source:         "guest_qr",
	})
	if !errors.Is(err, operations.ErrNotFound) {
		t.Fatalf("assist on inactive table error = %v, want not found", err)
	}
}

func TestDatabaseConstraintsRejectBrokenInvariants(t *testing.T) {
	ctx := context.Background()
	pool := setupDatabase(t, ctx)
	fixture := createFixture(t, ctx, pool)
	other := createFixture(t, ctx, pool)

	withAdminQueries(t, ctx, pool, func(q *db.Queries) {
		_, err := q.CreateTableSession(ctx, db.CreateTableSessionParams{
			OrganisationID: fixture.Organisation.ID,
			LocationID:     fixture.Location.ID,
			TableID:        fixture.Table.ID,
			Source:         "test",
		})
		if err != nil {
			t.Fatalf("create first active session: %v", err)
		}
		_, err = q.CreateTableSession(ctx, db.CreateTableSessionParams{
			OrganisationID: fixture.Organisation.ID,
			LocationID:     fixture.Location.ID,
			TableID:        fixture.Table.ID,
			Source:         "test",
		})
		if err == nil {
			t.Fatalf("second active session succeeded, want constraint failure")
		}
	})

	withAdminQueries(t, ctx, pool, func(q *db.Queries) {
		token := "test-token"
		_, err := q.CreateTableQRCapability(ctx, db.CreateTableQRCapabilityParams{
			OrganisationID: fixture.Organisation.ID,
			LocationID:     fixture.Location.ID,
			TableID:        fixture.Table.ID,
			Token:          token,
		})
		if err != nil {
			t.Fatalf("create first qr capability: %v", err)
		}
		_, err = q.CreateTableQRCapability(ctx, db.CreateTableQRCapabilityParams{
			OrganisationID: fixture.Organisation.ID,
			LocationID:     fixture.Location.ID,
			TableID:        fixture.Table.ID,
			Token:          token,
		})
		if err == nil {
			t.Fatalf("duplicate qr token succeeded, want constraint failure")
		}
	})

	withAdminQueries(t, ctx, pool, func(q *db.Queries) {
		_, err := q.CreateTable(ctx, db.CreateTableParams{
			OrganisationID: fixture.Organisation.ID,
			LocationID:     fixture.Location.ID,
			FloorID:        other.Floor.ID,
			ZoneID:         other.Zone.ID,
			Label:          "bad-tenant-chain",
			CapacityLabel:  "2",
			Shape:          "rectangle",
			Geometry:       []byte(`{"x":1,"y":1,"width":1,"height":1}`),
		})
		if err == nil {
			t.Fatalf("cross-tenant table insert succeeded, want foreign key failure")
		}
	})
}

func TestTenantRLSBlocksCrossTenantAccess(t *testing.T) {
	ctx := context.Background()
	pool := setupDatabase(t, ctx)
	fixture := createFixture(t, ctx, pool)
	other := createFixture(t, ctx, pool)

	withTenantTx(t, ctx, pool, fixture.Organisation.ID, fixture.Location.ID, func(tx pgx.Tx, _ *db.Queries) {
		var tableID uuid.UUID
		err := tx.QueryRow(ctx, "SELECT id FROM tables WHERE id = $1", other.Table.ID).Scan(&tableID)
		if !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("cross-tenant flawed read error = %v, want no rows", err)
		}

		_, err = tx.Exec(ctx, `
INSERT INTO assist_requests (organisation_id, location_id, table_id, source)
VALUES ($1, $2, $3, 'rls-regression')
`, other.Organisation.ID, other.Location.ID, other.Table.ID)
		if err == nil {
			t.Fatalf("cross-tenant flawed write succeeded, want RLS failure")
		}
	})
}

type fixture struct {
	Organisation db.Organisation
	Location     db.Location
	Floor        db.Floor
	Zone         db.Zone
	Table        db.Table
}

func setupDatabase(t *testing.T, ctx context.Context) *pgxpool.Pool {
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
	for _, path := range migrationPaths(t) {
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

func migrationPaths(t *testing.T) []string {
	t.Helper()

	root := repoRoot(t)
	entries, err := filepath.Glob(filepath.Join(root, "infra/database/migrations/*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	return entries
}

func repoRoot(t *testing.T) string {
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

func createFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) fixture {
	t.Helper()

	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")
	var created fixture
	withAdminQueries(t, ctx, pool, func(q *db.Queries) {
		org, err := q.CreateOrganisation(ctx, db.CreateOrganisationParams{
			Slug:               "org-" + suffix,
			Name:               "Seatd Test Group",
			LegacyRestaurantID: pgtype.Text{String: "legacy-restaurant-" + suffix, Valid: true},
		})
		if err != nil {
			t.Fatalf("create organisation: %v", err)
		}
		location, err := q.CreateLocation(ctx, db.CreateLocationParams{
			OrganisationID:     org.ID,
			Slug:               "location-" + suffix,
			Name:               "Main Street",
			Timezone:           "Africa/Johannesburg",
			LegacyRestaurantID: pgtype.Text{String: "legacy-location-" + suffix, Valid: true},
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
			t.Fatalf("create table occupancy: %v", err)
		}

		created = fixture{
			Organisation: org,
			Location:     location,
			Floor:        floor,
			Zone:         zone,
			Table:        table,
		}
	})
	return created
}

func withAdminQueries(t *testing.T, ctx context.Context, pool *pgxpool.Pool, fn func(*db.Queries)) {
	t.Helper()
	withContextTx(t, ctx, pool, []contextCommand{
		{sql: "SELECT set_config('seatd.platform_admin', 'true', true)"},
	}, func(_ pgx.Tx, q *db.Queries) {
		fn(q)
	})
}

func withTenantQueries(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	fn func(*db.Queries),
) {
	t.Helper()
	withTenantTx(t, ctx, pool, organisationID, locationID, func(_ pgx.Tx, q *db.Queries) {
		fn(q)
	})
}

func withTenantTx(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	fn func(pgx.Tx, *db.Queries),
) {
	t.Helper()
	withContextTx(t, ctx, pool, []contextCommand{
		{sql: "SET LOCAL ROLE seatd_app"},
		{
			sql: `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`,
			args: []any{organisationID.String(), locationID.String()},
		},
	}, fn)
}

type contextCommand struct {
	sql  string
	args []any
}

func withContextTx(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	commands []contextCommand,
	fn func(pgx.Tx, *db.Queries),
) {
	t.Helper()

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin context transaction: %v", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			t.Fatalf("rollback context transaction: %v", err)
		}
	}()
	for _, command := range commands {
		if _, err := tx.Exec(ctx, command.sql, command.args...); err != nil {
			t.Fatalf("set database context: %v", err)
		}
	}

	fn(tx, db.New(tx))

	if err := tx.Commit(ctx); err != nil {
		if errors.Is(err, pgx.ErrTxCommitRollback) {
			return
		}
		t.Fatalf("commit context transaction: %v", err)
	}
}
