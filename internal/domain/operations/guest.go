package operations

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kadebhug/seatd_v2/internal/domain/events"
	"github.com/kadebhug/seatd_v2/internal/observability"
	"github.com/kadebhug/seatd_v2/internal/store/db"
	"go.opentelemetry.io/otel/attribute"
)

const (
	GuestSource            = "guest_qr"
	guestActorRef          = "guest:qr"
	guestTokenPrefixLength = 16
	defaultGuestCooldown   = 2 * time.Minute
)

var guestActionKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type GuestAction struct {
	Key       string
	Label     string
	SortOrder int
	Enabled   bool
}

type GuestContext struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	LocationName   string
	TableID        uuid.UUID
	TableLabel     string
	Occupancy      db.TableOccupancy
	Actions        []GuestAction
	ActiveRequest  *db.AssistRequest
}

type GuestRequestParams struct {
	Token          string
	ActionKey      string
	IdempotencyKey uuid.UUID
	PayloadHash    []byte
}

type GuestCancelParams struct {
	Token          string
	AssistID       uuid.UUID
	IdempotencyKey uuid.UUID
	PayloadHash    []byte
}

type QRExport struct {
	ID           uuid.UUID
	TableID      uuid.UUID
	Token        string
	LookupPrefix string
	Label        string
	ExpiresAt    pgtype.Timestamptz
	IssuedAt     pgtype.Timestamptz
	RevokedAt    pgtype.Timestamptz
	Version      int32
}

func (s *Service) GuestContext(ctx context.Context, token string) (GuestContext, error) {
	ctx, span := observability.StartSpan(ctx, "operations.guest_context")
	defer span.End()
	capability, err := s.resolveQRCapability(ctx, token)
	if err != nil {
		observability.RecordSpanError(span, err)
		return GuestContext{}, err
	}

	var result GuestContext
	err = s.inTenantTx(ctx, capability.OrganisationID, capability.LocationID, func(tx pgx.Tx, q *db.Queries) error {
		if err := q.TouchTableQRCapability(ctx, capability.ID); err != nil {
			return fmt.Errorf("updating qr capability use: %w", err)
		}
		location, err := q.GetLocation(ctx, db.GetLocationParams{
			ID:             capability.LocationID,
			OrganisationID: capability.OrganisationID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("getting guest location: %w", err)
		}
		state, err := q.GetTableState(ctx, db.GetTableStateParams{
			ID:             capability.TableID,
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("getting guest table state: %w", err)
		}
		result = GuestContext{
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			LocationName:   location.Name,
			TableID:        capability.TableID,
			TableLabel:     state.Table.Label,
			Occupancy:      state.TableOccupancy,
			Actions:        GuestActionsFromConfig(location.OperatingConfig),
		}
		if state.TableOccupancy.Status == TableStatusOccupied && state.TableOccupancy.CurrentSessionID.Valid {
			active, err := q.GetActiveGuestAssistBySession(ctx, db.GetActiveGuestAssistBySessionParams{
				OrganisationID: capability.OrganisationID,
				LocationID:     capability.LocationID,
				TableID:        capability.TableID,
				TableSessionID: state.TableOccupancy.CurrentSessionID,
			})
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("getting active guest assist: %w", err)
			}
			if err == nil {
				result.ActiveRequest = &active
			}
		}
		return nil
	})
	observability.RecordSpanError(span, err)
	return result, err
}

func (s *Service) CreateGuestRequest(ctx context.Context, arg GuestRequestParams, encode func(db.AssistRequest) (int32, []byte, error)) (db.AssistRequest, CommandReplay, error) {
	ctx, span := observability.StartSpan(ctx, "operations.create_guest_request",
		attribute.String("seatd.guest_action_key", strings.TrimSpace(arg.ActionKey)),
	)
	defer span.End()
	actionKey := strings.TrimSpace(arg.ActionKey)
	if !validGuestActionKey(actionKey) {
		observability.RecordSpanError(span, ErrActionNotEnabled)
		return db.AssistRequest{}, CommandReplay{}, ErrActionNotEnabled
	}
	capability, err := s.resolveQRCapability(ctx, arg.Token)
	if err != nil {
		observability.RecordSpanError(span, err)
		return db.AssistRequest{}, CommandReplay{}, err
	}
	span.SetAttributes(
		attribute.String("seatd.organisation_id", capability.OrganisationID.String()),
		attribute.String("seatd.location_id", capability.LocationID.String()),
		attribute.String("seatd.table_id", capability.TableID.String()),
	)
	meta := CommandMeta{
		ID:          arg.IdempotencyKey,
		Type:        "guest.request",
		PayloadHash: arg.PayloadHash,
		Retention:   DefaultIdempotencyRetention,
	}
	createdNew := false
	assist, replay, err := runGuestCommand(ctx, s, capability.OrganisationID, capability.LocationID, guestActorRef, meta, func(q *db.Queries) (db.AssistRequest, error) {
		location, err := q.GetLocation(ctx, db.GetLocationParams{
			ID:             capability.LocationID,
			OrganisationID: capability.OrganisationID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.AssistRequest{}, ErrNotFound
			}
			return db.AssistRequest{}, fmt.Errorf("getting guest location: %w", err)
		}
		if !guestActionEnabled(location.OperatingConfig, actionKey) {
			return db.AssistRequest{}, ErrActionNotEnabled
		}
		occupancy, err := lockTableOccupancy(ctx, q, capability.TableID, capability.OrganisationID, capability.LocationID)
		if err != nil {
			return db.AssistRequest{}, err
		}
		if occupancy.Status != TableStatusOccupied || !occupancy.CurrentSessionID.Valid {
			return db.AssistRequest{}, ErrNoActiveSession
		}
		pending, err := q.GetPendingGuestAssistByAction(ctx, db.GetPendingGuestAssistByActionParams{
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			TableID:        capability.TableID,
			TableSessionID: occupancy.CurrentSessionID,
			ActionKey:      pgtype.Text{String: actionKey, Valid: true},
		})
		if err == nil {
			return pending, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return db.AssistRequest{}, fmt.Errorf("getting pending guest assist: %w", err)
		}
		terminal, err := q.GetLatestTerminalGuestAssistByAction(ctx, db.GetLatestTerminalGuestAssistByActionParams{
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			TableID:        capability.TableID,
			TableSessionID: occupancy.CurrentSessionID,
			ActionKey:      pgtype.Text{String: actionKey, Valid: true},
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return db.AssistRequest{}, fmt.Errorf("getting latest terminal guest assist: %w", err)
		}
		if err == nil && time.Since(terminal.RequestedAt.Time) < defaultGuestCooldown {
			return db.AssistRequest{}, ErrRateLimited
		}
		created, err := q.CreateAssistRequest(ctx, db.CreateAssistRequestParams{
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			TableID:        capability.TableID,
			TableSessionID: occupancy.CurrentSessionID,
			RequestedBy:    nullableText(guestActorRef),
			Source:         GuestSource,
			Note:           pgtype.Text{},
			ActionKey:      pgtype.Text{String: actionKey, Valid: true},
		})
		if err != nil {
			return db.AssistRequest{}, fmt.Errorf("creating guest assist request: %w", err)
		}
		createdNew = true
		return created, nil
	}, func(tx pgx.Tx, assist db.AssistRequest) error {
		if !createdNew {
			return nil
		}
		return writeOperationalEvent(ctx, tx, operationalEvent{
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			EventType:      events.TypeAssistRequested,
			EntityType:     "assist",
			EntityID:       assist.ID,
			EntityVersion:  assist.Version,
			ActorRef:       guestActorRef,
			CommandID:      nullableUUID(arg.IdempotencyKey),
			Topic:          "operations.assist",
			Destinations:   operationDestinations(),
			Data: events.Data{
				"assistId":  assist.ID.String(),
				"tableId":   capability.TableID.String(),
				"status":    assist.Status,
				"source":    GuestSource,
				"actionKey": actionKey,
			},
		})
	}, encode)
	observability.RecordSpanError(span, err)
	return assist, replay, err
}

func (s *Service) GetGuestRequest(ctx context.Context, token string, assistID uuid.UUID) (db.AssistRequest, error) {
	ctx, span := observability.StartSpan(ctx, "operations.get_guest_request",
		attribute.String("seatd.assist_id", assistID.String()),
	)
	defer span.End()
	capability, err := s.resolveQRCapability(ctx, token)
	if err != nil {
		observability.RecordSpanError(span, err)
		return db.AssistRequest{}, err
	}
	var assist db.AssistRequest
	err = s.inTenantTx(ctx, capability.OrganisationID, capability.LocationID, func(_ pgx.Tx, q *db.Queries) error {
		found, err := q.GetGuestAssistByID(ctx, db.GetGuestAssistByIDParams{
			ID:             assistID,
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			TableID:        capability.TableID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("getting guest assist: %w", err)
		}
		assist = found
		return nil
	})
	observability.RecordSpanError(span, err)
	return assist, err
}

func (s *Service) CancelGuestRequest(ctx context.Context, arg GuestCancelParams, encode func(db.AssistRequest) (int32, []byte, error)) (db.AssistRequest, CommandReplay, error) {
	ctx, span := observability.StartSpan(ctx, "operations.cancel_guest_request",
		attribute.String("seatd.assist_id", arg.AssistID.String()),
	)
	defer span.End()
	capability, err := s.resolveQRCapability(ctx, arg.Token)
	if err != nil {
		observability.RecordSpanError(span, err)
		return db.AssistRequest{}, CommandReplay{}, err
	}
	meta := CommandMeta{
		ID:          arg.IdempotencyKey,
		Type:        "guest.cancel",
		PayloadHash: arg.PayloadHash,
		Retention:   DefaultIdempotencyRetention,
	}
	assist, replay, err := runGuestCommand(ctx, s, capability.OrganisationID, capability.LocationID, guestActorRef, meta, func(q *db.Queries) (db.AssistRequest, error) {
		current, err := q.GetGuestAssistByID(ctx, db.GetGuestAssistByIDParams{
			ID:             arg.AssistID,
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			TableID:        capability.TableID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.AssistRequest{}, ErrNotFound
			}
			return db.AssistRequest{}, fmt.Errorf("getting guest assist: %w", err)
		}
		if current.Status != AssistStatusPending {
			return db.AssistRequest{}, ErrInvalidAssistTransition
		}
		cancelled, err := q.CancelPendingGuestAssistRequest(ctx, db.CancelPendingGuestAssistRequestParams{
			ID:             arg.AssistID,
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			CancelledBy:    nullableText(guestActorRef),
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return db.AssistRequest{}, ErrInvalidAssistTransition
			}
			return db.AssistRequest{}, fmt.Errorf("cancelling guest assist: %w", err)
		}
		return cancelled, nil
	}, func(tx pgx.Tx, assist db.AssistRequest) error {
		return writeOperationalEvent(ctx, tx, operationalEvent{
			OrganisationID: capability.OrganisationID,
			LocationID:     capability.LocationID,
			EventType:      events.TypeAssistCancelled,
			EntityType:     "assist",
			EntityID:       assist.ID,
			EntityVersion:  assist.Version,
			ActorRef:       guestActorRef,
			CommandID:      nullableUUID(arg.IdempotencyKey),
			Topic:          "operations.assist",
			Destinations:   operationDestinations(),
			Data: events.Data{
				"assistId": assist.ID.String(),
				"tableId":  capability.TableID.String(),
				"status":   assist.Status,
			},
		})
	}, encode)
	observability.RecordSpanError(span, err)
	return assist, replay, err
}

func (s *Service) ExportTableQRCapability(ctx context.Context, organisationID, locationID, tableID uuid.UUID, label string, expiresAt pgtype.Timestamptz) (QRExport, error) {
	token, err := GenerateCapabilityToken()
	if err != nil {
		return QRExport{}, err
	}
	prefix, hash, err := TokenLookupParts(token)
	if err != nil {
		return QRExport{}, err
	}
	var out QRExport
	err = s.inTenantTx(ctx, organisationID, locationID, func(_ pgx.Tx, q *db.Queries) error {
		capability, err := q.CreateTableQRCapabilityHashed(ctx, db.CreateTableQRCapabilityHashedParams{
			OrganisationID:    organisationID,
			LocationID:        locationID,
			TableID:           tableID,
			TokenLookupPrefix: prefix,
			TokenHash:         hash,
			Label:             strings.TrimSpace(label),
			ExpiresAt:         expiresAt,
		})
		if err != nil {
			return fmt.Errorf("creating table qr capability: %w", err)
		}
		out = qrExportFromDB(capability, token)
		return nil
	})
	return out, err
}

func (s *Service) RotateTableQRCapability(ctx context.Context, organisationID, locationID, tableID, capabilityID uuid.UUID) (QRExport, error) {
	var out QRExport
	err := s.inTenantTx(ctx, organisationID, locationID, func(_ pgx.Tx, q *db.Queries) error {
		previous, err := q.RotateTableQRCapability(ctx, db.RotateTableQRCapabilityParams{
			ID:             capabilityID,
			OrganisationID: organisationID,
			LocationID:     locationID,
			TableID:        tableID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("revoking previous table qr capability: %w", err)
		}
		token, err := GenerateCapabilityToken()
		if err != nil {
			return err
		}
		prefix, hash, err := TokenLookupParts(token)
		if err != nil {
			return err
		}
		next, err := q.CreateTableQRCapabilityHashed(ctx, db.CreateTableQRCapabilityHashedParams{
			OrganisationID:    organisationID,
			LocationID:        locationID,
			TableID:           tableID,
			TokenLookupPrefix: prefix,
			TokenHash:         hash,
			Label:             qrLabel(previous.Label),
			ExpiresAt:         previous.ExpiresAt,
		})
		if err != nil {
			return fmt.Errorf("creating rotated table qr capability: %w", err)
		}
		out = qrExportFromDB(next, token)
		return nil
	})
	return out, err
}

func (s *Service) RevokeTableQRCapability(ctx context.Context, organisationID, locationID, tableID, capabilityID uuid.UUID) (db.TableQrCapability, error) {
	var capability db.TableQrCapability
	err := s.inTenantTx(ctx, organisationID, locationID, func(_ pgx.Tx, q *db.Queries) error {
		revoked, err := q.RevokeTableQRCapability(ctx, db.RevokeTableQRCapabilityParams{
			ID:             capabilityID,
			OrganisationID: organisationID,
			LocationID:     locationID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("revoking table qr capability: %w", err)
		}
		if revoked.TableID != tableID {
			return ErrNotFound
		}
		capability = revoked
		return nil
	})
	return capability, err
}

func (s *Service) resolveQRCapability(ctx context.Context, token string) (db.TableQrCapability, error) {
	prefix, hash, err := TokenLookupParts(token)
	if err != nil {
		return db.TableQrCapability{}, ErrNotFound
	}
	var match db.TableQrCapability
	err = s.inTx(ctx, func(q *db.Queries) error {
		candidates, err := q.ListActiveTableQRCapabilitiesByPrefix(ctx, prefix)
		if err != nil {
			return fmt.Errorf("listing qr capabilities by prefix: %w", err)
		}
		for _, candidate := range candidates {
			if subtle.ConstantTimeCompare(candidate.TokenHash, hash) == 1 {
				match = candidate
				break
			}
		}
		if match.ID == uuid.Nil {
			return ErrNotFound
		}
		return nil
	})
	return match, err
}

func runGuestCommand[T any](
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
	ctx, span := observability.StartSpan(ctx, "operations.guest_command",
		attribute.String("seatd.organisation_id", organisationID.String()),
		attribute.String("seatd.location_id", locationID.String()),
		attribute.String("seatd.command_type", meta.Type),
	)
	defer span.End()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		observability.RecordSpanError(span, err)
		return zero, CommandReplay{}, fmt.Errorf("beginning transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, organisationID.String(), locationID.String()); err != nil {
		observability.RecordSpanError(span, err)
		return zero, CommandReplay{}, rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	queries := s.queries.WithTx(tx)
	if meta.ID != uuid.Nil {
		replay, found, err := getIdempotencyReplay(ctx, tx, organisationID, meta)
		if err != nil {
			observability.RecordSpanError(span, err)
			return zero, CommandReplay{}, rollback(tx, ctx, err)
		}
		if found {
			span.SetAttributes(attribute.Bool("seatd.command_replayed", true))
			if err := tx.Commit(ctx); err != nil {
				observability.RecordSpanError(span, err)
				return zero, CommandReplay{}, fmt.Errorf("committing replay transaction: %w", err)
			}
			return zero, replay, nil
		}
	}
	value, err := execute(queries)
	if err != nil {
		observability.RecordSpanError(span, err)
		return zero, CommandReplay{}, rollback(tx, ctx, err)
	}
	if err := after(tx, value); err != nil {
		observability.RecordSpanError(span, err)
		return zero, CommandReplay{}, rollback(tx, ctx, err)
	}
	status, body, err := encode(value)
	if err != nil {
		observability.RecordSpanError(span, err)
		return zero, CommandReplay{}, rollback(tx, ctx, fmt.Errorf("encoding idempotency response: %w", err))
	}
	if meta.ID != uuid.Nil {
		if len(body) == 0 {
			body = []byte(`{}`)
		}
		if err := insertIdempotencyRecord(ctx, tx, organisationID, locationID, actorRef, meta, status, body); err != nil {
			observability.RecordSpanError(span, err)
			return zero, CommandReplay{}, rollback(tx, ctx, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		observability.RecordSpanError(span, err)
		return zero, CommandReplay{}, fmt.Errorf("committing transaction: %w", err)
	}
	return value, CommandReplay{Status: status, Body: body}, nil
}

func TokenLookupParts(token string) (string, []byte, error) {
	token = strings.TrimSpace(token)
	if len(token) < guestTokenPrefixLength {
		return "", nil, fmt.Errorf("qr token must be at least %d characters", guestTokenPrefixLength)
	}
	sum := sha256.Sum256([]byte(token))
	return token[:guestTokenPrefixLength], sum[:], nil
}

func GuestActionsFromConfig(raw json.RawMessage) []GuestAction {
	type actionConfig struct {
		Key       string `json:"key"`
		Label     string `json:"label"`
		SortOrder int    `json:"sortOrder"`
		Enabled   *bool  `json:"enabled"`
	}
	var cfg struct {
		GuestActions []actionConfig `json:"guestActions"`
	}
	if len(bytes.TrimSpace(raw)) > 0 {
		_ = json.Unmarshal(raw, &cfg)
	}
	if len(cfg.GuestActions) == 0 {
		return defaultGuestActions()
	}
	actions := make([]GuestAction, 0, len(cfg.GuestActions))
	for _, item := range cfg.GuestActions {
		key := strings.TrimSpace(item.Key)
		label := strings.TrimSpace(item.Label)
		if !validGuestActionKey(key) || label == "" || (item.Enabled != nil && !*item.Enabled) {
			continue
		}
		actions = append(actions, GuestAction{
			Key:       key,
			Label:     label,
			SortOrder: item.SortOrder,
			Enabled:   true,
		})
	}
	sort.SliceStable(actions, func(i, j int) bool {
		if actions[i].SortOrder == actions[j].SortOrder {
			return actions[i].Label < actions[j].Label
		}
		return actions[i].SortOrder < actions[j].SortOrder
	})
	if len(actions) == 0 {
		return defaultGuestActions()
	}
	return actions
}

func guestActionEnabled(raw json.RawMessage, key string) bool {
	for _, action := range GuestActionsFromConfig(raw) {
		if action.Key == key {
			return true
		}
	}
	return false
}

func defaultGuestActions() []GuestAction {
	return []GuestAction{
		{Key: "call_waiter", Label: "Call waiter", SortOrder: 10, Enabled: true},
		{Key: "request_bill", Label: "Request bill", SortOrder: 20, Enabled: true},
		{Key: "request_water", Label: "Request water", SortOrder: 30, Enabled: true},
		{Key: "request_service", Label: "Request service", SortOrder: 40, Enabled: true},
	}
}

func validGuestActionKey(key string) bool {
	return guestActionKeyPattern.MatchString(key)
}

func qrExportFromDB(capability db.TableQrCapability, token string) QRExport {
	return QRExport{
		ID:           capability.ID,
		TableID:      capability.TableID,
		Token:        token,
		LookupPrefix: capability.TokenLookupPrefix,
		Label:        qrLabel(capability.Label),
		ExpiresAt:    capability.ExpiresAt,
		IssuedAt:     capability.IssuedAt,
		RevokedAt:    capability.RevokedAt,
		Version:      capability.Version,
	}
}

func qrLabel(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
