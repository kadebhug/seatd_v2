package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kadebhug/seatd_v2/internal/domain/integrations"
)

const (
	headerIntegrationConnectionID = "X-Seatd-Integration-Connection-ID"
	headerIntegrationSignature    = "X-Seatd-Integration-Signature"
)

type integrationConnectionDTO struct {
	ID                   string          `json:"id"`
	OrganisationID       string          `json:"organisationId"`
	LocationID           string          `json:"locationId"`
	Vendor               string          `json:"vendor"`
	DisplayName          string          `json:"displayName"`
	CredentialRef        string          `json:"credentialRef"`
	Status               string          `json:"status"`
	Capabilities         json.RawMessage `json:"capabilities"`
	Config               json.RawMessage `json:"config"`
	LastSuccessfulSyncAt *string         `json:"lastSuccessfulSyncAt,omitempty"`
	LastError            *string         `json:"lastError,omitempty"`
	CreatedAt            string          `json:"createdAt"`
	UpdatedAt            string          `json:"updatedAt"`
}

type integrationMappingDTO struct {
	ID              string  `json:"id"`
	ConnectionID    string  `json:"connectionId"`
	ExternalTableID string  `json:"externalTableId"`
	TableID         *string `json:"tableId,omitempty"`
	ExternalLabel   *string `json:"externalLabel,omitempty"`
	Status          string  `json:"status"`
	LastSeenAt      *string `json:"lastSeenAt,omitempty"`
}

type integrationWebhookDTO struct {
	ID              string          `json:"id"`
	ConnectionID    string          `json:"connectionId"`
	Vendor          string          `json:"vendor"`
	ExternalEventID string          `json:"externalEventId"`
	ReceivedAt      string          `json:"receivedAt"`
	SignatureValid  bool            `json:"signatureValid"`
	Payload         json.RawMessage `json:"payload"`
	ProcessingState string          `json:"processingState"`
	Attempts        int32           `json:"attempts"`
	LastError       *string         `json:"lastError,omitempty"`
	ProcessedAt     *string         `json:"processedAt,omitempty"`
}

type integrationDiscrepancyDTO struct {
	ID                  string          `json:"id"`
	ConnectionID        string          `json:"connectionId"`
	ReconciliationRunID string          `json:"reconciliationRunId"`
	MappingID           *string         `json:"mappingId,omitempty"`
	TableID             *string         `json:"tableId,omitempty"`
	ExternalTableID     string          `json:"externalTableId"`
	Type                string          `json:"type"`
	SeatdState          *string         `json:"seatdState,omitempty"`
	ExternalState       *string         `json:"externalState,omitempty"`
	ResolutionState     string          `json:"resolutionState"`
	Details             json.RawMessage `json:"details"`
	CreatedAt           string          `json:"createdAt"`
}

type integrationRunDTO struct {
	ID                 string  `json:"id"`
	ConnectionID       string  `json:"connectionId"`
	StartedAt          string  `json:"startedAt"`
	CompletedAt        *string `json:"completedAt,omitempty"`
	Status             string  `json:"status"`
	CheckedCount       int32   `json:"checkedCount"`
	DiscrepancyCount   int32   `json:"discrepancyCount"`
	AutoCorrectedCount int32   `json:"autoCorrectedCount"`
	LastError          *string `json:"lastError,omitempty"`
}

type integrationHealthDTO struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type updateIntegrationMappingRequest struct {
	TableID *uuid.UUID `json:"tableId,omitempty"`
	Status  string     `json:"status"`
}

func (api *API) listIntegrations(w http.ResponseWriter, r *http.Request) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return
	}
	connections, err := api.ints.ListConnections(r.Context(), integrationActor(ctx))
	if err != nil {
		api.writeIntegrationError(w, err)
		return
	}
	out := make([]integrationConnectionDTO, 0, len(connections))
	for _, connection := range connections {
		out = append(out, integrationConnectionFromDomain(connection))
	}
	writeJSON(w, http.StatusOK, map[string]any{"integrations": out})
}

func (api *API) getIntegrationHealth(w http.ResponseWriter, r *http.Request) {
	ctx, id, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	health, err := api.ints.Health(r.Context(), integrationActor(ctx), id)
	if err != nil {
		api.writeIntegrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, integrationHealthDTO{Status: health.Status, Message: health.Message})
}

func (api *API) listIntegrationMappings(w http.ResponseWriter, r *http.Request) {
	ctx, id, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	mappings, err := api.ints.ListMappings(r.Context(), integrationActor(ctx), id)
	if err != nil {
		api.writeIntegrationError(w, err)
		return
	}
	out := make([]integrationMappingDTO, 0, len(mappings))
	for _, mapping := range mappings {
		out = append(out, integrationMappingFromDomain(mapping))
	}
	writeJSON(w, http.StatusOK, map[string]any{"mappings": out})
}

func (api *API) updateIntegrationMapping(w http.ResponseWriter, r *http.Request) {
	ctx, _, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	mappingID, ok := pathUUID(w, r, "mappingId")
	if !ok {
		return
	}
	var req updateIntegrationMappingRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	tableID := uuid.Nil
	if req.TableID != nil {
		tableID = *req.TableID
	}
	mapping, err := api.ints.UpdateMapping(r.Context(), integrationActor(ctx), mappingID, tableID, req.Status)
	if err != nil {
		api.writeIntegrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, integrationMappingFromDomain(mapping))
}

func (api *API) listIntegrationWebhooks(w http.ResponseWriter, r *http.Request) {
	ctx, id, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	webhooks, err := api.ints.ListWebhooks(r.Context(), integrationActor(ctx), id)
	if err != nil {
		api.writeIntegrationError(w, err)
		return
	}
	out := make([]integrationWebhookDTO, 0, len(webhooks))
	for _, webhook := range webhooks {
		out = append(out, integrationWebhookFromDomain(webhook))
	}
	writeJSON(w, http.StatusOK, map[string]any{"webhooks": out})
}

func (api *API) replayIntegrationWebhook(w http.ResponseWriter, r *http.Request) {
	ctx, _, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	webhookID, ok := pathUUID(w, r, "webhookId")
	if !ok {
		return
	}
	result, err := api.ints.ReplayWebhook(r.Context(), integrationActor(ctx), webhookID)
	if err != nil {
		if api.metrics != nil {
			api.metrics.ObserveIntegrationWebhook("", "replay_failed")
		}
		api.writeIntegrationError(w, err)
		return
	}
	if api.metrics != nil {
		api.metrics.ObserveIntegrationWebhook(result.Webhook.Vendor, "replay_"+result.Webhook.ProcessingState)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"webhook": integrationWebhookFromDomain(result.Webhook),
		"applied": result.Applied,
	})
}

func (api *API) listIntegrationDiscrepancies(w http.ResponseWriter, r *http.Request) {
	ctx, id, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	discrepancies, err := api.ints.ListDiscrepancies(r.Context(), integrationActor(ctx), id)
	if err != nil {
		api.writeIntegrationError(w, err)
		return
	}
	out := make([]integrationDiscrepancyDTO, 0, len(discrepancies))
	for _, discrepancy := range discrepancies {
		out = append(out, integrationDiscrepancyFromDomain(discrepancy))
	}
	writeJSON(w, http.StatusOK, map[string]any{"discrepancies": out})
}

func (api *API) reconcileIntegration(w http.ResponseWriter, r *http.Request) {
	ctx, id, ok := api.integrationInput(w, r)
	if !ok {
		return
	}
	run, err := api.ints.Reconcile(r.Context(), integrationActor(ctx), id)
	if err != nil {
		if api.metrics != nil {
			api.metrics.ObserveIntegrationReconciliation("failed", id.String(), 0)
		}
		api.writeIntegrationError(w, err)
		return
	}
	if api.metrics != nil {
		api.metrics.ObserveIntegrationReconciliation(run.Status, run.ConnectionID.String(), run.DiscrepancyCount)
	}
	writeJSON(w, http.StatusOK, map[string]any{"reconciliationRun": integrationRunFromDomain(run)})
}

func (api *API) receiveIntegrationWebhook(w http.ResponseWriter, r *http.Request) {
	connectionID, ok := parseHeaderUUID(w, r, headerIntegrationConnectionID, true)
	if !ok {
		return
	}
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer body.Close()
	payload, err := io.ReadAll(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "validation_failed", "request body must be valid JSON", nil)
		return
	}
	result, err := api.ints.ReceiveWebhook(
		r.Context(),
		strings.TrimSpace(r.PathValue("vendor")),
		connectionID,
		r.Header.Get(headerIntegrationSignature),
		payload,
	)
	if err != nil {
		if api.metrics != nil {
			api.metrics.ObserveIntegrationWebhook(strings.TrimSpace(r.PathValue("vendor")), "failed")
		}
		api.writeIntegrationError(w, err)
		return
	}
	if api.metrics != nil {
		api.metrics.ObserveIntegrationWebhook(result.Webhook.Vendor, result.Webhook.ProcessingState)
	}
	writeJSON(w, http.StatusAccepted, map[string]any{
		"webhook": integrationWebhookFromDomain(result.Webhook),
		"applied": result.Applied,
	})
}

func (api *API) integrationInput(w http.ResponseWriter, r *http.Request) (requestContext, uuid.UUID, bool) {
	ctx, ok := api.requestContext(w, r, true, true)
	if !ok {
		return requestContext{}, uuid.Nil, false
	}
	id, ok := pathUUID(w, r, "id")
	return ctx, id, ok
}

func integrationActor(ctx requestContext) integrations.TenantActor {
	return integrations.TenantActor{
		OrganisationID: ctx.OrganisationID,
		LocationID:     ctx.LocationID,
		ActorRef:       ctx.ActorRef,
	}
}

func (api *API) writeIntegrationError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, integrations.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
	case errors.Is(err, integrations.ErrValidation):
		writeError(w, http.StatusBadRequest, "validation_failed", "request validation failed", nil)
	case errors.Is(err, integrations.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("integration request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func integrationConnectionFromDomain(connection integrations.Connection) integrationConnectionDTO {
	return integrationConnectionDTO{
		ID:                   connection.ID.String(),
		OrganisationID:       connection.OrganisationID.String(),
		LocationID:           connection.LocationID.String(),
		Vendor:               connection.Vendor,
		DisplayName:          connection.DisplayName,
		CredentialRef:        connection.CredentialRef,
		Status:               connection.Status,
		Capabilities:         connection.Capabilities,
		Config:               connection.Config,
		LastSuccessfulSyncAt: timeValuePtr(connection.LastSuccessfulSyncAt),
		LastError:            connection.LastError,
		CreatedAt:            timeInstantString(connection.CreatedAt),
		UpdatedAt:            timeInstantString(connection.UpdatedAt),
	}
}

func integrationMappingFromDomain(mapping integrations.Mapping) integrationMappingDTO {
	return integrationMappingDTO{
		ID:              mapping.ID.String(),
		ConnectionID:    mapping.ConnectionID.String(),
		ExternalTableID: mapping.ExternalTableID,
		TableID:         uuidValuePtr(mapping.TableID),
		ExternalLabel:   mapping.ExternalLabel,
		Status:          mapping.Status,
		LastSeenAt:      timeValuePtr(mapping.LastSeenAt),
	}
}

func integrationWebhookFromDomain(webhook integrations.Webhook) integrationWebhookDTO {
	return integrationWebhookDTO{
		ID:              webhook.ID.String(),
		ConnectionID:    webhook.ConnectionID.String(),
		Vendor:          webhook.Vendor,
		ExternalEventID: webhook.ExternalEventID,
		ReceivedAt:      timeInstantString(webhook.ReceivedAt),
		SignatureValid:  webhook.SignatureValid,
		Payload:         webhook.Payload,
		ProcessingState: webhook.ProcessingState,
		Attempts:        webhook.Attempts,
		LastError:       webhook.LastError,
		ProcessedAt:     timeValuePtr(webhook.ProcessedAt),
	}
}

func integrationDiscrepancyFromDomain(discrepancy integrations.Discrepancy) integrationDiscrepancyDTO {
	return integrationDiscrepancyDTO{
		ID:                  discrepancy.ID.String(),
		ConnectionID:        discrepancy.ConnectionID.String(),
		ReconciliationRunID: discrepancy.RunID.String(),
		MappingID:           uuidValuePtr(discrepancy.MappingID),
		TableID:             uuidValuePtr(discrepancy.TableID),
		ExternalTableID:     discrepancy.ExternalTableID,
		Type:                discrepancy.Type,
		SeatdState:          discrepancy.SeatdState,
		ExternalState:       discrepancy.ExternalState,
		ResolutionState:     discrepancy.ResolutionState,
		Details:             discrepancy.Details,
		CreatedAt:           timeInstantString(discrepancy.CreatedAt),
	}
}

func integrationRunFromDomain(run integrations.ReconciliationRun) integrationRunDTO {
	return integrationRunDTO{
		ID:                 run.ID.String(),
		ConnectionID:       run.ConnectionID.String(),
		StartedAt:          timeInstantString(run.StartedAt),
		CompletedAt:        timeValuePtr(run.CompletedAt),
		Status:             run.Status,
		CheckedCount:       run.CheckedCount,
		DiscrepancyCount:   run.DiscrepancyCount,
		AutoCorrectedCount: run.AutoCorrectedCount,
		LastError:          run.LastError,
	}
}

func uuidValuePtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	value := id.String()
	return &value
}

func timeValuePtr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	out := timeInstantString(*value)
	return &out
}

func timeInstantString(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
