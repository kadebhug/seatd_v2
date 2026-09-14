package httpapi

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/identity"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

type principalType string

const (
	principalUser          principalType = "user"
	principalDevice        principalType = "device"
	principalTrustedHeader principalType = "trusted_header"
)

type authPolicy struct {
	Permission      string
	AllowUser       bool
	AllowDevice     bool
	RequireLocation bool
	RequireActor    bool
}

type oidcSessionRequest struct {
	Issuer        string `json:"issuer"`
	Subject       string `json:"subject"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	DisplayName   string `json:"displayName"`
	TTLSeconds    int64  `json:"ttlSeconds"`
}

type webSessionResponse struct {
	Session webSessionDTO `json:"session"`
}

type webSessionDTO struct {
	SessionSecret string          `json:"sessionSecret,omitempty"`
	ID            string          `json:"id"`
	UserProfileID string          `json:"userProfileId"`
	ActorRef      string          `json:"actorRef"`
	DisplayName   string          `json:"displayName"`
	Email         *string         `json:"email,omitempty"`
	IssuedAt      string          `json:"issuedAt"`
	ExpiresAt     string          `json:"expiresAt"`
	Memberships   []membershipDTO `json:"memberships"`
	Locations     []locationDTO   `json:"locations"`
}

func (api *API) createOIDCWebSession(w http.ResponseWriter, r *http.Request) {
	if !api.authorizeInternalRequest(w, r) {
		return
	}
	var req oidcSessionRequest
	if !decodeJSONBody(w, r, &req) {
		return
	}
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = 8 * time.Hour
	}
	profile, err := api.ident.ResolveExternalIdentity(r.Context(), identity.ResolveExternalIdentityParams{
		DisplayName:   req.DisplayName,
		Email:         req.Email,
		EmailVerified: req.EmailVerified,
		External: identity.ExternalIdentity{
			Issuer:  req.Issuer,
			Subject: req.Subject,
			Email:   req.Email,
		},
	})
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	session, err := api.ident.CreateWebSession(r.Context(), identity.CreateWebSessionParams{
		UserProfileID: profile.User.ID,
		TTL:           ttl,
		UserAgent:     r.UserAgent(),
		IPAddress:     clientIP(r),
	})
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	session.User = profile.User
	api.writeWebSession(w, r, http.StatusCreated, session)
}

func (api *API) getWebSession(w http.ResponseWriter, r *http.Request) {
	secret, ok := bearerCredential(w, r)
	if !ok {
		return
	}
	session, err := api.ident.ValidateWebSession(r.Context(), secret)
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	api.writeWebSession(w, r, http.StatusOK, session)
}

func (api *API) rotateWebSession(w http.ResponseWriter, r *http.Request) {
	secret, ok := bearerCredential(w, r)
	if !ok {
		return
	}
	session, err := api.ident.RotateWebSession(r.Context(), secret, 8*time.Hour, r.UserAgent(), clientIP(r))
	if err != nil {
		api.writeIdentityError(w, err)
		return
	}
	api.writeWebSession(w, r, http.StatusOK, session)
}

func (api *API) logoutWebSession(w http.ResponseWriter, r *http.Request) {
	secret, ok := bearerCredential(w, r)
	if !ok {
		return
	}
	if err := api.ident.RevokeWebSession(r.Context(), secret); err != nil {
		api.writeIdentityError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (api *API) authenticatedRequestContext(w http.ResponseWriter, r *http.Request, requireLocation bool, requireActor bool) (requestContext, bool) {
	orgID, ok := parseHeaderUUID(w, r, headerOrganisationID, true)
	if !ok {
		return requestContext{}, false
	}
	locationID, ok := parseHeaderUUID(w, r, headerLocationID, requireLocation)
	if !ok {
		return requestContext{}, false
	}
	secret, ok := bearerCredential(w, r)
	if !ok {
		return requestContext{}, false
	}

	policy := routeAuthPolicy(r)
	if !policy.AllowUser && !policy.AllowDevice {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return requestContext{}, false
	}

	session, err := api.ident.ValidateWebSession(r.Context(), secret)
	if err == nil {
		if !policy.AllowUser {
			writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
			return requestContext{}, false
		}
		return api.webSessionRequestContext(w, session, orgID, locationID, requireActor, policy)
	}
	if !errors.Is(err, identity.ErrUnauthorized) || !policy.AllowDevice {
		api.writeIdentityError(w, err)
		return requestContext{}, false
	}

	device, err := api.ident.AuthenticateDeviceCredential(r.Context(), secret)
	if err != nil {
		api.writeIdentityError(w, err)
		return requestContext{}, false
	}
	return api.deviceRequestContext(w, device, orgID, locationID, requireActor, policy)
}

func (api *API) webSessionRequestContext(
	w http.ResponseWriter,
	session identity.WebSession,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	requireActor bool,
	policy authPolicy,
) (requestContext, bool) {
	if !sessionAllowsScope(session, organisationID, locationID) {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return requestContext{}, false
	}
	permissions := sessionPermissions(session, organisationID, locationID)
	if policy.Permission != "" && !hasPermission(permissions, policy.Permission) {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return requestContext{}, false
	}
	actorRef := ""
	if requireActor || len(session.Memberships) > 0 {
		actorRef = actorRefForSession(session)
	}
	return requestContext{
		OrganisationID: organisationID,
		LocationID:     locationID,
		ActorRef:       actorRef,
		UserProfileID:  uuid.NullUUID{UUID: session.User.ID, Valid: session.User.ID != uuid.Nil},
		PrincipalType:  principalUser,
		Roles:          sessionRoles(session, organisationID, locationID),
		Permissions:    permissions,
	}, true
}

func (api *API) deviceRequestContext(
	w http.ResponseWriter,
	device db.Device,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	requireActor bool,
	policy authPolicy,
) (requestContext, bool) {
	if device.OrganisationID != organisationID {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return requestContext{}, false
	}
	if !device.LocationID.Valid {
		writeError(w, http.StatusForbidden, "forbidden", "device is not assigned to a location", nil)
		return requestContext{}, false
	}
	if locationID == uuid.Nil {
		locationID = device.LocationID.UUID
	}
	if device.LocationID.UUID != locationID {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return requestContext{}, false
	}
	permissions := devicePermissions(device)
	if policy.Permission != "" && !hasPermission(permissions, policy.Permission) {
		writeError(w, http.StatusForbidden, "forbidden", "permission denied", nil)
		return requestContext{}, false
	}
	actorRef := ""
	if requireActor {
		actorRef = "device:" + device.ID.String()
	}
	return requestContext{
		OrganisationID: organisationID,
		LocationID:     locationID,
		ActorRef:       actorRef,
		DeviceID:       uuid.NullUUID{UUID: device.ID, Valid: true},
		PrincipalType:  principalDevice,
		Permissions:    permissions,
	}, true
}

func (api *API) authorizeInternalRequest(w http.ResponseWriter, r *http.Request) bool {
	if isDevelopmentEnvironment(api.cfg.Environment) {
		return true
	}
	if api.cfg.InternalAPISecret == "" {
		writeError(w, http.StatusInternalServerError, "internal_error", "internal API secret is not configured", nil)
		return false
	}
	got := strings.TrimSpace(r.Header.Get(headerInternalSecret))
	if subtle.ConstantTimeCompare([]byte(got), []byte(api.cfg.InternalAPISecret)) != 1 {
		writeError(w, http.StatusUnauthorized, "unauthorized", "internal credential is invalid", nil)
		return false
	}
	return true
}

func (api *API) writeWebSession(w http.ResponseWriter, r *http.Request, status int, session identity.WebSession) {
	writeJSON(w, status, webSessionResponse{Session: api.webSessionDTO(r, session)})
}

func (api *API) webSessionDTO(r *http.Request, session identity.WebSession) webSessionDTO {
	return webSessionDTO{
		SessionSecret: session.Secret,
		ID:            session.Session.ID.String(),
		UserProfileID: session.User.ID.String(),
		ActorRef:      actorRefForSession(session),
		DisplayName:   session.User.DisplayName,
		Email:         textPtr(session.User.Email),
		IssuedAt:      timeString(session.Session.IssuedAt),
		ExpiresAt:     timeString(session.Session.ExpiresAt),
		Memberships:   membershipDTOs(session.Memberships),
		Locations:     api.locationsForSession(r, session),
	}
}

func (api *API) locationsForSession(r *http.Request, session identity.WebSession) []locationDTO {
	seen := map[uuid.UUID]bool{}
	locations := make([]locationDTO, 0)
	for _, membership := range session.Memberships {
		if membership.LocationID != uuid.Nil && !seen[membership.LocationID] {
			location, ok := api.locationForSession(r, membership.OrganisationID, membership.LocationID)
			if ok {
				locations = append(locations, locationFromDB(location))
				seen[membership.LocationID] = true
			}
			continue
		}
		if membership.OrganisationID == uuid.Nil {
			continue
		}
		api.inTenantTx(r.Context(), requestContext{OrganisationID: membership.OrganisationID}, func(q *db.Queries) error {
			items, err := q.ListLocationsByOrganisation(r.Context(), membership.OrganisationID)
			if err != nil {
				return err
			}
			for _, location := range items {
				if !seen[location.ID] {
					locations = append(locations, locationFromDB(location))
					seen[location.ID] = true
				}
			}
			return nil
		})
	}
	return locations
}

func (api *API) locationForSession(r *http.Request, organisationID, locationID uuid.UUID) (db.Location, bool) {
	var location db.Location
	err := api.inTenantTx(r.Context(), requestContext{OrganisationID: organisationID, LocationID: locationID}, func(q *db.Queries) error {
		var err error
		location, err = q.GetLocation(r.Context(), db.GetLocationParams{
			ID:             locationID,
			OrganisationID: organisationID,
		})
		return err
	})
	return location, err == nil
}

func membershipDTOs(memberships []identity.Membership) []membershipDTO {
	out := make([]membershipDTO, 0, len(memberships))
	for _, membership := range memberships {
		dto := membershipDTO{
			Scope:          "organisation",
			OrganisationID: membership.OrganisationID.String(),
			MemberRef:      membership.MemberRef,
			Role:           membership.Role,
		}
		if membership.OrganisationID == uuid.Nil {
			dto.Scope = "platform"
			dto.OrganisationID = ""
		}
		if membership.LocationID != uuid.Nil {
			locationID := membership.LocationID.String()
			dto.Scope = "location"
			dto.LocationID = &locationID
		}
		out = append(out, dto)
	}
	return out
}

func sessionAllowsScope(session identity.WebSession, organisationID, locationID uuid.UUID) bool {
	for _, membership := range session.Memberships {
		if membership.OrganisationID != organisationID {
			continue
		}
		if locationID == uuid.Nil {
			if membership.LocationID == uuid.Nil {
				return true
			}
			continue
		}
		if membership.LocationID == uuid.Nil || membership.LocationID == locationID {
			return true
		}
	}
	return false
}

func actorRefForSession(session identity.WebSession) string {
	for _, membership := range session.Memberships {
		if strings.TrimSpace(membership.MemberRef) != "" {
			return membership.MemberRef
		}
	}
	return userActorRef(session.User.ID)
}

func sessionRoles(session identity.WebSession, organisationID, locationID uuid.UUID) []string {
	seen := map[string]bool{}
	var roles []string
	for _, membership := range session.Memberships {
		if membership.OrganisationID != organisationID {
			continue
		}
		if locationID == uuid.Nil && membership.LocationID != uuid.Nil {
			continue
		}
		if locationID != uuid.Nil && membership.LocationID != uuid.Nil && membership.LocationID != locationID {
			continue
		}
		if membership.Role != "" && !seen[membership.Role] {
			roles = append(roles, membership.Role)
			seen[membership.Role] = true
		}
	}
	return roles
}

func sessionPermissions(session identity.WebSession, organisationID, locationID uuid.UUID) []string {
	seen := map[string]bool{}
	var permissions []string
	for _, role := range sessionRoles(session, organisationID, locationID) {
		for _, permission := range rolePermissions(role) {
			if !seen[permission] {
				permissions = append(permissions, permission)
				seen[permission] = true
			}
		}
	}
	return permissions
}

func rolePermissions(role string) []string {
	switch role {
	case identity.RolePlatformAdmin:
		return []string{
			identity.PermissionPlatformAdmin,
			identity.PermissionPlatformAdminWrite,
			identity.PermissionPlatformManageAdmins,
			identity.PermissionAuditRead,
		}
	case identity.RoleSupport:
		return []string{identity.PermissionPlatformAdmin, identity.PermissionAuditRead}
	case identity.RoleOrganisationOwner:
		return []string{
			identity.PermissionOrganisationManage,
			identity.PermissionLocationManage,
			identity.PermissionLayoutRead,
			identity.PermissionLayoutWrite,
			identity.PermissionOperationsRead,
			identity.PermissionOperationsWrite,
			identity.PermissionDeviceManage,
			identity.PermissionAuditRead,
			identity.PermissionAnalyticsRead,
			identity.PermissionIntegrationsRead,
			identity.PermissionIntegrationsManage,
		}
	case identity.RoleLocationManager:
		return []string{
			identity.PermissionLocationManage,
			identity.PermissionLayoutRead,
			identity.PermissionLayoutWrite,
			identity.PermissionOperationsRead,
			identity.PermissionOperationsWrite,
			identity.PermissionDeviceManage,
			identity.PermissionAnalyticsRead,
			identity.PermissionIntegrationsRead,
			identity.PermissionIntegrationsManage,
		}
	case identity.RoleWaiter:
		return []string{
			identity.PermissionLayoutRead,
			identity.PermissionOperationsRead,
			identity.PermissionOperationsWrite,
		}
	case identity.RoleReadOnly:
		return []string{identity.PermissionLayoutRead, identity.PermissionOperationsRead}
	default:
		return nil
	}
}

func userProfileIDFromActorRef(actorRef string) (uuid.UUID, error) {
	value, ok := strings.CutPrefix(strings.TrimSpace(actorRef), "user:")
	if !ok {
		return uuid.Nil, identity.ErrValidation
	}
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, identity.ErrValidation
	}
	return id, nil
}

func devicePermissions(device db.Device) []string {
	switch device.DeviceType {
	case identity.DeviceTypeDisplay:
		return []string{identity.PermissionLayoutRead, identity.PermissionOperationsRead}
	case identity.DeviceTypeWaiterMobile, identity.DeviceTypeManagerTablet, identity.DeviceTypeHostDevice:
		return []string{
			identity.PermissionLayoutRead,
			identity.PermissionOperationsRead,
			identity.PermissionOperationsWrite,
		}
	default:
		return nil
	}
}

func hasPermission(permissions []string, permission string) bool {
	for _, candidate := range permissions {
		if candidate == permission {
			return true
		}
	}
	return false
}

func userActorRef(id uuid.UUID) string {
	if id == uuid.Nil {
		return "user:"
	}
	return "user:" + id.String()
}

func isDevelopmentEnvironment(env string) bool {
	return env == app.EnvLocal || env == app.EnvTest
}

func ttlSeconds(value string, fallback time.Duration) time.Duration {
	seconds, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || seconds <= 0 {
		return fallback
	}
	return time.Duration(seconds) * time.Second
}
