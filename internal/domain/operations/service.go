package operations

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/domain/events"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	TableStatusAvailable = "available"
	TableStatusOccupied  = "occupied"

	AssistStatusPending      = "pending"
	AssistStatusAcknowledged = "acknowledged"
	AssistStatusResolved     = "resolved"
	AssistStatusCancelled    = "cancelled"

	DefaultIdempotencyRetention = 30 * 24 * time.Hour
)

type Service struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: db.New(pool),
	}
}

type OccupyTableParams struct {
	OrganisationID  uuid.UUID
	LocationID      uuid.UUID
	TableID         uuid.UUID
	ExpectedVersion int32
	PartySize       *int32
	ActorRef        string
	Source          string
	Command         CommandMeta
}

type OccupyTableResult struct {
	Table     db.Table
	Occupancy db.TableOccupancy
	Session   db.TableSession
}

type CommandMeta struct {
	ID          uuid.UUID
	Type        string
	PayloadHash []byte
	DeviceID    uuid.NullUUID
	Retention   time.Duration
}

type CommandReplay struct {
	Replayed bool
	Status   int32
	Body     []byte
}

func (s *Service) OccupyTable(ctx context.Context, arg OccupyTableParams) (OccupyTableResult, error) {
	result, _, err := s.OccupyTableCommand(ctx, arg, nil)
	return result, err
}

func (s *Service) OccupyTableCommand(
	ctx context.Context,
	arg OccupyTableParams,
	encode func(OccupyTableResult) (int32, []byte, error),
) (OccupyTableResult, CommandReplay, error) {
	return runTenantCommand(ctx, s, arg.OrganisationID, arg.LocationID, arg.ActorRef, arg.Command, func(q *db.Queries) (OccupyTableResult, error) {
		return occupyTableInTx(ctx, q, arg)
	}, func(tx pgx.Tx, result OccupyTableResult) error {
		return writeOperationalEvent(ctx, tx, operationalEvent{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			EventType:      events.TypeTableOccupied,
			EntityType:     "table",
			EntityID:       arg.TableID,
			EntityVersion:  result.Occupancy.Version,
			ActorRef:       arg.ActorRef,
			DeviceID:       arg.Command.DeviceID,
			CommandID:      nullableUUID(arg.Command.ID),
			Topic:          "operations.table",
			Destinations:   operationDestinations(),
			Data: events.Data{
				"tableId":   arg.TableID.String(),
				"sessionId": result.Session.ID.String(),
				"status":    result.Occupancy.Status,
				"partySize": intPtr(arg.PartySize),
			},
		})
	}, encode)
}

type ClearTableParams struct {
	OrganisationID  uuid.UUID
	LocationID      uuid.UUID
	TableID         uuid.UUID
	ExpectedVersion int32
	ActorRef        string
	Command         CommandMeta
}

type ClearTableResult struct {
	Table     db.Table
	Occupancy db.TableOccupancy
	Session   db.TableSession
}

func (s *Service) ClearTable(ctx context.Context, arg ClearTableParams) (ClearTableResult, error) {
	result, _, err := s.ClearTableCommand(ctx, arg, nil)
	return result, err
}

func (s *Service) ClearTableCommand(
	ctx context.Context,
	arg ClearTableParams,
	encode func(ClearTableResult) (int32, []byte, error),
) (ClearTableResult, CommandReplay, error) {
	return runTenantCommand(ctx, s, arg.OrganisationID, arg.LocationID, arg.ActorRef, arg.Command, func(q *db.Queries) (ClearTableResult, error) {
		return clearTableInTx(ctx, q, arg)
	}, func(tx pgx.Tx, result ClearTableResult) error {
		return writeOperationalEvent(ctx, tx, operationalEvent{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			EventType:      events.TypeTableCleared,
			EntityType:     "table",
			EntityID:       arg.TableID,
			EntityVersion:  result.Occupancy.Version,
			ActorRef:       arg.ActorRef,
			DeviceID:       arg.Command.DeviceID,
			CommandID:      nullableUUID(arg.Command.ID),
			Topic:          "operations.table",
			Destinations:   operationDestinations(),
			Data: events.Data{
				"tableId":   arg.TableID.String(),
				"sessionId": result.Session.ID.String(),
				"status":    result.Occupancy.Status,
			},
		})
	}, encode)
}

type RequestAssistParams struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	TableID        uuid.UUID
	RequestedBy    string
	Source         string
	Note           string
}

func (s *Service) RequestAssist(ctx context.Context, arg RequestAssistParams) (db.AssistRequest, error) {
	var assist db.AssistRequest
	err := s.inTenantTx(ctx, arg.OrganisationID, arg.LocationID, func(tx pgx.Tx, q *db.Queries) error {
		occupancy, err := lockTableOccupancy(ctx, q, arg.TableID, arg.OrganisationID, arg.LocationID)
		if err != nil {
			return err
		}

		created, err := q.CreateAssistRequest(ctx, db.CreateAssistRequestParams{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			TableID:        arg.TableID,
			TableSessionID: occupancy.CurrentSessionID,
			RequestedBy:    nullableText(arg.RequestedBy),
			Source:         arg.Source,
			Note:           nullableText(arg.Note),
		})
		if err != nil {
			return fmt.Errorf("creating assist request: %w", err)
		}
		assist = created
		return writeOperationalEvent(ctx, tx, operationalEvent{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			EventType:      events.TypeAssistRequested,
			EntityType:     "assist",
			EntityID:       created.ID,
			EntityVersion:  created.Version,
			ActorRef:       arg.RequestedBy,
			Topic:          "operations.assist",
			Destinations:   operationDestinations(),
			Data: events.Data{
				"assistId": created.ID.String(),
				"tableId":  arg.TableID.String(),
				"status":   created.Status,
				"source":   arg.Source,
				"note":     nullableString(arg.Note),
			},
		})
	})
	return assist, err
}

type ChangeAssistParams struct {
	OrganisationID  uuid.UUID
	LocationID      uuid.UUID
	AssistID        uuid.UUID
	ExpectedVersion int32
	ActorRef        string
	Command         CommandMeta
}

func (s *Service) AcknowledgeAssist(ctx context.Context, arg ChangeAssistParams) (db.AssistRequest, error) {
	assist, _, err := s.AcknowledgeAssistCommand(ctx, arg, nil)
	return assist, err
}

func (s *Service) ResolveAssist(ctx context.Context, arg ChangeAssistParams) (db.AssistRequest, error) {
	assist, _, err := s.ResolveAssistCommand(ctx, arg, nil)
	return assist, err
}

func (s *Service) CancelAssist(ctx context.Context, arg ChangeAssistParams) (db.AssistRequest, error) {
	assist, _, err := s.CancelAssistCommand(ctx, arg, nil)
	return assist, err
}

func (s *Service) AcknowledgeAssistCommand(
	ctx context.Context,
	arg ChangeAssistParams,
	encode func(db.AssistRequest) (int32, []byte, error),
) (db.AssistRequest, CommandReplay, error) {
	return s.changeAssistCommand(ctx, arg, AssistStatusAcknowledged, "assist.acknowledged", encode)
}

func (s *Service) ResolveAssistCommand(
	ctx context.Context,
	arg ChangeAssistParams,
	encode func(db.AssistRequest) (int32, []byte, error),
) (db.AssistRequest, CommandReplay, error) {
	return s.changeAssistCommand(ctx, arg, AssistStatusResolved, "assist.resolved", encode)
}

func (s *Service) CancelAssistCommand(
	ctx context.Context,
	arg ChangeAssistParams,
	encode func(db.AssistRequest) (int32, []byte, error),
) (db.AssistRequest, CommandReplay, error) {
	return s.changeAssistCommand(ctx, arg, AssistStatusCancelled, "assist.cancelled", encode)
}

func (s *Service) changeAssistCommand(
	ctx context.Context,
	arg ChangeAssistParams,
	nextStatus string,
	eventType string,
	encode func(db.AssistRequest) (int32, []byte, error),
) (db.AssistRequest, CommandReplay, error) {
	return runTenantCommand(ctx, s, arg.OrganisationID, arg.LocationID, arg.ActorRef, arg.Command, func(q *db.Queries) (db.AssistRequest, error) {
		return changeAssistInTx(ctx, q, arg, nextStatus)
	}, func(tx pgx.Tx, assist db.AssistRequest) error {
		return writeOperationalEvent(ctx, tx, operationalEvent{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			EventType:      eventType,
			EntityType:     "assist",
			EntityID:       arg.AssistID,
			EntityVersion:  assist.Version,
			ActorRef:       arg.ActorRef,
			DeviceID:       arg.Command.DeviceID,
			CommandID:      nullableUUID(arg.Command.ID),
			Topic:          "operations.assist",
			Destinations:   operationDestinations(),
			Data: events.Data{
				"assistId": arg.AssistID.String(),
				"tableId":  assist.TableID.String(),
				"status":   assist.Status,
			},
		})
	}, encode)
}

func changeAssistInTx(ctx context.Context, q *db.Queries, arg ChangeAssistParams, nextStatus string) (db.AssistRequest, error) {
	current, err := q.LockAssistRequest(ctx, db.LockAssistRequestParams{
		ID:             arg.AssistID,
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AssistRequest{}, ErrNotFound
		}
		return db.AssistRequest{}, fmt.Errorf("locking assist request: %w", err)
	}
	if current.Version != arg.ExpectedVersion {
		return db.AssistRequest{}, VersionConflictError{
			Entity:   "assist_request",
			ID:       arg.AssistID.String(),
			Expected: arg.ExpectedVersion,
			Current:  current.Version,
		}
	}
	if current.Status == AssistStatusResolved && nextStatus == AssistStatusResolved {
		return db.AssistRequest{}, ErrAssistAlreadyResolved
	}
	if !validAssistTransition(current.Status, nextStatus) {
		return db.AssistRequest{}, ErrInvalidAssistTransition
	}

	var updated db.AssistRequest
	switch nextStatus {
	case AssistStatusAcknowledged:
		updated, err = q.AcknowledgeAssistRequest(ctx, db.AcknowledgeAssistRequestParams{
			ID:             arg.AssistID,
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			AcknowledgedBy: nullableText(arg.ActorRef),
			Version:        arg.ExpectedVersion,
		})
	case AssistStatusResolved:
		updated, err = q.ResolveAssistRequest(ctx, db.ResolveAssistRequestParams{
			ID:             arg.AssistID,
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			ResolvedBy:     nullableText(arg.ActorRef),
			Version:        arg.ExpectedVersion,
		})
	case AssistStatusCancelled:
		updated, err = q.CancelAssistRequest(ctx, db.CancelAssistRequestParams{
			ID:             arg.AssistID,
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			CancelledBy:    nullableText(arg.ActorRef),
			Version:        arg.ExpectedVersion,
		})
	default:
		return db.AssistRequest{}, ErrInvalidAssistTransition
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.AssistRequest{}, ErrVersionConflict
		}
		return db.AssistRequest{}, fmt.Errorf("changing assist request status: %w", err)
	}

	return updated, nil
}

func occupyTableInTx(ctx context.Context, q *db.Queries, arg OccupyTableParams) (OccupyTableResult, error) {
	occupancy, err := lockTableOccupancy(ctx, q, arg.TableID, arg.OrganisationID, arg.LocationID)
	if err != nil {
		return OccupyTableResult{}, err
	}
	if occupancy.Status == TableStatusOccupied {
		return OccupyTableResult{}, ErrAlreadyOccupied
	}
	if occupancy.Version != arg.ExpectedVersion {
		return OccupyTableResult{}, VersionConflictError{
			Entity:   "table_occupancy",
			ID:       arg.TableID.String(),
			Expected: arg.ExpectedVersion,
			Current:  occupancy.Version,
		}
	}

	session, err := q.CreateTableSession(ctx, db.CreateTableSessionParams{
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
		TableID:        arg.TableID,
		PartySize:      nullableInt4(arg.PartySize),
		OpenedBy:       nullableText(arg.ActorRef),
		Source:         arg.Source,
	})
	if err != nil {
		return OccupyTableResult{}, fmt.Errorf("creating table session: %w", err)
	}

	updated, err := q.MarkTableOccupied(ctx, db.MarkTableOccupiedParams{
		TableID:          arg.TableID,
		OrganisationID:   arg.OrganisationID,
		LocationID:       arg.LocationID,
		CurrentSessionID: uuid.NullUUID{UUID: session.ID, Valid: true},
		UpdatedBy:        nullableText(arg.ActorRef),
		Version:          arg.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return OccupyTableResult{}, ErrVersionConflict
		}
		return OccupyTableResult{}, fmt.Errorf("marking table occupied: %w", err)
	}

	state, err := q.GetTableState(ctx, db.GetTableStateParams{
		ID:             arg.TableID,
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
	})
	if err != nil {
		return OccupyTableResult{}, fmt.Errorf("getting updated table state: %w", err)
	}

	return OccupyTableResult{Table: state.Table, Occupancy: updated, Session: session}, nil
}

func clearTableInTx(ctx context.Context, q *db.Queries, arg ClearTableParams) (ClearTableResult, error) {
	occupancy, err := lockTableOccupancy(ctx, q, arg.TableID, arg.OrganisationID, arg.LocationID)
	if err != nil {
		return ClearTableResult{}, err
	}
	if occupancy.Status == TableStatusAvailable {
		return ClearTableResult{}, ErrAlreadyAvailable
	}
	if occupancy.Version != arg.ExpectedVersion {
		return ClearTableResult{}, VersionConflictError{
			Entity:   "table_occupancy",
			ID:       arg.TableID.String(),
			Expected: arg.ExpectedVersion,
			Current:  occupancy.Version,
		}
	}
	if !occupancy.CurrentSessionID.Valid {
		return ClearTableResult{}, ErrNoActiveSession
	}

	session, err := q.CloseTableSession(ctx, db.CloseTableSessionParams{
		ID:             occupancy.CurrentSessionID.UUID,
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
		ClosedBy:       nullableText(arg.ActorRef),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClearTableResult{}, ErrNoActiveSession
		}
		return ClearTableResult{}, fmt.Errorf("closing table session: %w", err)
	}

	updated, err := q.MarkTableAvailable(ctx, db.MarkTableAvailableParams{
		TableID:        arg.TableID,
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
		UpdatedBy:      nullableText(arg.ActorRef),
		Version:        arg.ExpectedVersion,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ClearTableResult{}, ErrVersionConflict
		}
		return ClearTableResult{}, fmt.Errorf("marking table available: %w", err)
	}

	state, err := q.GetTableState(ctx, db.GetTableStateParams{
		ID:             arg.TableID,
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
	})
	if err != nil {
		return ClearTableResult{}, fmt.Errorf("getting updated table state: %w", err)
	}

	return ClearTableResult{Table: state.Table, Occupancy: updated, Session: session}, nil
}

func (s *Service) inTx(ctx context.Context, fn func(*db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if err := fn(s.queries.WithTx(tx)); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (s *Service) inTenantTx(ctx context.Context, organisationID, locationID uuid.UUID, fn func(pgx.Tx, *db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}

	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, organisationID, locationID); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(fmt.Errorf("setting tenant context: %w", err), fmt.Errorf("rolling back transaction: %w", rollbackErr))
		}
		return fmt.Errorf("setting tenant context: %w", err)
	}

	if err := fn(tx, s.queries.WithTx(tx)); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

type operationalEvent struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	EventType      string
	EntityType     string
	EntityID       uuid.UUID
	EntityVersion  int32
	ActorRef       string
	DeviceID       uuid.NullUUID
	CommandID      uuid.NullUUID
	Topic          string
	Destinations   []string
	Data           events.Data
}

func runTenantCommand[T any](
	ctx context.Context,
	s *Service,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	actorRef string,
	meta CommandMeta,
	execute func(*db.Queries) (T, error),
	after func(pgx.Tx, T) error,
	encode func(T) (int32, []byte, error),
) (T, CommandReplay, error) {
	var zero T
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return zero, CommandReplay{}, fmt.Errorf("beginning transaction: %w", err)
	}

	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, organisationID.String(), locationID.String()); err != nil {
		return zero, CommandReplay{}, rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}

	queries := s.queries.WithTx(tx)
	if meta.ID != uuid.Nil {
		replay, found, err := getIdempotencyReplay(ctx, tx, organisationID, meta)
		if err != nil {
			return zero, CommandReplay{}, rollback(tx, ctx, err)
		}
		if found {
			if err := tx.Commit(ctx); err != nil {
				return zero, CommandReplay{}, fmt.Errorf("committing replay transaction: %w", err)
			}
			return zero, replay, nil
		}
	}

	value, err := execute(queries)
	if err != nil {
		return zero, CommandReplay{}, rollback(tx, ctx, err)
	}
	if err := after(tx, value); err != nil {
		return zero, CommandReplay{}, rollback(tx, ctx, err)
	}

	var status int32 = 200
	var body []byte
	if encode != nil {
		status, body, err = encode(value)
		if err != nil {
			return zero, CommandReplay{}, rollback(tx, ctx, fmt.Errorf("encoding idempotency response: %w", err))
		}
	}
	if meta.ID != uuid.Nil {
		if len(body) == 0 {
			body = []byte(`{}`)
		}
		if err := insertIdempotencyRecord(ctx, tx, organisationID, locationID, actorRef, meta, status, body); err != nil {
			return zero, CommandReplay{}, rollback(tx, ctx, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return zero, CommandReplay{}, fmt.Errorf("committing transaction: %w", err)
	}
	return value, CommandReplay{Status: status, Body: body}, nil
}

func getIdempotencyReplay(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, meta CommandMeta) (CommandReplay, bool, error) {
	var commandType string
	var payloadHash []byte
	var status int32
	var body []byte
	err := tx.QueryRow(ctx, `
SELECT command_type, payload_hash, response_status, response_body
FROM api_idempotency_records
WHERE organisation_id = $1 AND command_id = $2
FOR UPDATE
`, organisationID, meta.ID).Scan(&commandType, &payloadHash, &status, &body)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return CommandReplay{}, false, nil
		}
		return CommandReplay{}, false, fmt.Errorf("getting idempotency record: %w", err)
	}
	if commandType != meta.Type || !bytes.Equal(payloadHash, meta.PayloadHash) {
		return CommandReplay{}, false, ErrIdempotencyConflict
	}
	return CommandReplay{Replayed: true, Status: status, Body: body}, true, nil
}

func insertIdempotencyRecord(
	ctx context.Context,
	tx pgx.Tx,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	actorRef string,
	meta CommandMeta,
	status int32,
	body []byte,
) error {
	retention := meta.Retention
	if retention <= 0 {
		retention = DefaultIdempotencyRetention
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO api_idempotency_records (
    command_id,
    organisation_id,
    location_id,
    command_type,
    payload_hash,
    response_status,
    response_body,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, now() + ($8::bigint * interval '1 second'))
`, meta.ID, organisationID, locationID, meta.Type, meta.PayloadHash, status, body, int64(retention.Seconds())); err != nil {
		return fmt.Errorf("recording idempotency result for %s: %w", actorRef, err)
	}
	return nil
}

func writeOperationalEvent(ctx context.Context, tx pgx.Tx, event operationalEvent) error {
	data, err := events.MarshalData(event.Data)
	if err != nil {
		return fmt.Errorf("encoding operational event data: %w", err)
	}
	actorRef := event.ActorRef
	if actorRef == "" {
		actorRef = "system"
	}
	eventID := uuid.New()
	now := time.Now().UTC()
	var deviceID *uuid.UUID
	if event.DeviceID.Valid {
		deviceID = &event.DeviceID.UUID
	}
	var commandID *uuid.UUID
	if event.CommandID.Valid {
		commandID = &event.CommandID.UUID
	}
	entityVersion := event.EntityVersion
	envelope := events.Envelope{
		ID:             eventID,
		Type:           event.EventType,
		SchemaVersion:  events.SchemaVersion,
		OccurredAt:     now,
		OrganisationID: event.OrganisationID,
		LocationID:     event.LocationID,
		ActorRef:       actorRef,
		DeviceID:       deviceID,
		EntityType:     event.EntityType,
		EntityID:       event.EntityID,
		EntityVersion:  &entityVersion,
		CommandID:      commandID,
		CorrelationID:  commandID,
		Data:           data,
	}
	payload, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encoding operational event envelope: %w", err)
	}

	if _, err := tx.Exec(ctx, `
INSERT INTO operational_events (
    id,
    organisation_id,
    location_id,
    event_type,
    schema_version,
    aggregate_type,
    aggregate_id,
    aggregate_version,
    entity_type,
    entity_id,
    entity_version,
    actor_ref,
    device_id,
    command_id,
    correlation_id,
    event_data,
    payload,
    occurred_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $6, $7, $8, $9, $10, $11, $11, $12, $13, $14)
`, eventID, event.OrganisationID, event.LocationID, event.EventType, events.SchemaVersion, event.EntityType, event.EntityID, event.EntityVersion, actorRef, event.DeviceID, event.CommandID, data, payload, now); err != nil {
		return fmt.Errorf("writing operational event: %w", err)
	}
	topic := event.Topic
	if topic == "" {
		topic = event.EntityType
	}
	destinations := event.Destinations
	if len(destinations) == 0 {
		destinations = []string{topic}
	}
	for _, destination := range destinations {
		if _, err := tx.Exec(ctx, `
INSERT INTO outbox_records (organisation_id, location_id, event_id, topic, destination, payload)
VALUES ($1, $2, $3, $4, $5, $6)
`, event.OrganisationID, event.LocationID, eventID, topic, destination, payload); err != nil {
			return fmt.Errorf("writing outbox record for %s: %w", destination, err)
		}
	}
	return nil
}

func operationDestinations() []string {
	return []string{"realtime.operations", "analytics.operations", "audit.operations"}
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}

func lockTableOccupancy(ctx context.Context, q *db.Queries, tableID, organisationID, locationID uuid.UUID) (db.TableOccupancy, error) {
	occupancy, err := q.LockTableOccupancy(ctx, db.LockTableOccupancyParams{
		TableID:        tableID,
		OrganisationID: organisationID,
		LocationID:     locationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.TableOccupancy{}, ErrNotFound
		}
		return db.TableOccupancy{}, fmt.Errorf("locking table occupancy: %w", err)
	}
	return occupancy, nil
}

func validAssistTransition(current, next string) bool {
	switch next {
	case AssistStatusAcknowledged:
		return current == AssistStatusPending
	case AssistStatusResolved, AssistStatusCancelled:
		return current == AssistStatusPending || current == AssistStatusAcknowledged
	default:
		return false
	}
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func nullableInt4(value *int32) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *value, Valid: true}
}

func intPtr(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableUUID(value uuid.UUID) uuid.NullUUID {
	if value == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: value, Valid: true}
}
