package integrations

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/domain/operations"
	"github.com/kadebhug/seatd_v2/internal/observability"
	"github.com/kadebhug/seatd_v2/internal/store/db"
	"go.opentelemetry.io/otel/attribute"
)

const (
	VendorReferencePOS = "reference_pos"

	StatusConnected    = "connected"
	StatusDisconnected = "disconnected"
	StatusDegraded     = "degraded"

	MappingMapped   = "mapped"
	MappingUnmapped = "unmapped"
	MappingIgnored  = "ignored"

	WebhookReceived  = "received"
	WebhookProcessed = "processed"
	WebhookFailed    = "failed"
	WebhookSkipped   = "skipped"

	ResolutionAutoCorrected = "auto_corrected"
	ResolutionReview        = "review_required"
	ResolutionIgnored       = "ignored"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("not found")
)

type Adapter interface {
	Vendor() string
	HandleWebhook(ctx context.Context, connection Connection, payload []byte, signature string) (WebhookFact, bool, error)
	FetchExternalStatus(ctx context.Context, connection Connection) ([]ExternalTableState, error)
	CheckHealth(ctx context.Context, connection Connection) (Health, error)
}

type Service struct {
	pool     *pgxpool.Pool
	ops      *operations.Service
	adapters map[string]Adapter
}

func NewService(pool *pgxpool.Pool, ops *operations.Service) *Service {
	reference := ReferencePOSAdapter{}
	return &Service{
		pool: pool,
		ops:  ops,
		adapters: map[string]Adapter{
			reference.Vendor(): reference,
		},
	}
}

type TenantActor struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	ActorRef       string
}

type Connection struct {
	ID                   uuid.UUID       `json:"id"`
	OrganisationID       uuid.UUID       `json:"organisationId"`
	LocationID           uuid.UUID       `json:"locationId"`
	Vendor               string          `json:"vendor"`
	DisplayName          string          `json:"displayName"`
	CredentialRef        string          `json:"credentialRef"`
	Status               string          `json:"status"`
	Capabilities         json.RawMessage `json:"capabilities"`
	Config               json.RawMessage `json:"config"`
	LastSuccessfulSyncAt *time.Time      `json:"lastSuccessfulSyncAt,omitempty"`
	LastError            *string         `json:"lastError,omitempty"`
	CreatedAt            time.Time       `json:"createdAt"`
	UpdatedAt            time.Time       `json:"updatedAt"`
}

type Mapping struct {
	ID              uuid.UUID  `json:"id"`
	ConnectionID    uuid.UUID  `json:"connectionId"`
	ExternalTableID string     `json:"externalTableId"`
	TableID         *uuid.UUID `json:"tableId,omitempty"`
	ExternalLabel   *string    `json:"externalLabel,omitempty"`
	Status          string     `json:"status"`
	LastSeenAt      *time.Time `json:"lastSeenAt,omitempty"`
}

type Webhook struct {
	ID              uuid.UUID       `json:"id"`
	ConnectionID    uuid.UUID       `json:"connectionId"`
	Vendor          string          `json:"vendor"`
	ExternalEventID string          `json:"externalEventId"`
	ReceivedAt      time.Time       `json:"receivedAt"`
	SignatureValid  bool            `json:"signatureValid"`
	Payload         json.RawMessage `json:"payload"`
	ProcessingState string          `json:"processingState"`
	Attempts        int32           `json:"attempts"`
	LastError       *string         `json:"lastError,omitempty"`
	ProcessedAt     *time.Time      `json:"processedAt,omitempty"`
}

type Discrepancy struct {
	ID              uuid.UUID       `json:"id"`
	ConnectionID    uuid.UUID       `json:"connectionId"`
	RunID           uuid.UUID       `json:"reconciliationRunId"`
	MappingID       *uuid.UUID      `json:"mappingId,omitempty"`
	TableID         *uuid.UUID      `json:"tableId,omitempty"`
	ExternalTableID string          `json:"externalTableId"`
	Type            string          `json:"type"`
	SeatdState      *string         `json:"seatdState,omitempty"`
	ExternalState   *string         `json:"externalState,omitempty"`
	ResolutionState string          `json:"resolutionState"`
	Details         json.RawMessage `json:"details"`
	CreatedAt       time.Time       `json:"createdAt"`
}

type ReconciliationRun struct {
	ID                 uuid.UUID  `json:"id"`
	ConnectionID       uuid.UUID  `json:"connectionId"`
	StartedAt          time.Time  `json:"startedAt"`
	CompletedAt        *time.Time `json:"completedAt,omitempty"`
	Status             string     `json:"status"`
	CheckedCount       int32      `json:"checkedCount"`
	DiscrepancyCount   int32      `json:"discrepancyCount"`
	AutoCorrectedCount int32      `json:"autoCorrectedCount"`
	LastError          *string    `json:"lastError,omitempty"`
}

type Health struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type WebhookFact struct {
	ExternalEventID string
	ExternalTableID string
	Status          string
	PartySize       *int32
	OccurredAt      *time.Time
}

type ExternalTableState struct {
	ExternalTableID string
	Status          string
	Label           string
}

type WebhookResult struct {
	Webhook Webhook
	Applied bool
}

func (s *Service) ListConnections(ctx context.Context, actor TenantActor) ([]Connection, error) {
	if err := actor.validate(true); err != nil {
		return nil, err
	}
	var connections []Connection
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionIntegrationsRead); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
SELECT id, organisation_id, location_id, vendor, display_name, credential_ref, status, capabilities, config,
       last_successful_sync_at, last_error, created_at, updated_at
FROM integration_connections
WHERE organisation_id = $1 AND location_id = $2
ORDER BY vendor, display_name
`, actor.OrganisationID, actor.LocationID)
		if err != nil {
			return fmt.Errorf("listing integration connections: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			connection, err := scanConnection(rows)
			if err != nil {
				return err
			}
			connections = append(connections, connection)
		}
		return rows.Err()
	})
	return connections, err
}

func (s *Service) ListMappings(ctx context.Context, actor TenantActor, connectionID uuid.UUID) ([]Mapping, error) {
	if err := actor.validate(true); err != nil || connectionID == uuid.Nil {
		return nil, ErrValidation
	}
	var mappings []Mapping
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionIntegrationsRead); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
SELECT id, connection_id, external_table_id, table_id, external_label, status, last_seen_at
FROM integration_table_mappings
WHERE organisation_id = $1 AND location_id = $2 AND connection_id = $3
ORDER BY status, external_table_id
`, actor.OrganisationID, actor.LocationID, connectionID)
		if err != nil {
			return fmt.Errorf("listing integration mappings: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			mapping, err := scanMapping(rows)
			if err != nil {
				return err
			}
			mappings = append(mappings, mapping)
		}
		return rows.Err()
	})
	return mappings, err
}

func (s *Service) UpdateMapping(ctx context.Context, actor TenantActor, mappingID uuid.UUID, tableID uuid.UUID, status string) (Mapping, error) {
	if err := actor.validate(true); err != nil || mappingID == uuid.Nil || !validMappingStatus(status) {
		return Mapping{}, ErrValidation
	}
	if status == MappingMapped && tableID == uuid.Nil {
		return Mapping{}, ErrValidation
	}
	var mapping Mapping
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionIntegrationsManage); err != nil {
			return err
		}
		row := tx.QueryRow(ctx, `
UPDATE integration_table_mappings
SET table_id = NULLIF($4::uuid, '00000000-0000-0000-0000-000000000000'::uuid),
    status = $5,
    updated_at = now()
WHERE id = $1 AND organisation_id = $2 AND location_id = $3
RETURNING id, connection_id, external_table_id, table_id, external_label, status, last_seen_at
`, mappingID, actor.OrganisationID, actor.LocationID, tableID, status)
		var err error
		mapping, err = scanMapping(row)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return mapping, err
}

func (s *Service) ListWebhooks(ctx context.Context, actor TenantActor, connectionID uuid.UUID) ([]Webhook, error) {
	if err := actor.validate(true); err != nil || connectionID == uuid.Nil {
		return nil, ErrValidation
	}
	var webhooks []Webhook
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionIntegrationsRead); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
SELECT id, connection_id, vendor, external_event_id, received_at, signature_valid, payload,
       processing_state, attempts, last_error, processed_at
FROM integration_webhook_inbox
WHERE organisation_id = $1 AND location_id = $2 AND connection_id = $3
ORDER BY received_at DESC
LIMIT 50
`, actor.OrganisationID, actor.LocationID, connectionID)
		if err != nil {
			return fmt.Errorf("listing integration webhooks: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			webhook, err := scanWebhook(rows)
			if err != nil {
				return err
			}
			webhooks = append(webhooks, webhook)
		}
		return rows.Err()
	})
	return webhooks, err
}

func (s *Service) ListDiscrepancies(ctx context.Context, actor TenantActor, connectionID uuid.UUID) ([]Discrepancy, error) {
	if err := actor.validate(true); err != nil || connectionID == uuid.Nil {
		return nil, ErrValidation
	}
	var discrepancies []Discrepancy
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionIntegrationsRead); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
SELECT id, connection_id, reconciliation_run_id, mapping_id, table_id, external_table_id,
       discrepancy_type, seatd_state, external_state, resolution_state, details, created_at
FROM integration_discrepancies
WHERE organisation_id = $1 AND location_id = $2 AND connection_id = $3
ORDER BY created_at DESC
LIMIT 100
`, actor.OrganisationID, actor.LocationID, connectionID)
		if err != nil {
			return fmt.Errorf("listing integration discrepancies: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			discrepancy, err := scanDiscrepancy(rows)
			if err != nil {
				return err
			}
			discrepancies = append(discrepancies, discrepancy)
		}
		return rows.Err()
	})
	return discrepancies, err
}

func (s *Service) Health(ctx context.Context, actor TenantActor, connectionID uuid.UUID) (Health, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.health",
		attribute.String("seatd.connection_id", connectionID.String()),
	)
	defer span.End()
	connection, err := s.getConnectionForActor(ctx, actor, connectionID, identity.PermissionIntegrationsRead)
	if err != nil {
		observability.RecordSpanError(span, err)
		return Health{}, err
	}
	adapter, err := s.adapter(connection.Vendor)
	if err != nil {
		observability.RecordSpanError(span, err)
		return Health{}, err
	}
	health, err := adapter.CheckHealth(ctx, connection)
	observability.RecordSpanError(span, err)
	return health, err
}

func (s *Service) ReceiveWebhook(ctx context.Context, vendor string, connectionID uuid.UUID, signature string, payload []byte) (WebhookResult, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.receive_webhook",
		attribute.String("seatd.vendor", strings.TrimSpace(vendor)),
		attribute.String("seatd.connection_id", connectionID.String()),
	)
	defer span.End()
	vendor = strings.TrimSpace(vendor)
	if vendor == "" || connectionID == uuid.Nil || len(payload) == 0 {
		observability.RecordSpanError(span, ErrValidation)
		return WebhookResult{}, ErrValidation
	}
	connection, err := s.getConnectionByID(ctx, connectionID)
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	if connection.Vendor != vendor {
		observability.RecordSpanError(span, ErrNotFound)
		return WebhookResult{}, ErrNotFound
	}
	adapter, err := s.adapter(vendor)
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	fact, signatureValid, err := adapter.HandleWebhook(ctx, connection, payload, signature)
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	span.SetAttributes(attribute.Bool("seatd.signature_valid", signatureValid))
	webhook, inserted, err := s.insertWebhook(ctx, connection, fact.ExternalEventID, signatureValid, payload)
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	if !inserted || webhook.ProcessingState == WebhookProcessed {
		return WebhookResult{Webhook: webhook}, nil
	}
	if !signatureValid {
		webhook, err = s.markWebhook(ctx, webhook.ID, WebhookSkipped, "signature validation failed")
		observability.RecordSpanError(span, err)
		return WebhookResult{Webhook: webhook}, err
	}
	if err := s.applyWebhookFact(ctx, connection, fact); err != nil {
		observability.RecordSpanError(span, err)
		webhook, markErr := s.markWebhook(ctx, webhook.ID, WebhookFailed, err.Error())
		if markErr != nil {
			return WebhookResult{}, errors.Join(err, markErr)
		}
		return WebhookResult{Webhook: webhook}, nil
	}
	webhook, err = s.markWebhook(ctx, webhook.ID, WebhookProcessed, "")
	observability.RecordSpanError(span, err)
	return WebhookResult{Webhook: webhook, Applied: true}, err
}

func (s *Service) ReplayWebhook(ctx context.Context, actor TenantActor, webhookID uuid.UUID) (WebhookResult, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.replay_webhook",
		attribute.String("seatd.webhook_id", webhookID.String()),
	)
	defer span.End()
	if err := actor.validate(true); err != nil || webhookID == uuid.Nil {
		observability.RecordSpanError(span, ErrValidation)
		return WebhookResult{}, ErrValidation
	}
	webhook, connection, err := s.getWebhookForActor(ctx, actor, webhookID, identity.PermissionIntegrationsManage)
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	adapter, err := s.adapter(connection.Vendor)
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	fact, signatureValid, err := adapter.HandleWebhook(ctx, connection, webhook.Payload, "")
	if err != nil {
		observability.RecordSpanError(span, err)
		return WebhookResult{}, err
	}
	if !signatureValid && !webhook.SignatureValid {
		webhook, err = s.markWebhook(ctx, webhook.ID, WebhookSkipped, "signature validation failed")
		observability.RecordSpanError(span, err)
		return WebhookResult{Webhook: webhook}, err
	}
	if err := s.applyWebhookFact(ctx, connection, fact); err != nil {
		observability.RecordSpanError(span, err)
		webhook, markErr := s.markWebhook(ctx, webhook.ID, WebhookFailed, err.Error())
		if markErr != nil {
			return WebhookResult{}, errors.Join(err, markErr)
		}
		return WebhookResult{Webhook: webhook}, nil
	}
	webhook, err = s.markWebhook(ctx, webhook.ID, WebhookProcessed, "")
	observability.RecordSpanError(span, err)
	return WebhookResult{Webhook: webhook, Applied: true}, err
}

func (s *Service) Reconcile(ctx context.Context, actor TenantActor, connectionID uuid.UUID) (ReconciliationRun, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.reconcile",
		attribute.String("seatd.connection_id", connectionID.String()),
	)
	defer span.End()
	connection, err := s.getConnectionForActor(ctx, actor, connectionID, identity.PermissionIntegrationsManage)
	if err != nil {
		observability.RecordSpanError(span, err)
		return ReconciliationRun{}, err
	}
	run, err := s.reconcileConnection(ctx, connection)
	observability.RecordSpanError(span, err)
	return run, err
}

func (s *Service) ReconcileConnected(ctx context.Context) (int, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.reconcile_connected")
	defer span.End()
	connections, err := s.listConnectedConnections(ctx)
	if err != nil {
		observability.RecordSpanError(span, err)
		return 0, err
	}
	span.SetAttributes(attribute.Int("seatd.connections", len(connections)))
	processed := 0
	for _, connection := range connections {
		if _, err := s.reconcileConnection(ctx, connection); err != nil {
			observability.RecordSpanError(span, err)
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (s *Service) reconcileConnection(ctx context.Context, connection Connection) (ReconciliationRun, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.reconcile_connection",
		attribute.String("seatd.connection_id", connection.ID.String()),
		attribute.String("seatd.vendor", connection.Vendor),
		attribute.String("seatd.organisation_id", connection.OrganisationID.String()),
		attribute.String("seatd.location_id", connection.LocationID.String()),
	)
	defer span.End()
	adapter, err := s.adapter(connection.Vendor)
	if err != nil {
		observability.RecordSpanError(span, err)
		return ReconciliationRun{}, err
	}
	run, err := s.createRun(ctx, connection)
	if err != nil {
		observability.RecordSpanError(span, err)
		return ReconciliationRun{}, err
	}
	span.SetAttributes(attribute.String("seatd.reconciliation_run_id", run.ID.String()))
	states, err := adapter.FetchExternalStatus(ctx, connection)
	if err != nil {
		observability.RecordSpanError(span, err)
		_, _ = s.finishRun(ctx, run.ID, "failed", 0, 0, 0, err.Error())
		return ReconciliationRun{}, err
	}
	span.SetAttributes(attribute.Int("seatd.external_states", len(states)))
	var discrepancyCount int32
	var correctedCount int32
	for _, state := range states {
		resolution, hadDiscrepancy, corrected, err := s.reconcileTableState(ctx, connection, run.ID, state)
		if err != nil {
			observability.RecordSpanError(span, err)
			_, _ = s.finishRun(ctx, run.ID, "failed", int32(len(states)), discrepancyCount, correctedCount, err.Error())
			return ReconciliationRun{}, err
		}
		if hadDiscrepancy {
			discrepancyCount++
		}
		if corrected {
			correctedCount++
		}
		_ = resolution
	}
	finished, err := s.finishRun(ctx, run.ID, "completed", int32(len(states)), discrepancyCount, correctedCount, "")
	observability.RecordSpanError(span, err)
	return finished, err
}

func (s *Service) reconcileTableState(ctx context.Context, connection Connection, runID uuid.UUID, state ExternalTableState) (string, bool, bool, error) {
	ctx, span := observability.StartSpan(ctx, "integrations.reconcile_table_state",
		attribute.String("seatd.connection_id", connection.ID.String()),
		attribute.String("seatd.external_table_id", state.ExternalTableID),
	)
	defer span.End()
	mapping, err := s.mappingByExternalID(ctx, connection, state.ExternalTableID, state.Label)
	if err != nil {
		observability.RecordSpanError(span, err)
		return "", false, false, err
	}
	if mapping.TableID == nil || mapping.Status != MappingMapped {
		_, err := s.recordDiscrepancy(ctx, connection, runID, mapping, nil, state.ExternalTableID, "unmapped_table", nil, &state.Status, ResolutionReview, nil)
		observability.RecordSpanError(span, err)
		return ResolutionReview, true, false, err
	}
	current, err := s.currentTableState(ctx, connection, *mapping.TableID)
	if err != nil {
		observability.RecordSpanError(span, err)
		return "", false, false, err
	}
	if current.Status == state.Status {
		return ResolutionIgnored, false, false, nil
	}
	hasAssists, err := s.hasActiveAssists(ctx, connection, *mapping.TableID)
	if err != nil {
		observability.RecordSpanError(span, err)
		return "", false, false, err
	}
	resolution := ResolutionAutoCorrected
	discrepancyType := "occupancy_mismatch"
	if hasAssists {
		resolution = ResolutionReview
		discrepancyType = "ambiguous_conflict"
	}
	_, err = s.recordDiscrepancy(ctx, connection, runID, mapping, mapping.TableID, state.ExternalTableID, discrepancyType, &current.Status, &state.Status, resolution, nil)
	if err != nil || resolution != ResolutionAutoCorrected {
		observability.RecordSpanError(span, err)
		return resolution, true, false, err
	}
	err = s.applyTableState(ctx, connection, *mapping.TableID, current.Version, state.Status, nil)
	observability.RecordSpanError(span, err)
	return resolution, true, err == nil, err
}

func (s *Service) applyWebhookFact(ctx context.Context, connection Connection, fact WebhookFact) error {
	ctx, span := observability.StartSpan(ctx, "integrations.apply_webhook_fact",
		attribute.String("seatd.connection_id", connection.ID.String()),
		attribute.String("seatd.external_table_id", fact.ExternalTableID),
		attribute.String("seatd.next_status", fact.Status),
	)
	defer span.End()
	mapping, err := s.mappingByExternalID(ctx, connection, fact.ExternalTableID, "")
	if err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	if mapping.TableID == nil || mapping.Status != MappingMapped {
		err := fmt.Errorf("external table %s is not mapped", fact.ExternalTableID)
		observability.RecordSpanError(span, err)
		return err
	}
	current, err := s.currentTableState(ctx, connection, *mapping.TableID)
	if err != nil {
		observability.RecordSpanError(span, err)
		return err
	}
	if current.Status == fact.Status {
		return nil
	}
	err = s.applyTableState(ctx, connection, *mapping.TableID, current.Version, fact.Status, fact.PartySize)
	observability.RecordSpanError(span, err)
	return err
}

func (s *Service) applyTableState(ctx context.Context, connection Connection, tableID uuid.UUID, expectedVersion int32, status string, partySize *int32) error {
	ctx, span := observability.StartSpan(ctx, "integrations.apply_table_state",
		attribute.String("seatd.connection_id", connection.ID.String()),
		attribute.String("seatd.table_id", tableID.String()),
		attribute.String("seatd.next_status", status),
	)
	defer span.End()
	actor := "integration:" + connection.ID.String()
	command := operations.CommandMeta{
		ID:          uuid.New(),
		Type:        "integration.table_state",
		PayloadHash: commandHash(connection.ID, tableID, status),
	}
	switch status {
	case operations.TableStatusOccupied:
		_, err := s.ops.OccupyTable(ctx, operations.OccupyTableParams{
			OrganisationID:  connection.OrganisationID,
			LocationID:      connection.LocationID,
			TableID:         tableID,
			ExpectedVersion: expectedVersion,
			PartySize:       partySize,
			ActorRef:        actor,
			Source:          "integration:" + connection.Vendor,
			Command:         command,
		})
		if err != nil && !errors.Is(err, operations.ErrAlreadyOccupied) {
			observability.RecordSpanError(span, err)
			return err
		}
	case operations.TableStatusAvailable:
		_, err := s.ops.ClearTable(ctx, operations.ClearTableParams{
			OrganisationID:  connection.OrganisationID,
			LocationID:      connection.LocationID,
			TableID:         tableID,
			ExpectedVersion: expectedVersion,
			ActorRef:        actor,
			Command:         command,
		})
		if err != nil && !errors.Is(err, operations.ErrAlreadyAvailable) {
			observability.RecordSpanError(span, err)
			return err
		}
	default:
		observability.RecordSpanError(span, ErrValidation)
		return ErrValidation
	}
	return nil
}

func (s *Service) adapter(vendor string) (Adapter, error) {
	adapter := s.adapters[vendor]
	if adapter == nil {
		return nil, ErrNotFound
	}
	return adapter, nil
}

func commandHash(parts ...any) []byte {
	sum := sha256.Sum256([]byte(fmt.Sprint(parts...)))
	return sum[:]
}

func validMappingStatus(status string) bool {
	return status == MappingMapped || status == MappingUnmapped || status == MappingIgnored
}

func (actor TenantActor) validate(requireLocation bool) error {
	if actor.OrganisationID == uuid.Nil || strings.TrimSpace(actor.ActorRef) == "" {
		return ErrValidation
	}
	if requireLocation && actor.LocationID == uuid.Nil {
		return ErrValidation
	}
	return nil
}

func requireLocationPermission(ctx context.Context, q *db.Queries, actor TenantActor, permission string) error {
	allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
		OrganisationID: actor.OrganisationID,
		LocationID:     actor.LocationID,
		MemberRef:      actor.ActorRef,
		PermissionName: permission,
	})
	if err != nil {
		return fmt.Errorf("checking location permission: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) inTenantTx(ctx context.Context, actor TenantActor, fn func(pgx.Tx, *db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, actor.OrganisationID.String(), actor.LocationID.String()); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	if err := fn(tx, db.New(tx)); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (s *Service) inAdminTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning admin transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting admin context: %w", err))
	}
	if err := fn(tx); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing admin transaction: %w", err)
	}
	return nil
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}

func nullableText(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func timePtr(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanConnection(row rowScanner) (Connection, error) {
	var connection Connection
	var lastSync pgtype.Timestamptz
	var lastError pgtype.Text
	if err := row.Scan(
		&connection.ID,
		&connection.OrganisationID,
		&connection.LocationID,
		&connection.Vendor,
		&connection.DisplayName,
		&connection.CredentialRef,
		&connection.Status,
		&connection.Capabilities,
		&connection.Config,
		&lastSync,
		&lastError,
		&connection.CreatedAt,
		&connection.UpdatedAt,
	); err != nil {
		return Connection{}, err
	}
	connection.LastSuccessfulSyncAt = timePtr(lastSync)
	connection.LastError = textPtr(lastError)
	return connection, nil
}

func scanMapping(row rowScanner) (Mapping, error) {
	var mapping Mapping
	var tableID uuid.NullUUID
	var label pgtype.Text
	var lastSeen pgtype.Timestamptz
	if err := row.Scan(&mapping.ID, &mapping.ConnectionID, &mapping.ExternalTableID, &tableID, &label, &mapping.Status, &lastSeen); err != nil {
		return Mapping{}, err
	}
	if tableID.Valid {
		mapping.TableID = &tableID.UUID
	}
	mapping.ExternalLabel = textPtr(label)
	mapping.LastSeenAt = timePtr(lastSeen)
	return mapping, nil
}

func scanWebhook(row rowScanner) (Webhook, error) {
	var webhook Webhook
	var lastError pgtype.Text
	var processedAt pgtype.Timestamptz
	if err := row.Scan(
		&webhook.ID,
		&webhook.ConnectionID,
		&webhook.Vendor,
		&webhook.ExternalEventID,
		&webhook.ReceivedAt,
		&webhook.SignatureValid,
		&webhook.Payload,
		&webhook.ProcessingState,
		&webhook.Attempts,
		&lastError,
		&processedAt,
	); err != nil {
		return Webhook{}, err
	}
	webhook.LastError = textPtr(lastError)
	webhook.ProcessedAt = timePtr(processedAt)
	return webhook, nil
}

func scanDiscrepancy(row rowScanner) (Discrepancy, error) {
	var discrepancy Discrepancy
	var mappingID uuid.NullUUID
	var tableID uuid.NullUUID
	var seatdState pgtype.Text
	var externalState pgtype.Text
	if err := row.Scan(
		&discrepancy.ID,
		&discrepancy.ConnectionID,
		&discrepancy.RunID,
		&mappingID,
		&tableID,
		&discrepancy.ExternalTableID,
		&discrepancy.Type,
		&seatdState,
		&externalState,
		&discrepancy.ResolutionState,
		&discrepancy.Details,
		&discrepancy.CreatedAt,
	); err != nil {
		return Discrepancy{}, err
	}
	if mappingID.Valid {
		discrepancy.MappingID = &mappingID.UUID
	}
	if tableID.Valid {
		discrepancy.TableID = &tableID.UUID
	}
	discrepancy.SeatdState = textPtr(seatdState)
	discrepancy.ExternalState = textPtr(externalState)
	return discrepancy, nil
}

type tableState struct {
	Status  string
	Version int32
}

func (s *Service) getConnectionForActor(ctx context.Context, actor TenantActor, connectionID uuid.UUID, permission string) (Connection, error) {
	if err := actor.validate(true); err != nil || connectionID == uuid.Nil {
		return Connection{}, ErrValidation
	}
	var connection Connection
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, permission); err != nil {
			return err
		}
		var err error
		connection, err = scanConnection(tx.QueryRow(ctx, `
SELECT id, organisation_id, location_id, vendor, display_name, credential_ref, status, capabilities, config,
       last_successful_sync_at, last_error, created_at, updated_at
FROM integration_connections
WHERE id = $1 AND organisation_id = $2 AND location_id = $3
`, connectionID, actor.OrganisationID, actor.LocationID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return connection, err
}

func (s *Service) getConnectionByID(ctx context.Context, connectionID uuid.UUID) (Connection, error) {
	var connection Connection
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		var err error
		connection, err = scanConnection(tx.QueryRow(ctx, `
SELECT id, organisation_id, location_id, vendor, display_name, credential_ref, status, capabilities, config,
       last_successful_sync_at, last_error, created_at, updated_at
FROM integration_connections
WHERE id = $1
`, connectionID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return connection, err
}

func (s *Service) listConnectedConnections(ctx context.Context) ([]Connection, error) {
	var connections []Connection
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
SELECT id, organisation_id, location_id, vendor, display_name, credential_ref, status, capabilities, config,
       last_successful_sync_at, last_error, created_at, updated_at
FROM integration_connections
WHERE status IN ('connected', 'degraded')
ORDER BY updated_at
`)
		if err != nil {
			return fmt.Errorf("listing connected integrations: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			connection, err := scanConnection(rows)
			if err != nil {
				return err
			}
			connections = append(connections, connection)
		}
		return rows.Err()
	})
	return connections, err
}

func (s *Service) insertWebhook(ctx context.Context, connection Connection, externalEventID string, signatureValid bool, payload []byte) (Webhook, bool, error) {
	var webhook Webhook
	inserted := true
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
INSERT INTO integration_webhook_inbox (
    organisation_id, location_id, connection_id, vendor, external_event_id, signature_valid, payload
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (vendor, external_event_id) DO NOTHING
RETURNING id, connection_id, vendor, external_event_id, received_at, signature_valid, payload,
          processing_state, attempts, last_error, processed_at
`, connection.OrganisationID, connection.LocationID, connection.ID, connection.Vendor, externalEventID, signatureValid, payload)
		var err error
		webhook, err = scanWebhook(row)
		if errors.Is(err, pgx.ErrNoRows) {
			inserted = false
			webhook, err = s.getWebhookByExternalEventID(ctx, tx, connection.Vendor, externalEventID)
		}
		return err
	})
	return webhook, inserted, err
}

func (s *Service) getWebhookByExternalEventID(ctx context.Context, tx pgx.Tx, vendor, externalEventID string) (Webhook, error) {
	return scanWebhook(tx.QueryRow(ctx, `
SELECT id, connection_id, vendor, external_event_id, received_at, signature_valid, payload,
       processing_state, attempts, last_error, processed_at
FROM integration_webhook_inbox
WHERE vendor = $1 AND external_event_id = $2
`, vendor, externalEventID))
}

func (s *Service) getWebhookForActor(ctx context.Context, actor TenantActor, webhookID uuid.UUID, permission string) (Webhook, Connection, error) {
	var webhook Webhook
	connection, err := s.getConnectionForWebhook(ctx, actor, webhookID, permission)
	if err != nil {
		return Webhook{}, Connection{}, err
	}
	err = s.inTenantTx(ctx, actor, func(tx pgx.Tx, _ *db.Queries) error {
		var err error
		webhook, err = scanWebhook(tx.QueryRow(ctx, `
SELECT id, connection_id, vendor, external_event_id, received_at, signature_valid, payload,
       processing_state, attempts, last_error, processed_at
FROM integration_webhook_inbox
WHERE id = $1 AND organisation_id = $2 AND location_id = $3
`, webhookID, actor.OrganisationID, actor.LocationID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return webhook, connection, err
}

func (s *Service) getConnectionForWebhook(ctx context.Context, actor TenantActor, webhookID uuid.UUID, permission string) (Connection, error) {
	if err := actor.validate(true); err != nil {
		return Connection{}, err
	}
	var connection Connection
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, permission); err != nil {
			return err
		}
		var err error
		connection, err = scanConnection(tx.QueryRow(ctx, `
SELECT c.id, c.organisation_id, c.location_id, c.vendor, c.display_name, c.credential_ref, c.status, c.capabilities, c.config,
       c.last_successful_sync_at, c.last_error, c.created_at, c.updated_at
FROM integration_connections c
JOIN integration_webhook_inbox w ON w.connection_id = c.id
WHERE w.id = $1 AND c.organisation_id = $2 AND c.location_id = $3
`, webhookID, actor.OrganisationID, actor.LocationID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	})
	return connection, err
}

func (s *Service) markWebhook(ctx context.Context, webhookID uuid.UUID, state string, lastError string) (Webhook, error) {
	var webhook Webhook
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		lastErrorArg := pgtype.Text{String: lastError, Valid: lastError != ""}
		row := tx.QueryRow(ctx, `
UPDATE integration_webhook_inbox
SET processing_state = $2,
    attempts = attempts + 1,
    last_error = $3,
    processed_at = CASE WHEN $2 = 'processed' THEN now() ELSE processed_at END,
    updated_at = now()
WHERE id = $1
RETURNING id, connection_id, vendor, external_event_id, received_at, signature_valid, payload,
          processing_state, attempts, last_error, processed_at
`, webhookID, state, lastErrorArg)
		var err error
		webhook, err = scanWebhook(row)
		if err != nil {
			return err
		}
		if state == WebhookProcessed {
			if _, err := tx.Exec(ctx, `
UPDATE integration_connections
SET last_successful_sync_at = now(),
    last_error = NULL,
    status = 'connected',
    updated_at = now()
WHERE id = $1
`, webhook.ConnectionID); err != nil {
				return fmt.Errorf("marking integration webhook sync successful: %w", err)
			}
		}
		return err
	})
	return webhook, err
}

func (s *Service) mappingByExternalID(ctx context.Context, connection Connection, externalTableID string, externalLabel string) (Mapping, error) {
	var mapping Mapping
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
INSERT INTO integration_table_mappings (
    organisation_id, location_id, connection_id, external_table_id, external_label, status, last_seen_at
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), 'unmapped', now())
ON CONFLICT (connection_id, external_table_id) DO UPDATE
SET external_label = COALESCE(NULLIF(EXCLUDED.external_label, ''), integration_table_mappings.external_label),
    last_seen_at = now(),
    updated_at = now()
RETURNING id, connection_id, external_table_id, table_id, external_label, status, last_seen_at
`, connection.OrganisationID, connection.LocationID, connection.ID, externalTableID, externalLabel)
		var err error
		mapping, err = scanMapping(row)
		return err
	})
	return mapping, err
}

func (s *Service) currentTableState(ctx context.Context, connection Connection, tableID uuid.UUID) (tableState, error) {
	var state tableState
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
SELECT status, version
FROM table_occupancy
WHERE organisation_id = $1 AND location_id = $2 AND table_id = $3
`, connection.OrganisationID, connection.LocationID, tableID).Scan(&state.Status, &state.Version)
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return tableState{}, ErrNotFound
	}
	return state, err
}

func (s *Service) hasActiveAssists(ctx context.Context, connection Connection, tableID uuid.UUID) (bool, error) {
	var exists bool
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
SELECT EXISTS (
    SELECT 1
    FROM assist_requests
    WHERE organisation_id = $1
      AND location_id = $2
      AND table_id = $3
      AND status IN ('pending', 'acknowledged')
)
`, connection.OrganisationID, connection.LocationID, tableID).Scan(&exists)
	})
	return exists, err
}

func (s *Service) createRun(ctx context.Context, connection Connection) (ReconciliationRun, error) {
	var runID uuid.UUID
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
INSERT INTO integration_reconciliation_runs (organisation_id, location_id, connection_id)
VALUES ($1, $2, $3)
RETURNING id
`, connection.OrganisationID, connection.LocationID, connection.ID).Scan(&runID)
	})
	if err != nil {
		return ReconciliationRun{}, err
	}
	return s.scanRun(ctx, runID)
}

func (s *Service) scanRun(ctx context.Context, runID uuid.UUID) (ReconciliationRun, error) {
	var run ReconciliationRun
	var completedAt pgtype.Timestamptz
	var lastError pgtype.Text
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
SELECT id, connection_id, started_at, completed_at, status, checked_count, discrepancy_count,
       auto_corrected_count, last_error
FROM integration_reconciliation_runs
WHERE id = $1
`, runID).Scan(&run.ID, &run.ConnectionID, &run.StartedAt, &completedAt, &run.Status, &run.CheckedCount, &run.DiscrepancyCount, &run.AutoCorrectedCount, &lastError)
	})
	run.CompletedAt = timePtr(completedAt)
	run.LastError = textPtr(lastError)
	return run, err
}

func (s *Service) finishRun(ctx context.Context, runID uuid.UUID, status string, checkedCount, discrepancyCount, correctedCount int32, lastError string) (ReconciliationRun, error) {
	var run ReconciliationRun
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		lastErrorArg := pgtype.Text{String: lastError, Valid: lastError != ""}
		var completedAt pgtype.Timestamptz
		var lastErrorOut pgtype.Text
		err := tx.QueryRow(ctx, `
UPDATE integration_reconciliation_runs
SET status = $2,
    completed_at = now(),
    checked_count = $3,
    discrepancy_count = $4,
    auto_corrected_count = $5,
    last_error = $6,
    updated_at = now()
WHERE id = $1
RETURNING id, connection_id, started_at, completed_at, status, checked_count, discrepancy_count,
          auto_corrected_count, last_error
`, runID, status, checkedCount, discrepancyCount, correctedCount, lastErrorArg).Scan(&run.ID, &run.ConnectionID, &run.StartedAt, &completedAt, &run.Status, &run.CheckedCount, &run.DiscrepancyCount, &run.AutoCorrectedCount, &lastErrorOut)
		if err != nil {
			return err
		}
		if status == "completed" {
			if _, err := tx.Exec(ctx, `
UPDATE integration_connections
SET last_successful_sync_at = now(),
    last_error = NULL,
    status = 'connected',
    updated_at = now()
WHERE id = $1
`, run.ConnectionID); err != nil {
				return fmt.Errorf("marking integration reconciliation successful: %w", err)
			}
			return nil
		}
		if status == "failed" {
			if _, err := tx.Exec(ctx, `
UPDATE integration_connections
SET status = 'degraded',
    last_error = $2,
    updated_at = now()
WHERE id = $1
`, run.ConnectionID, lastErrorArg); err != nil {
				return fmt.Errorf("marking integration reconciliation failed: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return ReconciliationRun{}, err
	}
	return s.scanRun(ctx, run.ID)
}

func (s *Service) recordDiscrepancy(
	ctx context.Context,
	connection Connection,
	runID uuid.UUID,
	mapping Mapping,
	tableID *uuid.UUID,
	externalTableID string,
	discrepancyType string,
	seatdState *string,
	externalState *string,
	resolution string,
	details json.RawMessage,
) (Discrepancy, error) {
	if len(details) == 0 {
		details = []byte(`{}`)
	}
	var discrepancy Discrepancy
	err := s.inAdminTx(ctx, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
INSERT INTO integration_discrepancies (
    organisation_id, location_id, connection_id, reconciliation_run_id, mapping_id, table_id,
    external_table_id, discrepancy_type, seatd_state, external_state, resolution_state, details
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
RETURNING id, connection_id, reconciliation_run_id, mapping_id, table_id, external_table_id,
          discrepancy_type, seatd_state, external_state, resolution_state, details, created_at
`, connection.OrganisationID, connection.LocationID, connection.ID, runID, mapping.ID, tableID, externalTableID, discrepancyType, nullableText(seatdState), nullableText(externalState), resolution, details)
		var err error
		discrepancy, err = scanDiscrepancy(row)
		return err
	})
	return discrepancy, err
}

type ReferencePOSAdapter struct{}

func (ReferencePOSAdapter) Vendor() string {
	return VendorReferencePOS
}

func (ReferencePOSAdapter) HandleWebhook(_ context.Context, _ Connection, payload []byte, signature string) (WebhookFact, bool, error) {
	var body struct {
		EventID         string     `json:"eventId"`
		ExternalTableID string     `json:"externalTableId"`
		Status          string     `json:"status"`
		PartySize       *int32     `json:"partySize"`
		OccurredAt      *time.Time `json:"occurredAt"`
	}
	if err := json.Unmarshal(payload, &body); err != nil {
		return WebhookFact{}, false, ErrValidation
	}
	body.EventID = strings.TrimSpace(body.EventID)
	body.ExternalTableID = strings.TrimSpace(body.ExternalTableID)
	if body.EventID == "" || body.ExternalTableID == "" || !validTableStatus(body.Status) {
		return WebhookFact{}, false, ErrValidation
	}
	valid := verifyReferenceSignature(payload, signature)
	return WebhookFact{
		ExternalEventID: body.EventID,
		ExternalTableID: body.ExternalTableID,
		Status:          body.Status,
		PartySize:       body.PartySize,
		OccurredAt:      body.OccurredAt,
	}, valid, nil
}

func (ReferencePOSAdapter) FetchExternalStatus(_ context.Context, connection Connection) ([]ExternalTableState, error) {
	var config struct {
		ReferenceTables []struct {
			ExternalTableID string `json:"externalTableId"`
			Status          string `json:"status"`
			Label           string `json:"label"`
		} `json:"referenceTables"`
	}
	if err := json.Unmarshal(connection.Config, &config); err != nil {
		return nil, ErrValidation
	}
	states := make([]ExternalTableState, 0, len(config.ReferenceTables))
	for _, table := range config.ReferenceTables {
		if strings.TrimSpace(table.ExternalTableID) == "" || !validTableStatus(table.Status) {
			continue
		}
		states = append(states, ExternalTableState{
			ExternalTableID: strings.TrimSpace(table.ExternalTableID),
			Status:          table.Status,
			Label:           strings.TrimSpace(table.Label),
		})
	}
	return states, nil
}

func (ReferencePOSAdapter) CheckHealth(_ context.Context, connection Connection) (Health, error) {
	if connection.Status == StatusDisconnected {
		return Health{Status: StatusDisconnected, Message: "connection is disabled"}, nil
	}
	return Health{Status: connection.Status}, nil
}

func verifyReferenceSignature(payload []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	expected := hmac.New(sha256.New, []byte(referenceSecret()))
	_, _ = expected.Write(payload)
	expectedHex := hex.EncodeToString(expected.Sum(nil))
	return hmac.Equal([]byte(expectedHex), []byte(signature))
}

func referenceSecret() string {
	secret := strings.TrimSpace(os.Getenv("SEATD_REFERENCE_POS_WEBHOOK_SECRET"))
	if secret == "" {
		return "reference-pos-local-secret"
	}
	return secret
}

func validTableStatus(status string) bool {
	return status == operations.TableStatusAvailable || status == operations.TableStatusOccupied
}
