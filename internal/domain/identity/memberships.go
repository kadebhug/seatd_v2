package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	MembershipScopeOrganisation = "organisation"
	MembershipScopeLocation     = "location"

	auditActionMembershipCreate     = "membership.create"
	auditActionMembershipUpdateRole = "membership.update_role"
	auditActionMembershipDisable    = "membership.disable"
	auditTargetMembership           = "membership"
)

type ManagedMembership struct {
	ID             uuid.UUID
	Scope          string
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	UserProfileID  uuid.UUID
	DisplayName    string
	Email          string
	MemberRef      string
	Role           string
	DisabledAt     pgtype.Timestamptz
	Permissions    []string
}

type CreateMembershipParams struct {
	TenantActor
	Scope       string
	LocationID  uuid.UUID
	Email       string
	DisplayName string
	Role        string
}

type UpdateMembershipRoleParams struct {
	TenantActor
	Scope string
	ID    uuid.UUID
	Role  string
}

type DisableMembershipParams struct {
	TenantActor
	Scope string
	ID    uuid.UUID
}

func (s *Service) ListManagedMemberships(ctx context.Context, actor TenantActor, includeDisabled bool) ([]ManagedMembership, error) {
	actor = normalizeTenantActor(actor)
	if err := validateTenantActor(actor, false); err != nil {
		return nil, err
	}

	var memberships []ManagedMembership
	err := s.inTenantDBTx(ctx, actor.OrganisationID, uuid.Nil, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireOrganisationPermission(ctx, q, actor, PermissionOrganisationManage); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `
SELECT om.id, 'organisation' AS scope, om.organisation_id, NULL::uuid AS location_id, om.user_profile_id,
       up.display_name, up.email, om.member_ref, om.role, om.disabled_at
FROM organisation_memberships om
LEFT JOIN user_profiles up ON up.id = om.user_profile_id
WHERE om.organisation_id = $1
  AND ($2::boolean OR om.disabled_at IS NULL)
UNION ALL
SELECT lm.id, 'location' AS scope, lm.organisation_id, lm.location_id, lm.user_profile_id,
       up.display_name, up.email, lm.member_ref, lm.role, lm.disabled_at
FROM location_memberships lm
LEFT JOIN user_profiles up ON up.id = lm.user_profile_id
WHERE lm.organisation_id = $1
  AND ($2::boolean OR lm.disabled_at IS NULL)
ORDER BY scope, role, display_name, member_ref
`, actor.OrganisationID, includeDisabled)
		if err != nil {
			return fmt.Errorf("listing memberships: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			membership, err := scanManagedMembership(rows)
			if err != nil {
				return err
			}
			memberships = append(memberships, membership)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterating memberships: %w", err)
		}
		return hydrateMembershipPermissions(ctx, q, memberships)
	})
	return memberships, err
}

func (s *Service) CreateMembership(ctx context.Context, arg CreateMembershipParams) (ManagedMembership, error) {
	arg.TenantActor = normalizeTenantActor(arg.TenantActor)
	arg.Scope = strings.TrimSpace(arg.Scope)
	arg.Role = strings.TrimSpace(arg.Role)
	arg.DisplayName = strings.TrimSpace(arg.DisplayName)
	email, err := normalizeEmail(arg.Email)
	if err != nil {
		return ManagedMembership{}, err
	}
	if err := validateTenantActor(arg.TenantActor, false); err != nil {
		return ManagedMembership{}, err
	}
	if !validMembershipRole(arg.Scope, arg.Role) {
		return ManagedMembership{}, ErrValidation
	}
	if arg.Scope == MembershipScopeLocation && arg.LocationID == uuid.Nil {
		return ManagedMembership{}, ErrValidation
	}
	if arg.Scope == MembershipScopeOrganisation && arg.LocationID != uuid.Nil {
		return ManagedMembership{}, ErrValidation
	}
	if arg.DisplayName == "" {
		arg.DisplayName = email
	}

	var membership ManagedMembership
	err = s.inTenantDBTx(ctx, arg.OrganisationID, uuid.Nil, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireOrganisationPermission(ctx, q, arg.TenantActor, PermissionOrganisationManage); err != nil {
			return err
		}
		profile, err := getOrCreateUserProfileByEmail(ctx, tx, q, email, arg.DisplayName)
		if err != nil {
			return err
		}
		memberRef := userProfileActorRef(profile.ID)
		switch arg.Scope {
		case MembershipScopeOrganisation:
			membership, err = createOrReactivateOrganisationMembership(ctx, tx, q, arg, profile, memberRef)
		case MembershipScopeLocation:
			if _, err := q.GetLocation(ctx, db.GetLocationParams{ID: arg.LocationID, OrganisationID: arg.OrganisationID}); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrNotFound
				}
				return fmt.Errorf("getting location: %w", err)
			}
			membership, err = createOrReactivateLocationMembership(ctx, tx, q, arg, profile, memberRef)
		default:
			return ErrValidation
		}
		if err != nil {
			return err
		}
		return recordMembershipAudit(ctx, q, arg.TenantActor, auditActionMembershipCreate, membership, map[string]any{
			"scope":      membership.Scope,
			"role":       membership.Role,
			"email":      membership.Email,
			"locationId": nullableAuditString(membership.LocationID),
		})
	})
	if mapped := mapIdentityConstraintError(err); mapped != nil {
		return ManagedMembership{}, mapped
	}
	return membership, err
}

func (s *Service) UpdateMembershipRole(ctx context.Context, arg UpdateMembershipRoleParams) (ManagedMembership, error) {
	arg.TenantActor = normalizeTenantActor(arg.TenantActor)
	arg.Scope = strings.TrimSpace(arg.Scope)
	arg.Role = strings.TrimSpace(arg.Role)
	if err := validateTenantActor(arg.TenantActor, false); err != nil {
		return ManagedMembership{}, err
	}
	if arg.ID == uuid.Nil || !validMembershipRole(arg.Scope, arg.Role) {
		return ManagedMembership{}, ErrValidation
	}

	var membership ManagedMembership
	err := s.inTenantDBTx(ctx, arg.OrganisationID, uuid.Nil, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireOrganisationPermission(ctx, q, arg.TenantActor, PermissionOrganisationManage); err != nil {
			return err
		}
		current, err := getManagedMembershipForUpdate(ctx, tx, arg.OrganisationID, arg.Scope, arg.ID)
		if err != nil {
			return err
		}
		if current.DisabledAt.Valid {
			return ErrNotFound
		}
		if current.Scope == MembershipScopeOrganisation && current.Role == RoleOrganisationOwner && arg.Role != RoleOrganisationOwner {
			if err := requireAnotherOrganisationOwner(ctx, tx, arg.OrganisationID); err != nil {
				return err
			}
		}
		membership, err = updateMembershipRole(ctx, tx, arg.OrganisationID, arg.Scope, arg.ID, arg.Role)
		if err != nil {
			return err
		}
		return recordMembershipAudit(ctx, q, arg.TenantActor, auditActionMembershipUpdateRole, membership, map[string]any{
			"scope":        membership.Scope,
			"previousRole": current.Role,
			"role":         membership.Role,
			"email":        membership.Email,
			"locationId":   nullableAuditString(membership.LocationID),
		})
	})
	return membership, err
}

func (s *Service) DisableMembership(ctx context.Context, arg DisableMembershipParams) (ManagedMembership, error) {
	arg.TenantActor = normalizeTenantActor(arg.TenantActor)
	arg.Scope = strings.TrimSpace(arg.Scope)
	if err := validateTenantActor(arg.TenantActor, false); err != nil {
		return ManagedMembership{}, err
	}
	if arg.ID == uuid.Nil || !validMembershipScope(arg.Scope) {
		return ManagedMembership{}, ErrValidation
	}

	var membership ManagedMembership
	err := s.inTenantDBTx(ctx, arg.OrganisationID, uuid.Nil, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireOrganisationPermission(ctx, q, arg.TenantActor, PermissionOrganisationManage); err != nil {
			return err
		}
		current, err := getManagedMembershipForUpdate(ctx, tx, arg.OrganisationID, arg.Scope, arg.ID)
		if err != nil {
			return err
		}
		if current.DisabledAt.Valid {
			return ErrNotFound
		}
		if current.Scope == MembershipScopeOrganisation && current.Role == RoleOrganisationOwner {
			if err := requireAnotherOrganisationOwner(ctx, tx, arg.OrganisationID); err != nil {
				return err
			}
		}
		membership, err = disableMembership(ctx, tx, arg.OrganisationID, arg.Scope, arg.ID)
		if err != nil {
			return err
		}
		return recordMembershipAudit(ctx, q, arg.TenantActor, auditActionMembershipDisable, membership, map[string]any{
			"scope":      membership.Scope,
			"role":       membership.Role,
			"email":      membership.Email,
			"locationId": nullableAuditString(membership.LocationID),
		})
	})
	return membership, err
}

func (s *Service) inTenantDBTx(ctx context.Context, organisationID, locationID uuid.UUID, fn func(pgx.Tx, *db.Queries) error) error {
	return s.inTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		locationValue := ""
		if locationID != uuid.Nil {
			locationValue = locationID.String()
		}
		if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, organisationID.String(), locationValue); err != nil {
			return fmt.Errorf("setting tenant context: %w", err)
		}
		return fn(tx, q)
	})
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanManagedMembership(row rowScanner) (ManagedMembership, error) {
	var (
		membership  ManagedMembership
		locationID  uuid.NullUUID
		userID      uuid.NullUUID
		displayName pgtype.Text
		email       pgtype.Text
	)
	if err := row.Scan(
		&membership.ID,
		&membership.Scope,
		&membership.OrganisationID,
		&locationID,
		&userID,
		&displayName,
		&email,
		&membership.MemberRef,
		&membership.Role,
		&membership.DisabledAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ManagedMembership{}, ErrNotFound
		}
		return ManagedMembership{}, fmt.Errorf("scanning membership: %w", err)
	}
	if locationID.Valid {
		membership.LocationID = locationID.UUID
	}
	if userID.Valid {
		membership.UserProfileID = userID.UUID
	}
	if displayName.Valid {
		membership.DisplayName = displayName.String
	}
	if email.Valid {
		membership.Email = email.String
	}
	return membership, nil
}

func getOrCreateUserProfileByEmail(ctx context.Context, tx pgx.Tx, q *db.Queries, email, displayName string) (db.UserProfile, error) {
	profile, err := getUserProfileByEmail(ctx, tx, email)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return db.UserProfile{}, err
	}
	profile, err = q.CreateUserProfile(ctx, db.CreateUserProfileParams{
		DisplayName: displayName,
		Email:       pgtype.Text{String: email, Valid: true},
	})
	if err != nil {
		return db.UserProfile{}, fmt.Errorf("creating user profile: %w", err)
	}
	return profile, nil
}

func getUserProfileByEmail(ctx context.Context, tx pgx.Tx, email string) (db.UserProfile, error) {
	var profile db.UserProfile
	err := tx.QueryRow(ctx, `
SELECT id, display_name, email, status, created_at, updated_at
FROM user_profiles
WHERE lower(btrim(email)) = $1
  AND status = 'active'
`, email).Scan(&profile.ID, &profile.DisplayName, &profile.Email, &profile.Status, &profile.CreatedAt, &profile.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.UserProfile{}, ErrNotFound
		}
		return db.UserProfile{}, fmt.Errorf("getting user profile by email: %w", err)
	}
	return profile, nil
}

func createOrReactivateOrganisationMembership(
	ctx context.Context,
	tx pgx.Tx,
	q *db.Queries,
	arg CreateMembershipParams,
	profile db.UserProfile,
	memberRef string,
) (ManagedMembership, error) {
	existing, err := getMembershipByProfile(ctx, tx, arg.OrganisationID, MembershipScopeOrganisation, uuid.Nil, profile.ID)
	if err == nil {
		if !existing.DisabledAt.Valid {
			return ManagedMembership{}, ErrAlreadyExists
		}
		return updateMembershipEnabled(ctx, tx, arg.OrganisationID, MembershipScopeOrganisation, existing.ID, arg.Role)
	}
	if !errors.Is(err, ErrNotFound) {
		return ManagedMembership{}, err
	}
	if _, err := q.CreateUserOrganisationMembership(ctx, db.CreateUserOrganisationMembershipParams{
		OrganisationID: arg.OrganisationID,
		UserProfileID:  nullableUUID(profile.ID),
		MemberRef:      memberRef,
		Role:           arg.Role,
	}); err != nil {
		return ManagedMembership{}, fmt.Errorf("creating organisation membership: %w", err)
	}
	return getMembershipByProfile(ctx, tx, arg.OrganisationID, MembershipScopeOrganisation, uuid.Nil, profile.ID)
}

func createOrReactivateLocationMembership(
	ctx context.Context,
	tx pgx.Tx,
	q *db.Queries,
	arg CreateMembershipParams,
	profile db.UserProfile,
	memberRef string,
) (ManagedMembership, error) {
	existing, err := getMembershipByProfile(ctx, tx, arg.OrganisationID, MembershipScopeLocation, arg.LocationID, profile.ID)
	if err == nil {
		if !existing.DisabledAt.Valid {
			return ManagedMembership{}, ErrAlreadyExists
		}
		return updateMembershipEnabled(ctx, tx, arg.OrganisationID, MembershipScopeLocation, existing.ID, arg.Role)
	}
	if !errors.Is(err, ErrNotFound) {
		return ManagedMembership{}, err
	}
	if _, err := q.CreateUserLocationMembership(ctx, db.CreateUserLocationMembershipParams{
		OrganisationID: arg.OrganisationID,
		LocationID:     arg.LocationID,
		UserProfileID:  nullableUUID(profile.ID),
		MemberRef:      memberRef,
		Role:           arg.Role,
	}); err != nil {
		return ManagedMembership{}, fmt.Errorf("creating location membership: %w", err)
	}
	return getMembershipByProfile(ctx, tx, arg.OrganisationID, MembershipScopeLocation, arg.LocationID, profile.ID)
}

func getMembershipByProfile(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, scope string, locationID uuid.UUID, userProfileID uuid.UUID) (ManagedMembership, error) {
	query := `
SELECT om.id, 'organisation' AS scope, om.organisation_id, NULL::uuid AS location_id, om.user_profile_id,
       up.display_name, up.email, om.member_ref, om.role, om.disabled_at
FROM organisation_memberships om
JOIN user_profiles up ON up.id = om.user_profile_id
WHERE om.organisation_id = $1 AND om.user_profile_id = $2
`
	args := []any{organisationID, nullableUUID(userProfileID)}
	if scope == MembershipScopeLocation {
		query = `
SELECT lm.id, 'location' AS scope, lm.organisation_id, lm.location_id, lm.user_profile_id,
       up.display_name, up.email, lm.member_ref, lm.role, lm.disabled_at
FROM location_memberships lm
JOIN user_profiles up ON up.id = lm.user_profile_id
WHERE lm.organisation_id = $1 AND lm.location_id = $2 AND lm.user_profile_id = $3
`
		args = []any{organisationID, locationID, nullableUUID(userProfileID)}
	}
	return scanManagedMembership(tx.QueryRow(ctx, query, args...))
}

func getManagedMembershipForUpdate(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, scope string, id uuid.UUID) (ManagedMembership, error) {
	query := `
SELECT om.id, 'organisation' AS scope, om.organisation_id, NULL::uuid AS location_id, om.user_profile_id,
       up.display_name, up.email, om.member_ref, om.role, om.disabled_at
FROM organisation_memberships om
LEFT JOIN user_profiles up ON up.id = om.user_profile_id
WHERE om.organisation_id = $1 AND om.id = $2
FOR UPDATE OF om
`
	if scope == MembershipScopeLocation {
		query = `
SELECT lm.id, 'location' AS scope, lm.organisation_id, lm.location_id, lm.user_profile_id,
       up.display_name, up.email, lm.member_ref, lm.role, lm.disabled_at
FROM location_memberships lm
LEFT JOIN user_profiles up ON up.id = lm.user_profile_id
WHERE lm.organisation_id = $1 AND lm.id = $2
FOR UPDATE OF lm
`
	}
	if !validMembershipScope(scope) {
		return ManagedMembership{}, ErrValidation
	}
	return scanManagedMembership(tx.QueryRow(ctx, query, organisationID, id))
}

func updateMembershipRole(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, scope string, id uuid.UUID, role string) (ManagedMembership, error) {
	query := `
UPDATE organisation_memberships
SET role = $3, updated_at = now()
WHERE organisation_id = $1 AND id = $2 AND disabled_at IS NULL
`
	if scope == MembershipScopeLocation {
		query = `
UPDATE location_memberships
SET role = $3, updated_at = now()
WHERE organisation_id = $1 AND id = $2 AND disabled_at IS NULL
`
	}
	if _, err := tx.Exec(ctx, query, organisationID, id, role); err != nil {
		return ManagedMembership{}, fmt.Errorf("updating membership role: %w", err)
	}
	return getManagedMembershipForUpdate(ctx, tx, organisationID, scope, id)
}

func updateMembershipEnabled(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, scope string, id uuid.UUID, role string) (ManagedMembership, error) {
	query := `
UPDATE organisation_memberships
SET role = $3, disabled_at = NULL, updated_at = now()
WHERE organisation_id = $1 AND id = $2
`
	if scope == MembershipScopeLocation {
		query = `
UPDATE location_memberships
SET role = $3, disabled_at = NULL, updated_at = now()
WHERE organisation_id = $1 AND id = $2
`
	}
	if _, err := tx.Exec(ctx, query, organisationID, id, role); err != nil {
		return ManagedMembership{}, fmt.Errorf("reactivating membership: %w", err)
	}
	return getManagedMembershipForUpdate(ctx, tx, organisationID, scope, id)
}

func disableMembership(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, scope string, id uuid.UUID) (ManagedMembership, error) {
	query := `
UPDATE organisation_memberships
SET disabled_at = now(), updated_at = now()
WHERE organisation_id = $1 AND id = $2 AND disabled_at IS NULL
`
	if scope == MembershipScopeLocation {
		query = `
UPDATE location_memberships
SET disabled_at = now(), updated_at = now()
WHERE organisation_id = $1 AND id = $2 AND disabled_at IS NULL
`
	}
	if _, err := tx.Exec(ctx, query, organisationID, id); err != nil {
		return ManagedMembership{}, fmt.Errorf("disabling membership: %w", err)
	}
	return getManagedMembershipForUpdate(ctx, tx, organisationID, scope, id)
}

func requireAnotherOrganisationOwner(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID) error {
	var ownerCount int
	if err := tx.QueryRow(ctx, `
SELECT count(*)::integer
FROM organisation_memberships
WHERE organisation_id = $1
  AND role = 'organisation_owner'
  AND disabled_at IS NULL
`, organisationID).Scan(&ownerCount); err != nil {
		return fmt.Errorf("counting organisation owners: %w", err)
	}
	if ownerCount <= 1 {
		return ErrValidation
	}
	return nil
}

func recordMembershipAudit(ctx context.Context, q *db.Queries, actor TenantActor, action string, membership ManagedMembership, metadata map[string]any) error {
	payload, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("encoding membership audit metadata: %w", err)
	}
	locationID := uuid.NullUUID{}
	if membership.LocationID != uuid.Nil {
		locationID = nullableUUID(membership.LocationID)
	}
	if _, err := q.RecordAuditEvent(ctx, db.RecordAuditEventParams{
		OrganisationID: nullableUUID(actor.OrganisationID),
		LocationID:     locationID,
		ActorRef:       actor.ActorRef,
		Action:         action,
		TargetType:     auditTargetMembership,
		TargetID:       membership.ID.String(),
		Metadata:       payload,
	}); err != nil {
		return fmt.Errorf("recording membership audit event: %w", err)
	}
	return nil
}

func hydrateMembershipPermissions(ctx context.Context, q *db.Queries, memberships []ManagedMembership) error {
	cache := map[string][]string{}
	for i := range memberships {
		permissions, ok := cache[memberships[i].Role]
		if !ok {
			var err error
			permissions, err = q.GetRolePermissions(ctx, memberships[i].Role)
			if err != nil {
				return fmt.Errorf("getting role permissions: %w", err)
			}
			cache[memberships[i].Role] = permissions
		}
		memberships[i].Permissions = append([]string(nil), permissions...)
	}
	return nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrValidation
	}
	address, err := mail.ParseAddress(value)
	if err != nil || strings.TrimSpace(address.Address) == "" {
		return "", ErrValidation
	}
	return strings.ToLower(address.Address), nil
}

func validMembershipScope(scope string) bool {
	return scope == MembershipScopeOrganisation || scope == MembershipScopeLocation
}

func validMembershipRole(scope string, role string) bool {
	switch scope {
	case MembershipScopeOrganisation:
		return role == RoleOrganisationOwner || role == RoleReadOnly
	case MembershipScopeLocation:
		return role == RoleLocationManager || role == RoleWaiter
	default:
		return false
	}
}

func nullableAuditString(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id.String()
}
