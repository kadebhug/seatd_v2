package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	platformActionTenantSearch          = "platform.tenant_search"
	platformActionTenantRead            = "platform.tenant_read"
	platformActionTenantDiagnosticsRead = "platform.tenant_diagnostics_read"
	platformActionTenantSuspend         = "platform.tenant_suspend"
	platformActionTenantReactivate      = "platform.tenant_reactivate"
)

type platformTenantSummaryDTO struct {
	ID                    string  `json:"id"`
	Slug                  string  `json:"slug"`
	Name                  string  `json:"name"`
	Status                string  `json:"status"`
	CreatedAt             string  `json:"createdAt"`
	UpdatedAt             string  `json:"updatedAt"`
	LocationCount         int32   `json:"locationCount"`
	ActiveLocationCount   int32   `json:"activeLocationCount"`
	DisabledLocationCount int32   `json:"disabledLocationCount"`
	DeviceCount           int32   `json:"deviceCount"`
	TrustedDeviceCount    int32   `json:"trustedDeviceCount"`
	LastDeviceSeenAt      *string `json:"lastDeviceSeenAt,omitempty"`
	LastAuditAt           *string `json:"lastAuditAt,omitempty"`
}

type platformTenantOverviewDTO struct {
	platformTenantSummaryDTO
	PendingDeviceCount int32 `json:"pendingDeviceCount"`
	RevokedDeviceCount int32 `json:"revokedDeviceCount"`
}

type platformLocationDTO struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Timezone       string `json:"timezone"`
	Status         string `json:"status"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type platformRoleCountDTO struct {
	Role        string `json:"role"`
	MemberCount int32  `json:"memberCount"`
}

type platformDeviceCountDTO struct {
	TrustState  string  `json:"trustState"`
	DeviceType  string  `json:"deviceType"`
	DeviceCount int32   `json:"deviceCount"`
	LastSeenAt  *string `json:"lastSeenAt,omitempty"`
}

type platformDiagnosticsDTO struct {
	DisabledLocationCount        int32    `json:"disabledLocationCount"`
	NeverHeartbeatDeviceCount    int32    `json:"neverHeartbeatDeviceCount"`
	StaleDeviceCount             int32    `json:"staleDeviceCount"`
	PendingDeviceCount           int32    `json:"pendingDeviceCount"`
	RevokedDeviceCount           int32    `json:"revokedDeviceCount"`
	IntegrationCount             int32    `json:"integrationCount"`
	ConnectedIntegrationCount    int32    `json:"connectedIntegrationCount"`
	DegradedIntegrationCount     int32    `json:"degradedIntegrationCount"`
	DisconnectedIntegrationCount int32    `json:"disconnectedIntegrationCount"`
	LastSuccessfulSyncAt         *string  `json:"lastSuccessfulSyncAt,omitempty"`
	RecentPlatformAuditCount     int32    `json:"recentPlatformAuditCount"`
	AnalyticsLagSeconds          *float64 `json:"analyticsLagSeconds,omitempty"`
	AnalyticsRebuildStatus       *string  `json:"analyticsRebuildStatus,omitempty"`
	AnalyticsCheckpointUpdatedAt *string  `json:"analyticsCheckpointUpdatedAt,omitempty"`
}

type platformTenantDetailDTO struct {
	Tenant       platformTenantOverviewDTO `json:"tenant"`
	Locations    []platformLocationDTO     `json:"locations"`
	RoleCounts   []platformRoleCountDTO    `json:"roleCounts"`
	DeviceCounts []platformDeviceCountDTO  `json:"deviceCounts"`
	Diagnostics  platformDiagnosticsDTO    `json:"diagnostics"`
}

type platformRequestContext struct {
	ActorRef string
}

func (api *API) searchPlatformTenants(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	search := strings.TrimSpace(r.URL.Query().Get("q"))
	limit, ok := platformLimit(w, r)
	if !ok {
		return
	}

	var tenants []platformTenantSummaryDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdmin, func(q *db.Queries) error {
		rows, err := q.SearchPlatformTenants(r.Context(), db.SearchPlatformTenantsParams{
			Search:      search,
			ResultLimit: limit,
		})
		if err != nil {
			return fmt.Errorf("searching platform tenants: %w", err)
		}
		tenants = make([]platformTenantSummaryDTO, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, platformTenantSummaryFromSearch(row))
		}
		return api.recordPlatformAudit(r.Context(), q, req, uuid.Nil, platformActionTenantSearch, "platform", "tenants", map[string]any{
			"query":       search,
			"limit":       limit,
			"resultCount": len(tenants),
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tenants": tenants})
}

func (api *API) getPlatformTenant(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	tenantID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}

	var detail platformTenantDetailDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdmin, func(q *db.Queries) error {
		var err error
		detail, err = loadPlatformTenantDetail(r.Context(), q, tenantID)
		if err != nil {
			return err
		}
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, platformActionTenantRead, "organisation", tenantID.String(), nil)
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (api *API) getPlatformTenantDiagnostics(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	tenantID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}

	var diagnostics platformDiagnosticsDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdmin, func(q *db.Queries) error {
		if _, err := q.GetPlatformTenantOverview(r.Context(), tenantID); err != nil {
			return fmt.Errorf("getting platform tenant overview: %w", err)
		}
		row, err := q.GetPlatformTenantDiagnostics(r.Context(), tenantID)
		if err != nil {
			return fmt.Errorf("getting platform tenant diagnostics: %w", err)
		}
		diagnostics = platformDiagnosticsFromDB(row)
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, platformActionTenantDiagnosticsRead, "organisation", tenantID.String(), nil)
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"diagnostics": diagnostics})
}

func loadPlatformTenantDetail(ctx context.Context, q *db.Queries, tenantID uuid.UUID) (platformTenantDetailDTO, error) {
	overview, err := q.GetPlatformTenantOverview(ctx, tenantID)
	if err != nil {
		return platformTenantDetailDTO{}, fmt.Errorf("getting platform tenant overview: %w", err)
	}
	locations, err := q.ListPlatformTenantLocations(ctx, tenantID)
	if err != nil {
		return platformTenantDetailDTO{}, fmt.Errorf("listing platform tenant locations: %w", err)
	}
	roleCounts, err := q.ListPlatformTenantMembershipRoleCounts(ctx, tenantID)
	if err != nil {
		return platformTenantDetailDTO{}, fmt.Errorf("listing platform tenant role counts: %w", err)
	}
	deviceCounts, err := q.ListPlatformTenantDeviceCounts(ctx, tenantID)
	if err != nil {
		return platformTenantDetailDTO{}, fmt.Errorf("listing platform tenant device counts: %w", err)
	}
	diagnostics, err := q.GetPlatformTenantDiagnostics(ctx, tenantID)
	if err != nil {
		return platformTenantDetailDTO{}, fmt.Errorf("getting platform tenant diagnostics: %w", err)
	}
	return platformTenantDetailDTO{
		Tenant:       platformTenantOverviewFromDB(overview),
		Locations:    platformLocationsFromDB(locations),
		RoleCounts:   platformRoleCountsFromDB(roleCounts),
		DeviceCounts: platformDeviceCountsFromDB(deviceCounts),
		Diagnostics:  platformDiagnosticsFromDB(diagnostics),
	}, nil
}

type platformTenantStatusChangeRequest struct {
	Reason string `json:"reason"`
}

func decodePlatformStatusChangeReason(w http.ResponseWriter, r *http.Request) (string, bool) {
	var body platformTenantStatusChangeRequest
	if !decodeJSONBody(w, r, &body) {
		return "", false
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "reason is required", nil)
		return "", false
	}
	return reason, true
}

func (api *API) suspendPlatformTenant(w http.ResponseWriter, r *http.Request) {
	api.changePlatformTenantStatus(w, r, "disabled", platformActionTenantSuspend)
}

func (api *API) reactivatePlatformTenant(w http.ResponseWriter, r *http.Request) {
	api.changePlatformTenantStatus(w, r, "active", platformActionTenantReactivate)
}

func (api *API) changePlatformTenantStatus(w http.ResponseWriter, r *http.Request, status string, action string) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	tenantID, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	reason, ok := decodePlatformStatusChangeReason(w, r)
	if !ok {
		return
	}

	var detail platformTenantDetailDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdminWrite, func(q *db.Queries) error {
		if _, err := q.SetOrganisationStatus(r.Context(), db.SetOrganisationStatusParams{
			ID:     tenantID,
			Status: status,
		}); err != nil {
			return fmt.Errorf("setting organisation status: %w", err)
		}
		var err error
		detail, err = loadPlatformTenantDetail(r.Context(), q, tenantID)
		if err != nil {
			return err
		}
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, action, "organisation", tenantID.String(), map[string]any{
			"reason": reason,
			"status": status,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (api *API) platformRequestContext(w http.ResponseWriter, r *http.Request) (platformRequestContext, bool) {
	if strings.TrimSpace(r.Header.Get(headerDeviceID)) != "" {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return platformRequestContext{}, false
	}
	if isDevelopmentEnvironment(api.cfg.Environment) {
		if actorRef := strings.TrimSpace(r.Header.Get(headerActorRef)); actorRef != "" {
			return platformRequestContext{ActorRef: actorRef}, true
		}
	}
	if hasClientIdentityHeaders(r) {
		writeError(w, http.StatusBadRequest, "validation_failed", "client-supplied identity headers are not accepted", nil)
		return platformRequestContext{}, false
	}
	secret, ok := bearerCredential(w, r)
	if !ok {
		return platformRequestContext{}, false
	}
	session, err := api.ident.ValidateWebSession(r.Context(), secret)
	if err != nil {
		api.writeIdentityError(w, err)
		return platformRequestContext{}, false
	}
	return platformRequestContext{ActorRef: actorRefForSession(session)}, true
}

func (api *API) inAuthorizedPlatformTx(ctx context.Context, req platformRequestContext, permission string, fn func(*db.Queries) error) error {
	if strings.TrimSpace(req.ActorRef) == "" {
		return identity.ErrUnauthorized
	}
	tx, err := api.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning platform transaction: %w", err)
	}
	q := db.New(tx)
	if err := q.SetPlatformAdminContext(ctx); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting platform admin context: %w", err))
	}
	allowed, err := q.ActorHasPlatformPermission(ctx, db.ActorHasPlatformPermissionParams{
		MemberRef:      req.ActorRef,
		PermissionName: permission,
	})
	if err != nil {
		return rollback(tx, ctx, fmt.Errorf("checking platform permission: %w", err))
	}
	if !allowed {
		return rollback(tx, ctx, identity.ErrForbidden)
	}
	if err := fn(q); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing platform transaction: %w", err)
	}
	return nil
}

func (api *API) recordPlatformAudit(
	ctx context.Context,
	q *db.Queries,
	req platformRequestContext,
	organisationID uuid.UUID,
	action string,
	targetType string,
	targetID string,
	metadata map[string]any,
) error {
	payload := []byte(`{}`)
	if metadata != nil {
		var err error
		payload, err = json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("encoding platform audit metadata: %w", err)
		}
	}
	_, err := q.RecordAuditEvent(ctx, db.RecordAuditEventParams{
		OrganisationID: nullableAuditUUID(organisationID),
		ActorRef:       req.ActorRef,
		Action:         action,
		TargetType:     targetType,
		TargetID:       targetID,
		Metadata:       payload,
	})
	if err != nil {
		return fmt.Errorf("recording platform audit event: %w", err)
	}
	return nil
}

func (api *API) writePlatformError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, identity.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "credential is invalid", nil)
	case errors.Is(err, identity.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, http.StatusNotFound, "not_found", "resource not found", nil)
	default:
		api.logger.Error("platform request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func platformLimit(w http.ResponseWriter, r *http.Request) (int32, bool) {
	limit := int32(25)
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeError(w, http.StatusBadRequest, "validation_failed", "limit must be between 1 and 100", nil)
			return 0, false
		}
		limit = int32(parsed)
	}
	return limit, true
}

func nullableAuditUUID(value uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: value, Valid: value != uuid.Nil}
}

func platformTenantSummaryFromSearch(row db.SearchPlatformTenantsRow) platformTenantSummaryDTO {
	return platformTenantSummaryDTO{
		ID:                    row.ID.String(),
		Slug:                  row.Slug,
		Name:                  row.Name,
		Status:                row.Status,
		CreatedAt:             timeString(row.CreatedAt),
		UpdatedAt:             timeString(row.UpdatedAt),
		LocationCount:         row.LocationCount,
		ActiveLocationCount:   row.ActiveLocationCount,
		DisabledLocationCount: row.DisabledLocationCount,
		DeviceCount:           row.DeviceCount,
		TrustedDeviceCount:    row.TrustedDeviceCount,
		LastDeviceSeenAt:      timePtr(row.LastDeviceSeenAt),
		LastAuditAt:           timePtr(row.LastAuditAt),
	}
}

func platformTenantOverviewFromDB(row db.GetPlatformTenantOverviewRow) platformTenantOverviewDTO {
	return platformTenantOverviewDTO{
		platformTenantSummaryDTO: platformTenantSummaryDTO{
			ID:                    row.ID.String(),
			Slug:                  row.Slug,
			Name:                  row.Name,
			Status:                row.Status,
			CreatedAt:             timeString(row.CreatedAt),
			UpdatedAt:             timeString(row.UpdatedAt),
			LocationCount:         row.LocationCount,
			ActiveLocationCount:   row.ActiveLocationCount,
			DisabledLocationCount: row.DisabledLocationCount,
			DeviceCount:           row.DeviceCount,
			TrustedDeviceCount:    row.TrustedDeviceCount,
			LastDeviceSeenAt:      timePtr(row.LastDeviceSeenAt),
			LastAuditAt:           timePtr(row.LastAuditAt),
		},
		PendingDeviceCount: row.PendingDeviceCount,
		RevokedDeviceCount: row.RevokedDeviceCount,
	}
}

func platformLocationsFromDB(rows []db.ListPlatformTenantLocationsRow) []platformLocationDTO {
	out := make([]platformLocationDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, platformLocationDTO{
			ID:             row.ID.String(),
			OrganisationID: row.OrganisationID.String(),
			Slug:           row.Slug,
			Name:           row.Name,
			Timezone:       row.Timezone,
			Status:         row.Status,
			CreatedAt:      timeString(row.CreatedAt),
			UpdatedAt:      timeString(row.UpdatedAt),
		})
	}
	return out
}

func platformRoleCountsFromDB(rows []db.ListPlatformTenantMembershipRoleCountsRow) []platformRoleCountDTO {
	out := make([]platformRoleCountDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, platformRoleCountDTO{
			Role:        row.Role,
			MemberCount: row.MemberCount,
		})
	}
	return out
}

func platformDeviceCountsFromDB(rows []db.ListPlatformTenantDeviceCountsRow) []platformDeviceCountDTO {
	out := make([]platformDeviceCountDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, platformDeviceCountDTO{
			TrustState:  row.TrustState,
			DeviceType:  row.DeviceType,
			DeviceCount: row.DeviceCount,
			LastSeenAt:  timePtr(row.LastSeenAt),
		})
	}
	return out
}

func platformDiagnosticsFromDB(row db.GetPlatformTenantDiagnosticsRow) platformDiagnosticsDTO {
	out := platformDiagnosticsDTO{
		DisabledLocationCount:        row.DisabledLocationCount,
		NeverHeartbeatDeviceCount:    row.NeverHeartbeatDeviceCount,
		StaleDeviceCount:             row.StaleDeviceCount,
		PendingDeviceCount:           row.PendingDeviceCount,
		RevokedDeviceCount:           row.RevokedDeviceCount,
		IntegrationCount:             row.IntegrationCount,
		ConnectedIntegrationCount:    row.ConnectedIntegrationCount,
		DegradedIntegrationCount:     row.DegradedIntegrationCount,
		DisconnectedIntegrationCount: row.DisconnectedIntegrationCount,
		LastSuccessfulSyncAt:         timePtr(row.LastSuccessfulSyncAt),
		RecentPlatformAuditCount:     row.RecentPlatformAuditCount,
		AnalyticsCheckpointUpdatedAt: timePtr(row.AnalyticsCheckpointUpdatedAt),
	}
	if row.AnalyticsLagSeconds.Valid {
		out.AnalyticsLagSeconds = &row.AnalyticsLagSeconds.Float64
	}
	if row.AnalyticsRebuildStatus.Valid {
		out.AnalyticsRebuildStatus = &row.AnalyticsRebuildStatus.String
	}
	return out
}
