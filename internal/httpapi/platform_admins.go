package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	platformActionAdminInvite       = "platform.admin_invite"
	platformActionAdminInviteRevoke = "platform.admin_invite_revoke"
	platformActionAdminRevoke       = "platform.admin_revoke"
	platformActionAdminReactivate   = "platform.admin_reactivate"
)

type platformAdminsResponse struct {
	Admins      []platformAdminDTO      `json:"admins"`
	Invitations []platformAdminGrantDTO `json:"invitations"`
}

type platformAdminDTO struct {
	ID                 string  `json:"id"`
	UserProfileID      string  `json:"userProfileId"`
	DisplayName        string  `json:"displayName"`
	Email              *string `json:"email,omitempty"`
	Role               string  `json:"role"`
	GrantedByActorRef  string  `json:"grantedByActorRef"`
	GrantedAt          string  `json:"grantedAt"`
	DisabledAt         *string `json:"disabledAt,omitempty"`
	DisabledByActorRef *string `json:"disabledByActorRef,omitempty"`
	CreatedAt          string  `json:"createdAt"`
	UpdatedAt          string  `json:"updatedAt"`
}

type platformAdminGrantDTO struct {
	ID                      string  `json:"id"`
	Email                   string  `json:"email"`
	Role                    string  `json:"role"`
	InvitedByActorRef       string  `json:"invitedByActorRef"`
	CreatedAt               string  `json:"createdAt"`
	ConsumedAt              *string `json:"consumedAt,omitempty"`
	ConsumedByUserProfileID *string `json:"consumedByUserProfileId,omitempty"`
	RevokedAt               *string `json:"revokedAt,omitempty"`
	RevokedByActorRef       *string `json:"revokedByActorRef,omitempty"`
}

type platformAdminInviteRequest struct {
	Email  string `json:"email"`
	Role   string `json:"role"`
	Reason string `json:"reason"`
}

func (api *API) listPlatformAdmins(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	var out platformAdminsResponse
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformAdmin, func(q *db.Queries) error {
		admins, err := q.ListPlatformMemberships(r.Context())
		if err != nil {
			return fmt.Errorf("listing platform memberships: %w", err)
		}
		grants, err := q.ListPlatformAdminGrants(r.Context())
		if err != nil {
			return fmt.Errorf("listing platform admin grants: %w", err)
		}
		out.Admins = make([]platformAdminDTO, 0, len(admins))
		for _, admin := range admins {
			out.Admins = append(out.Admins, platformAdminFromDB(admin))
		}
		out.Invitations = make([]platformAdminGrantDTO, 0, len(grants))
		for _, grant := range grants {
			out.Invitations = append(out.Invitations, platformAdminGrantFromDB(grant))
		}
		return nil
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (api *API) invitePlatformAdmin(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	var body platformAdminInviteRequest
	if !decodeJSONBody(w, r, &body) {
		return
	}
	body.Role = strings.TrimSpace(body.Role)
	reason := strings.TrimSpace(body.Reason)
	if body.Role == "" {
		body.Role = identity.RolePlatformAdmin
	}
	if body.Role != identity.RolePlatformAdmin && body.Role != identity.RoleSupport {
		writeError(w, http.StatusBadRequest, "validation_failed", "role must be platform_admin or support", nil)
		return
	}
	if reason == "" {
		writeError(w, http.StatusBadRequest, "validation_failed", "reason is required", nil)
		return
	}

	var grant db.PlatformAdminGrant
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformManageAdmins, func(q *db.Queries) error {
		var err error
		grant, err = q.CreatePlatformAdminGrant(r.Context(), db.CreatePlatformAdminGrantParams{
			Btrim:             body.Email,
			Role:              body.Role,
			InvitedByActorRef: req.ActorRef,
		})
		if err != nil {
			return fmt.Errorf("creating platform admin grant: %w", err)
		}
		return api.recordPlatformAudit(r.Context(), q, req, uuid.Nil, platformActionAdminInvite, "platform_admin_grant", grant.ID.String(), map[string]any{
			"email":  grant.Email,
			"role":   grant.Role,
			"reason": reason,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, platformAdminGrantFromDB(grant))
}

func (api *API) revokePlatformAdminInvitation(w http.ResponseWriter, r *http.Request) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	reason, ok := decodePlatformStatusChangeReason(w, r)
	if !ok {
		return
	}

	var grant db.PlatformAdminGrant
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformManageAdmins, func(q *db.Queries) error {
		var err error
		grant, err = q.RevokePlatformAdminGrant(r.Context(), db.RevokePlatformAdminGrantParams{
			ID:                id,
			RevokedByActorRef: pgtype.Text{String: req.ActorRef, Valid: true},
		})
		if err != nil {
			return fmt.Errorf("revoking platform admin grant: %w", err)
		}
		return api.recordPlatformAudit(r.Context(), q, req, uuid.Nil, platformActionAdminInviteRevoke, "platform_admin_grant", grant.ID.String(), map[string]any{
			"email":  grant.Email,
			"role":   grant.Role,
			"reason": reason,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, platformAdminGrantFromDB(grant))
}

func (api *API) revokePlatformAdmin(w http.ResponseWriter, r *http.Request) {
	api.changePlatformAdminStatus(w, r, false)
}

func (api *API) reactivatePlatformAdmin(w http.ResponseWriter, r *http.Request) {
	api.changePlatformAdminStatus(w, r, true)
}

func (api *API) changePlatformAdminStatus(w http.ResponseWriter, r *http.Request, active bool) {
	req, ok := api.platformRequestContext(w, r)
	if !ok {
		return
	}
	id, ok := pathUUID(w, r, "id")
	if !ok {
		return
	}
	reason, ok := decodePlatformStatusChangeReason(w, r)
	if !ok {
		return
	}

	var membership db.PlatformMembership
	action := platformActionAdminRevoke
	if active {
		action = platformActionAdminReactivate
	}
	err := api.inAuthorizedPlatformTx(r.Context(), req, identity.PermissionPlatformManageAdmins, func(q *db.Queries) error {
		current, err := q.GetPlatformMembership(r.Context(), id)
		if err != nil {
			return fmt.Errorf("getting platform membership: %w", err)
		}
		if !active && current.Role == identity.RolePlatformAdmin {
			count, err := q.CountActivePlatformAdmins(r.Context())
			if err != nil {
				return fmt.Errorf("counting active platform admins: %w", err)
			}
			if count <= 1 {
				return identity.ErrValidation
			}
		}
		if active {
			membership, err = q.ReactivatePlatformMembership(r.Context(), id)
		} else {
			membership, err = q.DisablePlatformMembership(r.Context(), db.DisablePlatformMembershipParams{
				ID:                 id,
				DisabledByActorRef: pgtype.Text{String: req.ActorRef, Valid: true},
			})
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return pgx.ErrNoRows
			}
			return fmt.Errorf("changing platform membership status: %w", err)
		}
		return api.recordPlatformAudit(r.Context(), q, req, uuid.Nil, action, "platform_membership", membership.ID.String(), map[string]any{
			"role":   membership.Role,
			"reason": reason,
			"active": active,
		})
	})
	if err != nil {
		api.writePlatformError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, platformMembershipFromDB(membership))
}

func platformAdminFromDB(row db.ListPlatformMembershipsRow) platformAdminDTO {
	return platformAdminDTO{
		ID:                 row.ID.String(),
		UserProfileID:      row.UserProfileID.String(),
		DisplayName:        row.DisplayName,
		Email:              textPtr(row.Email),
		Role:               row.Role,
		GrantedByActorRef:  row.GrantedByActorRef,
		GrantedAt:          timeString(row.GrantedAt),
		DisabledAt:         timePtr(row.DisabledAt),
		DisabledByActorRef: textPtr(row.DisabledByActorRef),
		CreatedAt:          timeString(row.CreatedAt),
		UpdatedAt:          timeString(row.UpdatedAt),
	}
}

func platformMembershipFromDB(row db.PlatformMembership) platformAdminDTO {
	return platformAdminDTO{
		ID:                 row.ID.String(),
		UserProfileID:      row.UserProfileID.String(),
		Role:               row.Role,
		GrantedByActorRef:  row.GrantedByActorRef,
		GrantedAt:          timeString(row.GrantedAt),
		DisabledAt:         timePtr(row.DisabledAt),
		DisabledByActorRef: textPtr(row.DisabledByActorRef),
		CreatedAt:          timeString(row.CreatedAt),
		UpdatedAt:          timeString(row.UpdatedAt),
	}
}

func platformAdminGrantFromDB(row db.PlatformAdminGrant) platformAdminGrantDTO {
	return platformAdminGrantDTO{
		ID:                      row.ID.String(),
		Email:                   row.Email,
		Role:                    row.Role,
		InvitedByActorRef:       row.InvitedByActorRef,
		CreatedAt:               timeString(row.CreatedAt),
		ConsumedAt:              timePtr(row.ConsumedAt),
		ConsumedByUserProfileID: uuidPtr(row.ConsumedByUserProfileID),
		RevokedAt:               timePtr(row.RevokedAt),
		RevokedByActorRef:       textPtr(row.RevokedByActorRef),
	}
}
