package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	platformActionTenantCreate    = "platform.tenant_create"
	platformActionOwnerInvite     = "platform.owner_invite"
	platformActionOwnerSuspend    = "platform.owner_suspend"
	platformActionOwnerReactivate = "platform.owner_reactivate"
	platformActionOwnerReassign   = "platform.owner_reassign"
	platformActionTenantAuditRead = "platform.tenant_audit_read"
)

type platformTenantCreateRequest struct {
	OrganisationName string `json:"organisationName"`
	OrganisationSlug string `json:"organisationSlug"`
	LocationName     string `json:"locationName"`
	LocationSlug     string `json:"locationSlug"`
	Timezone         string `json:"timezone"`
	OwnerEmail       string `json:"ownerEmail"`
	OwnerDisplayName string `json:"ownerDisplayName"`
	Reason           string `json:"reason"`
}

type platformTenantCreateResponse struct {
	Tenant platformTenantDetailDTO `json:"tenant"`
	Owner  membershipDTO           `json:"owner"`
}

type platformOwnerRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	Reason      string `json:"reason"`
}

type platformOwnerReassignRequest struct {
	ToEmail string `json:"toEmail"`
	Reason  string `json:"reason"`
}

type platformOwnersResponse struct {
	Owners []membershipDTO `json:"owners"`
}

type platformTenantAuditResponse struct {
	Events []platformAuditEventDTO `json:"events"`
}

type platformAuditEventDTO struct {
	ID             string          `json:"id"`
	OrganisationID *string         `json:"organisationId,omitempty"`
	LocationID     *string         `json:"locationId,omitempty"`
	ActorRef       string          `json:"actorRef"`
	Action         string          `json:"action"`
	TargetType     string          `json:"targetType"`
	TargetID       string          `json:"targetId"`
	Metadata       json.RawMessage `json:"metadata"`
	CreatedAt      string          `json:"createdAt"`
}

func (api *API) createPlatformTenant(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	var body platformTenantCreateRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "reason is required", nil)
		return
	}

	var result identity.PlatformProvisionTenantResult
	var detail platformTenantDetailDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdminWrite, func(q *db.Queries) error {
		var err error
		result, err = api.ident.PlatformProvisionTenant(r.Context(), identity.PlatformProvisionTenantParams{
			OrganisationName: body.OrganisationName,
			OrganisationSlug: body.OrganisationSlug,
			LocationName:     body.LocationName,
			LocationSlug:     body.LocationSlug,
			Timezone:         body.Timezone,
			OwnerEmail:       body.OwnerEmail,
			OwnerDisplayName: body.OwnerDisplayName,
		})
		if err != nil {
			return err
		}
		detail, err = loadPlatformTenantDetail(r.Context(), q, result.Organisation.ID)
		if err != nil {
			return err
		}
		return api.recordPlatformAudit(r.Context(), q, req, result.Organisation.ID, platformActionTenantCreate, "organisation", result.Organisation.ID.String(), map[string]any{
			"reason":     reason,
			"ownerEmail": body.OwnerEmail,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, platformTenantCreateResponse{
		Tenant: detail,
		Owner:  managedMembershipFromDomain(result.Owner),
	})
}

func (api *API) listPlatformTenantOwners(w http.ResponseWriter, r *http.Request) {
	req, tenantID, ok := api.platformTenantRequest(w, r)
	if !ok {
		return
	}
	var owners []membershipDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdmin, func(q *db.Queries) error {
		rows, err := q.ListPlatformTenantOwners(r.Context(), db.ListPlatformTenantOwnersParams{
			OrganisationID: tenantID,
			Column2:        true,
		})
		if err != nil {
			return fmt.Errorf("listing tenant owners: %w", err)
		}
		owners = make([]membershipDTO, 0, len(rows))
		for _, row := range rows {
			owners = append(owners, platformTenantOwnerFromDB(row))
		}
		return nil
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, platformOwnersResponse{Owners: owners})
}

func (api *API) invitePlatformTenantOwner(w http.ResponseWriter, r *http.Request) {
	req, tenantID, ok := api.platformTenantRequest(w, r)
	if !ok {
		return
	}
	var body platformOwnerRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "reason is required", nil)
		return
	}
	var membership identity.ManagedMembership
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdminWrite, func(q *db.Queries) error {
		var err error
		membership, err = api.ident.PlatformInviteOwner(r.Context(), identity.PlatformInviteOwnerParams{
			TargetOrganisationID: tenantID,
			Email:                body.Email,
			DisplayName:          body.DisplayName,
			ActorRef:             req.ActorRef,
		})
		if err != nil {
			return err
		}
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, platformActionOwnerInvite, "membership", membership.ID.String(), map[string]any{
			"reason": reason,
			"email":  membership.Email,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, managedMembershipFromDomain(membership))
}

func (api *API) suspendPlatformTenantOwner(w http.ResponseWriter, r *http.Request) {
	api.changePlatformTenantOwnerStatus(w, r, false)
}

func (api *API) reactivatePlatformTenantOwner(w http.ResponseWriter, r *http.Request) {
	api.changePlatformTenantOwnerStatus(w, r, true)
}

func (api *API) changePlatformTenantOwnerStatus(w http.ResponseWriter, r *http.Request, active bool) {
	req, tenantID, membershipID, ok := api.platformTenantOwnerRequest(w, r)
	if !ok {
		return
	}
	reason, ok := decodePlatformStatusChangeReason(w, r)
	if !ok {
		return
	}
	var membership identity.ManagedMembership
	action := platformActionOwnerSuspend
	if active {
		action = platformActionOwnerReactivate
	}
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdminWrite, func(q *db.Queries) error {
		var err error
		params := identity.PlatformOwnerMembershipParams{
			TargetOrganisationID: tenantID,
			MembershipID:         membershipID,
			ActorRef:             req.ActorRef,
		}
		if active {
			membership, err = api.ident.PlatformReactivateOwnerMembership(r.Context(), params)
		} else {
			membership, err = api.ident.PlatformDisableOwnerMembership(r.Context(), params)
		}
		if err != nil {
			return err
		}
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, action, "membership", membership.ID.String(), map[string]any{
			"reason": reason,
			"active": active,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, managedMembershipFromDomain(membership))
}

func (api *API) reassignPlatformTenantOwner(w http.ResponseWriter, r *http.Request) {
	req, tenantID, membershipID, ok := api.platformTenantOwnerRequest(w, r)
	if !ok {
		return
	}
	var body platformOwnerReassignRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	reason := strings.TrimSpace(body.Reason)
	if reason == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "reason is required", nil)
		return
	}
	var result identity.PlatformReassignOwnershipResult
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdminWrite, func(q *db.Queries) error {
		var err error
		result, err = api.ident.PlatformReassignOwnership(r.Context(), identity.PlatformReassignOwnershipParams{
			TargetOrganisationID: tenantID,
			FromMembershipID:     membershipID,
			ToEmail:              body.ToEmail,
			ActorRef:             req.ActorRef,
		})
		if err != nil {
			return err
		}
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, platformActionOwnerReassign, "membership", membershipID.String(), map[string]any{
			"reason":       reason,
			"toMembership": result.NextOwner.ID.String(),
			"toEmail":      result.NextOwner.Email,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]membershipDTO{
		"previousOwner": managedMembershipFromDomain(result.PreviousOwner),
		"nextOwner":     managedMembershipFromDomain(result.NextOwner),
	})
}

func (api *API) listPlatformTenantAudit(w http.ResponseWriter, r *http.Request) {
	req, tenantID, ok := api.platformTenantRequest(w, r)
	if !ok {
		return
	}
	limit, ok := platformLimit(w, r)
	if !ok {
		return
	}
	var events []platformAuditEventDTO
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdmin, func(q *db.Queries) error {
		rows, err := q.ListPlatformTenantAudit(r.Context(), db.ListPlatformTenantAuditParams{
			OrganisationID: nullableAuditUUID(tenantID),
			Limit:          limit,
		})
		if err != nil {
			return fmt.Errorf("listing tenant audit: %w", err)
		}
		events = make([]platformAuditEventDTO, 0, len(rows))
		for _, row := range rows {
			events = append(events, platformAuditEventFromDB(row))
		}
		return api.recordPlatformAudit(r.Context(), q, req, tenantID, platformActionTenantAuditRead, "organisation", tenantID.String(), map[string]any{
			"limit": limit,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, platformTenantAuditResponse{Events: events})
}

func (api *API) platformTenantRequest(w http.ResponseWriter, r *http.Request) (platformRequestContext, uuid.UUID, bool) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return platformRequestContext{}, uuid.Nil, false
	}
	tenantID, ok := pathUUID(w, r, "id")
	return req, tenantID, ok
}

func (api *API) platformTenantOwnerRequest(w http.ResponseWriter, r *http.Request) (platformRequestContext, uuid.UUID, uuid.UUID, bool) {
	req, tenantID, ok := api.platformTenantRequest(w, r)
	if !ok {
		return platformRequestContext{}, uuid.Nil, uuid.Nil, false
	}
	membershipID, ok := pathUUID(w, r, "membershipId")
	return req, tenantID, membershipID, ok
}

func platformTenantOwnerFromDB(row db.ListPlatformTenantOwnersRow) membershipDTO {
	userProfileID := uuid.Nil
	if row.UserProfileID.Valid {
		userProfileID = row.UserProfileID.UUID
	}
	membership := identity.ManagedMembership{
		ID:             row.ID,
		Scope:          row.Scope,
		OrganisationID: row.OrganisationID,
		UserProfileID:  userProfileID,
		DisplayName:    platformTextValue(row.DisplayName),
		Email:          platformTextValue(row.Email),
		MemberRef:      row.MemberRef,
		Role:           row.Role,
		DisabledAt:     row.DisabledAt,
	}
	return managedMembershipFromDomain(membership)
}

func platformTextValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func platformAuditEventFromDB(row db.AuditEvent) platformAuditEventDTO {
	metadata := row.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	return platformAuditEventDTO{
		ID:             row.ID.String(),
		OrganisationID: uuidPtr(row.OrganisationID),
		LocationID:     uuidPtr(row.LocationID),
		ActorRef:       row.ActorRef,
		Action:         row.Action,
		TargetType:     row.TargetType,
		TargetID:       row.TargetID,
		Metadata:       metadata,
		CreatedAt:      timeString(row.CreatedAt),
	}
}
