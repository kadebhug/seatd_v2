package identity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kadebhug/seatd_v2/internal/store/db"
)

const (
	RolePlatformAdmin     = "platform_admin"
	RoleOrganisationOwner = "organisation_owner"
	RoleLocationManager   = "location_manager"
	RoleWaiter            = "waiter"
	RoleReadOnly          = "read_only"
	RoleSupport           = "support"

	PermissionPlatformAdmin        = "platform.admin"
	PermissionPlatformAdminWrite   = "platform.admin.write"
	PermissionPlatformManageAdmins = "platform.admin.manage_admins"
	PermissionOrganisationManage   = "organisation.manage"
	PermissionLocationManage       = "location.manage"
	PermissionLayoutRead           = "layout.read"
	PermissionLayoutWrite          = "layout.write"
	PermissionOperationsRead       = "operations.read"
	PermissionOperationsWrite      = "operations.write"
	PermissionDeviceManage         = "device.manage"
	PermissionAuditRead            = "audit.read"
	PermissionAnalyticsRead        = "analytics.read"
	PermissionIntegrationsRead     = "integrations.read"
	PermissionIntegrationsManage   = "integrations.manage"

	DeviceTypeWaiterMobile  = "waiter_mobile"
	DeviceTypeManagerTablet = "manager_tablet"
	DeviceTypeDisplay       = "display"
	DeviceTypeHostDevice    = "host_device"

	DeviceTrustPending = "pending"
	DeviceTrustTrusted = "trusted"
	DeviceTrustRevoked = "revoked"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrForbidden      = errors.New("forbidden")
	ErrAlreadyExists  = errors.New("already exists")
	ErrAlreadyRevoked = errors.New("already revoked")
	ErrValidation     = errors.New("validation failed")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type TenantActor struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	ActorRef       string
}

type ExternalIdentity struct {
	Issuer  string
	Subject string
	Email   string
}

type UserProfileParams struct {
	DisplayName string
	Email       string
	External    ExternalIdentity
}

type UserProfileResult struct {
	User     db.UserProfile
	External db.ExternalIdentity
}

func (s *Service) CreateUserProfile(ctx context.Context, arg UserProfileParams) (UserProfileResult, error) {
	var result UserProfileResult
	err := s.inAdminTx(ctx, func(q *db.Queries) error {
		user, err := q.CreateUserProfile(ctx, db.CreateUserProfileParams{
			DisplayName: arg.DisplayName,
			Email:       nullableText(arg.Email),
		})
		if err != nil {
			return fmt.Errorf("creating user profile: %w", err)
		}
		external, err := q.LinkExternalIdentity(ctx, db.LinkExternalIdentityParams{
			UserProfileID: user.ID,
			Issuer:        arg.External.Issuer,
			Subject:       arg.External.Subject,
			Email:         nullableText(arg.External.Email),
		})
		if err != nil {
			return fmt.Errorf("linking external identity: %w", err)
		}
		result = UserProfileResult{User: user, External: external}
		return nil
	})
	return result, err
}

type Membership struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	MemberRef      string
	Role           string
}

type WebSession struct {
	Session              db.WebSession
	User                 db.UserProfile
	Secret               string
	Memberships          []Membership
	RotatedFromSessionID uuid.NullUUID
}

type OnboardOwnerParams struct {
	UserProfileID    uuid.UUID
	OrganisationName string
	OrganisationSlug string
	LocationName     string
	LocationSlug     string
	Timezone         string
	Floor            OnboardingFloor
	ServicePeriods   []OnboardingServicePeriod
	Staff            []OnboardingStaffMember
}

type OnboardingFloor struct {
	Name      string
	Slug      string
	Canvas    json.RawMessage
	Zones     []OnboardingZone
	Tables    []OnboardingTable
	SortOrder int32
}

type OnboardingZone struct {
	Name      string
	SortOrder int32
}

type OnboardingTable struct {
	Label         string
	CapacityLabel string
	Shape         string
	Geometry      json.RawMessage
	ZoneName      string
}

type OnboardingServicePeriod struct {
	Name       string
	DaysOfWeek []int16
	StartTime  string
	EndTime    string
}

type OnboardingStaffMember struct {
	Email      string
	Name       string
	Role       string
	LocationID uuid.UUID
}

type OnboardOwnerResult struct {
	Organisation   db.Organisation
	Location       db.Location
	Floor          *db.Floor
	Zones          []db.Zone
	Tables         []db.Table
	ServicePeriods []db.ServicePeriod
	Staff          []OnboardingStaffMember
	WebSession     WebSession
}

type PlatformProvisionTenantParams struct {
	OrganisationName string
	OrganisationSlug string
	LocationName     string
	LocationSlug     string
	Timezone         string
	OwnerEmail       string
	OwnerDisplayName string
}

type PlatformProvisionTenantResult struct {
	Organisation db.Organisation
	Location     db.Location
	Owner        ManagedMembership
}

type OwnerSetupParams struct {
	UserProfileID  uuid.UUID
	Floor          OnboardingFloor
	ServicePeriods []OnboardingServicePeriod
}

type OwnerSetupResult struct {
	Organisation   db.Organisation
	Location       db.Location
	Floor          *db.Floor
	Zones          []db.Zone
	Tables         []db.Table
	ServicePeriods []db.ServicePeriod
}

type ResolveExternalIdentityParams struct {
	DisplayName   string
	Email         string
	EmailVerified bool
	External      ExternalIdentity
}

func (s *Service) ResolveExternalIdentity(ctx context.Context, arg ResolveExternalIdentityParams) (UserProfileResult, error) {
	arg.DisplayName = strings.TrimSpace(arg.DisplayName)
	arg.Email = strings.TrimSpace(arg.Email)
	arg.External.Issuer = strings.TrimSpace(arg.External.Issuer)
	arg.External.Subject = strings.TrimSpace(arg.External.Subject)
	arg.External.Email = strings.TrimSpace(arg.External.Email)
	if arg.External.Email == "" {
		arg.External.Email = arg.Email
	}
	if arg.DisplayName == "" {
		arg.DisplayName = arg.Email
	}
	if arg.DisplayName == "" || arg.External.Issuer == "" || arg.External.Subject == "" {
		return UserProfileResult{}, ErrValidation
	}

	var result UserProfileResult
	err := s.inAdminTx(ctx, func(q *db.Queries) error {
		row, err := q.GetUserByExternalIdentity(ctx, db.GetUserByExternalIdentityParams{
			Issuer:  arg.External.Issuer,
			Subject: arg.External.Subject,
		})
		if err == nil {
			result = UserProfileResult{User: row.UserProfile, External: row.ExternalIdentity}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("getting external identity: %w", err)
		} else if arg.EmailVerified && arg.Email != "" {
			normalizedEmail, err := normalizeEmail(arg.Email)
			if err == nil {
				existing, err := q.GetActiveUserProfileByEmail(ctx, nullableText(normalizedEmail))
				if err == nil {
					external, err := q.LinkExternalIdentity(ctx, db.LinkExternalIdentityParams{
						UserProfileID: existing.ID,
						Issuer:        arg.External.Issuer,
						Subject:       arg.External.Subject,
						Email:         nullableText(arg.External.Email),
					})
					if err != nil {
						return fmt.Errorf("linking invited external identity: %w", err)
					}
					result = UserProfileResult{User: existing, External: external}
				} else if !errors.Is(err, pgx.ErrNoRows) {
					return fmt.Errorf("getting invited user profile by email: %w", err)
				}
			}
		}

		if result.User.ID == uuid.Nil {
			profileEmail := arg.Email
			if !arg.EmailVerified {
				profileEmail = ""
			}
			created, err := q.CreateUserProfile(ctx, db.CreateUserProfileParams{
				DisplayName: arg.DisplayName,
				Email:       nullableText(profileEmail),
			})
			if err != nil {
				return fmt.Errorf("creating user profile: %w", err)
			}
			external, err := q.LinkExternalIdentity(ctx, db.LinkExternalIdentityParams{
				UserProfileID: created.ID,
				Issuer:        arg.External.Issuer,
				Subject:       arg.External.Subject,
				Email:         nullableText(arg.External.Email),
			})
			if err != nil {
				return fmt.Errorf("linking external identity: %w", err)
			}
			result = UserProfileResult{User: created, External: external}
		}
		if arg.EmailVerified && arg.Email != "" {
			if err := consumePendingPlatformGrant(ctx, q, result.User.ID, arg.Email); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func consumePendingPlatformGrant(ctx context.Context, q *db.Queries, userProfileID uuid.UUID, email string) error {
	normalizedEmail, err := normalizeEmail(email)
	if err != nil {
		return nil
	}
	grant, err := q.ConsumePlatformAdminGrant(ctx, db.ConsumePlatformAdminGrantParams{
		ConsumedByUserProfileID: nullableUUID(userProfileID),
		Btrim:                   normalizedEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("consuming platform admin grant: %w", err)
	}
	if _, err := q.CreatePlatformMembership(ctx, db.CreatePlatformMembershipParams{
		UserProfileID:     userProfileID,
		Role:              grant.Role,
		GrantedByActorRef: grant.InvitedByActorRef,
	}); err != nil {
		return fmt.Errorf("creating platform membership from grant: %w", err)
	}
	return nil
}

func (s *Service) OnboardOwner(ctx context.Context, arg OnboardOwnerParams) (OnboardOwnerResult, error) {
	arg, err := normalizeOnboardOwnerParams(arg)
	if err != nil {
		return OnboardOwnerResult{}, err
	}
	if err := validateOnboardOwnerParams(arg); err != nil {
		return OnboardOwnerResult{}, err
	}

	var result OnboardOwnerResult
	err = s.inAdminTx(ctx, func(q *db.Queries) error {
		existingOrgs, err := q.ListUserOrganisationMemberships(ctx, nullableUUID(arg.UserProfileID))
		if err != nil {
			return fmt.Errorf("listing existing organisation memberships: %w", err)
		}
		existingLocations, err := q.ListUserLocationMemberships(ctx, nullableUUID(arg.UserProfileID))
		if err != nil {
			return fmt.Errorf("listing existing location memberships: %w", err)
		}
		if len(existingOrgs) > 0 || len(existingLocations) > 0 {
			return ErrAlreadyExists
		}

		org, err := q.CreateOrganisation(ctx, db.CreateOrganisationParams{
			Slug: arg.OrganisationSlug,
			Name: arg.OrganisationName,
		})
		if err != nil {
			return fmt.Errorf("creating organisation: %w", err)
		}
		location, err := q.CreateLocation(ctx, db.CreateLocationParams{
			OrganisationID: org.ID,
			Slug:           arg.LocationSlug,
			Name:           arg.LocationName,
			Timezone:       arg.Timezone,
		})
		if err != nil {
			return fmt.Errorf("creating location: %w", err)
		}
		memberRef := userProfileActorRef(arg.UserProfileID)
		if _, err := q.CreateUserOrganisationMembership(ctx, db.CreateUserOrganisationMembershipParams{
			OrganisationID: org.ID,
			UserProfileID:  nullableUUID(arg.UserProfileID),
			MemberRef:      memberRef,
			Role:           RoleOrganisationOwner,
		}); err != nil {
			return fmt.Errorf("creating owner membership: %w", err)
		}

		floor, zones, tables, err := createOnboardingLayout(ctx, q, org.ID, location.ID, arg.Floor)
		if err != nil {
			return err
		}
		periods, err := createOnboardingServicePeriods(ctx, q, org.ID, location.ID, arg.ServicePeriods)
		if err != nil {
			return err
		}

		result = OnboardOwnerResult{
			Organisation:   org,
			Location:       location,
			Floor:          floor,
			Zones:          zones,
			Tables:         tables,
			ServicePeriods: periods,
			Staff:          arg.Staff,
			WebSession: WebSession{
				User: db.UserProfile{ID: arg.UserProfileID},
				Memberships: []Membership{{
					OrganisationID: org.ID,
					MemberRef:      memberRef,
					Role:           RoleOrganisationOwner,
				}},
			},
		}
		return nil
	})
	if mapped := mapIdentityConstraintError(err); mapped != nil {
		return OnboardOwnerResult{}, mapped
	}
	return result, err
}

func (s *Service) PlatformProvisionTenant(ctx context.Context, arg PlatformProvisionTenantParams) (PlatformProvisionTenantResult, error) {
	arg.OrganisationName = strings.TrimSpace(arg.OrganisationName)
	arg.OrganisationSlug = strings.TrimSpace(arg.OrganisationSlug)
	arg.LocationName = strings.TrimSpace(arg.LocationName)
	arg.LocationSlug = strings.TrimSpace(arg.LocationSlug)
	arg.Timezone = strings.TrimSpace(arg.Timezone)
	if arg.Timezone == "" {
		arg.Timezone = "UTC"
	}
	arg.OwnerDisplayName = strings.TrimSpace(arg.OwnerDisplayName)
	ownerEmail, err := normalizeEmail(arg.OwnerEmail)
	if err != nil {
		return PlatformProvisionTenantResult{}, err
	}
	if arg.OwnerDisplayName == "" {
		arg.OwnerDisplayName = ownerEmail
	}
	if arg.OrganisationName == "" || arg.LocationName == "" || !validSlug(arg.OrganisationSlug) || !validSlug(arg.LocationSlug) {
		return PlatformProvisionTenantResult{}, ErrValidation
	}
	if _, err := time.LoadLocation(arg.Timezone); err != nil {
		return PlatformProvisionTenantResult{}, ErrValidation
	}

	var result PlatformProvisionTenantResult
	err = s.inTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
			return fmt.Errorf("setting platform admin context: %w", err)
		}
		org, err := q.CreateOrganisation(ctx, db.CreateOrganisationParams{
			Slug: arg.OrganisationSlug,
			Name: arg.OrganisationName,
		})
		if err != nil {
			return fmt.Errorf("creating organisation: %w", err)
		}
		location, err := q.CreateLocation(ctx, db.CreateLocationParams{
			OrganisationID: org.ID,
			Slug:           arg.LocationSlug,
			Name:           arg.LocationName,
			Timezone:       arg.Timezone,
		})
		if err != nil {
			return fmt.Errorf("creating location: %w", err)
		}
		profile, err := getOrCreateUserProfileByEmail(ctx, tx, q, ownerEmail, arg.OwnerDisplayName)
		if err != nil {
			return err
		}
		owner, err := createOrReactivateOrganisationMembership(ctx, tx, q, CreateMembershipParams{
			TenantActor: TenantActor{OrganisationID: org.ID, ActorRef: "platform"},
			Scope:       MembershipScopeOrganisation,
			Role:        RoleOrganisationOwner,
		}, profile, userProfileActorRef(profile.ID))
		if err != nil {
			return err
		}
		result = PlatformProvisionTenantResult{Organisation: org, Location: location, Owner: owner}
		return nil
	})
	if mapped := mapIdentityConstraintError(err); mapped != nil {
		return PlatformProvisionTenantResult{}, mapped
	}
	return result, err
}

func (s *Service) OwnerSetup(ctx context.Context, arg OwnerSetupParams) (OwnerSetupResult, error) {
	if arg.UserProfileID == uuid.Nil {
		return OwnerSetupResult{}, ErrValidation
	}
	onboarding := OnboardOwnerParams{
		UserProfileID:  arg.UserProfileID,
		Floor:          arg.Floor,
		ServicePeriods: arg.ServicePeriods,
	}
	normalized, err := normalizeOnboardOwnerParams(onboarding)
	if err != nil {
		return OwnerSetupResult{}, err
	}
	if err := validateOwnerSetupParams(normalized); err != nil {
		return OwnerSetupResult{}, err
	}

	var result OwnerSetupResult
	err = s.inAdminTx(ctx, func(q *db.Queries) error {
		orgs, err := q.ListUserOrganisationMemberships(ctx, nullableUUID(arg.UserProfileID))
		if err != nil {
			return fmt.Errorf("listing owner memberships: %w", err)
		}
		var ownerOrgID uuid.UUID
		for _, membership := range orgs {
			if membership.Role == RoleOrganisationOwner {
				ownerOrgID = membership.OrganisationID
				break
			}
		}
		if ownerOrgID == uuid.Nil {
			return ErrForbidden
		}
		org, err := q.GetOrganisation(ctx, ownerOrgID)
		if err != nil {
			return fmt.Errorf("getting owner organisation: %w", err)
		}
		if org.OnboardedAt.Valid {
			return ErrAlreadyExists
		}
		locations, err := q.ListLocationsByOrganisation(ctx, ownerOrgID)
		if err != nil {
			return fmt.Errorf("listing organisation locations: %w", err)
		}
		if len(locations) == 0 {
			return ErrNotFound
		}
		location := locations[0]
		floor, zones, tables, err := createOnboardingLayout(ctx, q, ownerOrgID, location.ID, normalized.Floor)
		if err != nil {
			return err
		}
		periods, err := createOnboardingServicePeriods(ctx, q, ownerOrgID, location.ID, normalized.ServicePeriods)
		if err != nil {
			return err
		}
		org, err = q.SetOrganisationOnboarded(ctx, ownerOrgID)
		if err != nil {
			return fmt.Errorf("marking organisation onboarded: %w", err)
		}
		result = OwnerSetupResult{
			Organisation:   org,
			Location:       location,
			Floor:          floor,
			Zones:          zones,
			Tables:         tables,
			ServicePeriods: periods,
		}
		return nil
	})
	if mapped := mapIdentityConstraintError(err); mapped != nil {
		return OwnerSetupResult{}, mapped
	}
	return result, err
}

func validateOwnerSetupParams(arg OnboardOwnerParams) error {
	if arg.UserProfileID == uuid.Nil || !validSlug(arg.Floor.Slug) || !jsonObject(arg.Floor.Canvas) {
		return ErrValidation
	}
	zoneNames := map[string]bool{}
	for _, zone := range arg.Floor.Zones {
		if zone.Name == "" || zoneNames[zone.Name] {
			return ErrValidation
		}
		zoneNames[zone.Name] = true
	}
	for _, table := range arg.Floor.Tables {
		if table.Label == "" ||
			table.CapacityLabel == "" ||
			!validTableShape(table.Shape) ||
			!jsonObject(table.Geometry) {
			return ErrValidation
		}
		if table.ZoneName != "" && !zoneNames[table.ZoneName] {
			return ErrValidation
		}
	}
	for _, period := range arg.ServicePeriods {
		if _, _, _, _, err := parseOnboardingServicePeriod(period); err != nil {
			return err
		}
	}
	return nil
}

type CreateWebSessionParams struct {
	UserProfileID uuid.UUID
	TTL           time.Duration
	UserAgent     string
	IPAddress     string
	RotatedFrom   uuid.UUID
}

func (s *Service) CreateWebSession(ctx context.Context, arg CreateWebSessionParams) (WebSession, error) {
	if arg.UserProfileID == uuid.Nil {
		return WebSession{}, ErrValidation
	}
	if arg.TTL <= 0 {
		arg.TTL = 8 * time.Hour
	}
	if arg.TTL < time.Minute || arg.TTL > 24*time.Hour {
		return WebSession{}, ErrValidation
	}
	secret, err := GenerateOpaqueSecret()
	if err != nil {
		return WebSession{}, fmt.Errorf("generating web session: %w", err)
	}
	prefix, err := LookupPrefix(secret)
	if err != nil {
		return WebSession{}, err
	}
	hash, err := HashSecret(secret)
	if err != nil {
		return WebSession{}, err
	}
	var ip *netip.Addr
	if strings.TrimSpace(arg.IPAddress) != "" {
		addr, err := netip.ParseAddr(strings.TrimSpace(arg.IPAddress))
		if err == nil {
			ip = &addr
		}
	}

	var session db.WebSession
	err = s.inAdminTx(ctx, func(q *db.Queries) error {
		var err error
		session, err = q.CreateWebSession(ctx, db.CreateWebSessionParams{
			UserProfileID:        arg.UserProfileID,
			LookupPrefix:         prefix,
			SessionHash:          hash,
			ExpiresAt:            pgtype.Timestamptz{Time: time.Now().Add(arg.TTL), Valid: true},
			UserAgent:            nullableText(strings.TrimSpace(arg.UserAgent)),
			IpAddress:            ip,
			RotatedFromSessionID: nullableUUID(arg.RotatedFrom),
		})
		if err != nil {
			return fmt.Errorf("creating web session: %w", err)
		}
		return nil
	})
	if err != nil {
		return WebSession{}, err
	}
	return s.hydrateWebSession(ctx, session, secret)
}

func (s *Service) ValidateWebSession(ctx context.Context, secret string) (WebSession, error) {
	prefix, err := LookupPrefix(secret)
	if err != nil {
		return WebSession{}, ErrUnauthorized
	}

	var row db.GetWebSessionByPrefixRow
	err = s.inAdminTx(ctx, func(q *db.Queries) error {
		var err error
		row, err = q.GetWebSessionByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return fmt.Errorf("getting web session: %w", err)
		}
		ok, err := SecretMatches(secret, row.WebSession.SessionHash)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUnauthorized
		}
		if err := q.TouchWebSession(ctx, row.WebSession.ID); err != nil {
			return fmt.Errorf("touching web session: %w", err)
		}
		return nil
	})
	if err != nil {
		return WebSession{}, err
	}

	session, err := s.hydrateWebSession(ctx, row.WebSession, secret)
	if err != nil {
		return WebSession{}, err
	}
	session.User = row.UserProfile
	return session, nil
}

func (s *Service) RotateWebSession(ctx context.Context, secret string, ttl time.Duration, userAgent, ipAddress string) (WebSession, error) {
	current, err := s.ValidateWebSession(ctx, secret)
	if err != nil {
		return WebSession{}, err
	}
	next, err := s.CreateWebSession(ctx, CreateWebSessionParams{
		UserProfileID: current.User.ID,
		TTL:           ttl,
		UserAgent:     userAgent,
		IPAddress:     ipAddress,
		RotatedFrom:   current.Session.ID,
	})
	if err != nil {
		return WebSession{}, err
	}
	next.User = current.User
	if err := s.RevokeWebSession(ctx, secret); err != nil {
		return WebSession{}, err
	}
	return next, nil
}

func (s *Service) RevokeWebSession(ctx context.Context, secret string) error {
	prefix, err := LookupPrefix(secret)
	if err != nil {
		return ErrUnauthorized
	}
	return s.inAdminTx(ctx, func(q *db.Queries) error {
		row, err := q.GetWebSessionByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("getting web session: %w", err)
		}
		ok, err := SecretMatches(secret, row.WebSession.SessionHash)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUnauthorized
		}
		if err := q.RevokeWebSession(ctx, row.WebSession.ID); err != nil {
			return fmt.Errorf("revoking web session: %w", err)
		}
		return nil
	})
}

func (s *Service) hydrateWebSession(ctx context.Context, session db.WebSession, secret string) (WebSession, error) {
	result := WebSession{Session: session, User: db.UserProfile{ID: session.UserProfileID}, Secret: secret}
	err := s.inAdminTx(ctx, func(q *db.Queries) error {
		orgs, err := q.ListUserOrganisationMemberships(ctx, nullableUUID(session.UserProfileID))
		if err != nil {
			return fmt.Errorf("listing organisation memberships: %w", err)
		}
		locations, err := q.ListUserLocationMemberships(ctx, nullableUUID(session.UserProfileID))
		if err != nil {
			return fmt.Errorf("listing location memberships: %w", err)
		}
		platform, err := q.ListUserPlatformMemberships(ctx, session.UserProfileID)
		if err != nil {
			return fmt.Errorf("listing platform memberships: %w", err)
		}
		result.Memberships = make([]Membership, 0, len(orgs)+len(locations)+len(platform))
		for _, membership := range orgs {
			result.Memberships = append(result.Memberships, Membership{
				OrganisationID: membership.OrganisationID,
				MemberRef:      membership.MemberRef,
				Role:           membership.Role,
			})
		}
		for _, membership := range platform {
			result.Memberships = append(result.Memberships, Membership{
				MemberRef: userProfileActorRef(membership.UserProfileID),
				Role:      membership.Role,
			})
		}
		for _, membership := range locations {
			result.Memberships = append(result.Memberships, Membership{
				OrganisationID: membership.OrganisationID,
				LocationID:     membership.LocationID,
				MemberRef:      membership.MemberRef,
				Role:           membership.Role,
			})
		}
		return nil
	})
	return result, err
}

func (s *Service) HasOrganisationPermission(
	ctx context.Context,
	organisationID uuid.UUID,
	userProfileID uuid.UUID,
	permission string,
) (bool, error) {
	var allowed bool
	err := s.inTenantTx(ctx, organisationID, uuid.Nil, func(q *db.Queries) error {
		value, err := q.UserHasOrganisationPermission(ctx, db.UserHasOrganisationPermissionParams{
			OrganisationID: organisationID,
			UserProfileID:  uuid.NullUUID{UUID: userProfileID, Valid: true},
			PermissionName: permission,
		})
		if err != nil {
			return fmt.Errorf("checking organisation permission: %w", err)
		}
		allowed = value
		return nil
	})
	return allowed, err
}

func (s *Service) HasLocationPermission(
	ctx context.Context,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	userProfileID uuid.UUID,
	permission string,
) (bool, error) {
	var allowed bool
	err := s.inTenantTx(ctx, organisationID, locationID, func(q *db.Queries) error {
		value, err := q.UserHasLocationPermission(ctx, db.UserHasLocationPermissionParams{
			OrganisationID: organisationID,
			LocationID:     locationID,
			UserProfileID:  uuid.NullUUID{UUID: userProfileID, Valid: true},
			PermissionName: permission,
		})
		if err != nil {
			return fmt.Errorf("checking location permission: %w", err)
		}
		allowed = value
		return nil
	})
	return allowed, err
}

type RegisterDeviceParams struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	UserProfileID  uuid.UUID
	DeviceType     string
	Platform       string
	AppVersion     string
	Name           string
	Capabilities   json.RawMessage
	Configuration  json.RawMessage
	HeartbeatEvery time.Duration
	ExpiresAt      time.Time
}

type RegisterDeviceResult struct {
	Device     db.Device
	Credential db.DeviceCredential
	Secret     string
}

func (s *Service) RegisterDevice(ctx context.Context, arg RegisterDeviceParams) (RegisterDeviceResult, error) {
	arg = normalizeRegisterDeviceParams(arg)
	if err := validateRegisterDeviceParams(arg); err != nil {
		return RegisterDeviceResult{}, err
	}
	secret, err := GenerateOpaqueSecret()
	if err != nil {
		return RegisterDeviceResult{}, fmt.Errorf("generating device credential: %w", err)
	}
	prefix, err := LookupPrefix(secret)
	if err != nil {
		return RegisterDeviceResult{}, err
	}
	hash, err := HashSecret(secret)
	if err != nil {
		return RegisterDeviceResult{}, err
	}

	var result RegisterDeviceResult
	err = s.inTenantTx(ctx, arg.OrganisationID, arg.LocationID, func(q *db.Queries) error {
		device, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
			OrganisationID:           arg.OrganisationID,
			LocationID:               nullableUUID(arg.LocationID),
			UserProfileID:            nullableUUID(arg.UserProfileID),
			DeviceType:               arg.DeviceType,
			Platform:                 arg.Platform,
			AppVersion:               arg.AppVersion,
			Name:                     nullableText(arg.Name),
			Capabilities:             arg.Capabilities,
			Configuration:            arg.Configuration,
			HeartbeatIntervalSeconds: int32(arg.HeartbeatEvery / time.Second),
		})
		if err != nil {
			return fmt.Errorf("registering device: %w", err)
		}
		credential, err := q.CreateDeviceCredential(ctx, db.CreateDeviceCredentialParams{
			OrganisationID: arg.OrganisationID,
			DeviceID:       device.ID,
			LookupPrefix:   prefix,
			CredentialHash: hash,
			ExpiresAt:      nullableTime(arg.ExpiresAt),
		})
		if err != nil {
			return fmt.Errorf("creating device credential: %w", err)
		}
		result = RegisterDeviceResult{Device: device, Credential: credential, Secret: secret}
		return nil
	})
	return result, err
}

type CreatePairingCodeParams struct {
	TenantActor
	DeviceType string
	TTL        time.Duration
}

type CreatePairingCodeResult struct {
	PairingCode db.DevicePairingCode
	Code        string
}

func (s *Service) CreatePairingCode(ctx context.Context, arg CreatePairingCodeParams) (CreatePairingCodeResult, error) {
	arg.TenantActor = normalizeTenantActor(arg.TenantActor)
	if err := validateTenantActor(arg.TenantActor, true); err != nil {
		return CreatePairingCodeResult{}, err
	}
	if arg.DeviceType == "" {
		arg.DeviceType = DeviceTypeDisplay
	}
	if !validDeviceType(arg.DeviceType) {
		return CreatePairingCodeResult{}, ErrValidation
	}
	if arg.TTL <= 0 {
		arg.TTL = 10 * time.Minute
	}
	if arg.TTL > time.Hour {
		return CreatePairingCodeResult{}, ErrValidation
	}
	code, err := GeneratePairingCode()
	if err != nil {
		return CreatePairingCodeResult{}, fmt.Errorf("generating pairing code: %w", err)
	}
	prefix, err := PairingCodeLookupPrefix(code)
	if err != nil {
		return CreatePairingCodeResult{}, err
	}
	hash, err := HashSecret(NormalizePairingCode(code))
	if err != nil {
		return CreatePairingCodeResult{}, err
	}

	var result CreatePairingCodeResult
	err = s.inTenantTx(ctx, arg.OrganisationID, arg.LocationID, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, PermissionDeviceManage); err != nil {
			return err
		}
		pairingCode, err := q.CreateDevicePairingCode(ctx, db.CreateDevicePairingCodeParams{
			OrganisationID:          arg.OrganisationID,
			LocationID:              arg.LocationID,
			CreatedBy:               arg.ActorRef,
			DeviceType:              arg.DeviceType,
			PairingCodeLookupPrefix: prefix,
			PairingCodeHash:         hash,
			ExpiresAt:               pgtype.Timestamptz{Time: time.Now().Add(arg.TTL), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("creating device pairing code: %w", err)
		}
		result = CreatePairingCodeResult{PairingCode: pairingCode, Code: code}
		return nil
	})
	return result, err
}

type PairDeviceParams struct {
	Code          string
	Platform      string
	AppVersion    string
	Name          string
	Capabilities  json.RawMessage
	Configuration json.RawMessage
}

func (s *Service) PairDevice(ctx context.Context, arg PairDeviceParams) (RegisterDeviceResult, error) {
	code := NormalizePairingCode(arg.Code)
	prefix, err := PairingCodeLookupPrefix(code)
	if err != nil {
		return RegisterDeviceResult{}, ErrUnauthorized
	}
	arg.Platform = strings.TrimSpace(arg.Platform)
	arg.AppVersion = strings.TrimSpace(arg.AppVersion)
	arg.Name = strings.TrimSpace(arg.Name)
	if arg.Platform == "" || arg.AppVersion == "" {
		return RegisterDeviceResult{}, ErrValidation
	}
	arg.Capabilities = defaultJSONObject(arg.Capabilities)
	arg.Configuration = defaultJSONObject(arg.Configuration)
	if !jsonObject(arg.Capabilities) || !jsonObject(arg.Configuration) {
		return RegisterDeviceResult{}, ErrValidation
	}

	secret, err := GenerateOpaqueSecret()
	if err != nil {
		return RegisterDeviceResult{}, fmt.Errorf("generating device credential: %w", err)
	}
	credentialPrefix, err := LookupPrefix(secret)
	if err != nil {
		return RegisterDeviceResult{}, err
	}
	credentialHash, err := HashSecret(secret)
	if err != nil {
		return RegisterDeviceResult{}, err
	}

	var result RegisterDeviceResult
	err = s.inAdminTx(ctx, func(q *db.Queries) error {
		pairingCode, err := q.GetDevicePairingCodeByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return fmt.Errorf("getting device pairing code: %w", err)
		}
		ok, err := SecretMatches(code, pairingCode.PairingCodeHash)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUnauthorized
		}
		device, err := q.RegisterDevice(ctx, db.RegisterDeviceParams{
			OrganisationID:           pairingCode.OrganisationID,
			LocationID:               nullableUUID(pairingCode.LocationID),
			UserProfileID:            uuid.NullUUID{},
			DeviceType:               pairingCode.DeviceType,
			Platform:                 arg.Platform,
			AppVersion:               arg.AppVersion,
			Name:                     nullableText(arg.Name),
			Capabilities:             arg.Capabilities,
			Configuration:            arg.Configuration,
			HeartbeatIntervalSeconds: 60,
		})
		if err != nil {
			return fmt.Errorf("registering paired device: %w", err)
		}
		if _, err := q.ConsumeDevicePairingCode(ctx, db.ConsumeDevicePairingCodeParams{
			ID:                 pairingCode.ID,
			ConsumedByDeviceID: nullableUUID(device.ID),
		}); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return fmt.Errorf("consuming device pairing code: %w", err)
		}
		device, err = q.TrustDevice(ctx, db.TrustDeviceParams{
			ID:             device.ID,
			OrganisationID: pairingCode.OrganisationID,
		})
		if err != nil {
			return fmt.Errorf("trusting paired device: %w", err)
		}
		credential, err := q.CreateDeviceCredential(ctx, db.CreateDeviceCredentialParams{
			OrganisationID: pairingCode.OrganisationID,
			DeviceID:       device.ID,
			LookupPrefix:   credentialPrefix,
			CredentialHash: credentialHash,
			ExpiresAt:      pgtype.Timestamptz{},
		})
		if err != nil {
			return fmt.Errorf("creating paired device credential: %w", err)
		}
		result = RegisterDeviceResult{Device: device, Credential: credential, Secret: secret}
		return nil
	})
	return result, err
}

type HeartbeatParams struct {
	Secret       string
	AppVersion   string
	Capabilities json.RawMessage
}

func (s *Service) RecordHeartbeat(ctx context.Context, arg HeartbeatParams) (db.Device, error) {
	arg.AppVersion = strings.TrimSpace(arg.AppVersion)
	if arg.AppVersion == "" {
		return db.Device{}, ErrValidation
	}
	arg.Capabilities = defaultJSONObject(arg.Capabilities)
	if !jsonObject(arg.Capabilities) {
		return db.Device{}, ErrValidation
	}
	device, err := s.AuthenticateDeviceCredential(ctx, arg.Secret)
	if err != nil {
		return db.Device{}, err
	}
	err = s.inTenantTx(ctx, device.OrganisationID, nullableUUIDValue(device.LocationID), func(q *db.Queries) error {
		updated, err := q.RecordDeviceHeartbeat(ctx, db.RecordDeviceHeartbeatParams{
			ID:             device.ID,
			OrganisationID: device.OrganisationID,
			AppVersion:     arg.AppVersion,
			Capabilities:   arg.Capabilities,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return fmt.Errorf("recording device heartbeat: %w", err)
		}
		device = updated
		return nil
	})
	return device, err
}

func (s *Service) ListDevices(ctx context.Context, actor TenantActor) ([]db.Device, error) {
	actor = normalizeTenantActor(actor)
	if err := validateTenantActor(actor, false); err != nil {
		return nil, err
	}
	var devices []db.Device
	err := s.inTenantTx(ctx, actor.OrganisationID, actor.LocationID, func(q *db.Queries) error {
		if actor.LocationID == uuid.Nil {
			if err := requireOrganisationPermission(ctx, q, actor, PermissionDeviceManage); err != nil {
				return err
			}
		} else {
			if err := requireLocationPermission(ctx, q, actor, PermissionDeviceManage); err != nil {
				return err
			}
		}
		var err error
		if actor.LocationID == uuid.Nil {
			devices, err = q.ListDevicesByOrganisation(ctx, actor.OrganisationID)
		} else {
			devices, err = q.ListDevicesByLocation(ctx, db.ListDevicesByLocationParams{
				OrganisationID: actor.OrganisationID,
				LocationID:     nullableUUID(actor.LocationID),
			})
		}
		if err != nil {
			return fmt.Errorf("listing devices: %w", err)
		}
		return nil
	})
	return devices, err
}

func (s *Service) TrustDevice(ctx context.Context, organisationID, deviceID uuid.UUID) (db.Device, error) {
	var device db.Device
	err := s.inTenantTx(ctx, organisationID, uuid.Nil, func(q *db.Queries) error {
		updated, err := q.TrustDevice(ctx, db.TrustDeviceParams{ID: deviceID, OrganisationID: organisationID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("trusting device: %w", err)
		}
		device = updated
		return nil
	})
	return device, err
}

func (s *Service) RevokeDevice(ctx context.Context, actor TenantActor, deviceID uuid.UUID) (db.Device, error) {
	actor = normalizeTenantActor(actor)
	if err := validateTenantActor(actor, false); err != nil {
		return db.Device{}, err
	}
	var device db.Device
	err := s.inTenantTx(ctx, actor.OrganisationID, uuid.Nil, func(q *db.Queries) error {
		current, err := q.GetDevice(ctx, db.GetDeviceParams{
			ID:             deviceID,
			OrganisationID: actor.OrganisationID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("getting device: %w", err)
		}
		if current.LocationID.Valid {
			scopedActor := actor
			scopedActor.LocationID = current.LocationID.UUID
			if err := requireLocationPermission(ctx, q, scopedActor, PermissionDeviceManage); err != nil {
				return err
			}
		} else if err := requireOrganisationPermission(ctx, q, actor, PermissionDeviceManage); err != nil {
			return err
		}
		if err := q.RevokeDeviceCredentials(ctx, db.RevokeDeviceCredentialsParams{
			OrganisationID: actor.OrganisationID,
			DeviceID:       deviceID,
		}); err != nil {
			return fmt.Errorf("revoking device credentials: %w", err)
		}
		updated, err := q.RevokeDevice(ctx, db.RevokeDeviceParams{ID: deviceID, OrganisationID: actor.OrganisationID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAlreadyRevoked
			}
			return fmt.Errorf("revoking device: %w", err)
		}
		device = updated
		return nil
	})
	return device, err
}

func (s *Service) AuthenticateDeviceCredential(ctx context.Context, secret string) (db.Device, error) {
	prefix, err := LookupPrefix(secret)
	if err != nil {
		return db.Device{}, err
	}

	var device db.Device
	err = s.inAdminTx(ctx, func(q *db.Queries) error {
		row, err := q.GetDeviceCredentialByPrefix(ctx, prefix)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrUnauthorized
			}
			return fmt.Errorf("getting device credential: %w", err)
		}
		ok, err := SecretMatches(secret, row.DeviceCredential.CredentialHash)
		if err != nil {
			return err
		}
		if !ok {
			return ErrUnauthorized
		}
		if _, err := q.MarkDeviceSeen(ctx, db.MarkDeviceSeenParams{
			ID:             row.Device.ID,
			OrganisationID: row.Device.OrganisationID,
		}); err != nil {
			return fmt.Errorf("marking device seen: %w", err)
		}
		if err := q.TouchDeviceCredential(ctx, row.DeviceCredential.ID); err != nil {
			return fmt.Errorf("touching device credential: %w", err)
		}
		device = row.Device
		return nil
	})
	return device, err
}

func (s *Service) inAdminTx(ctx context.Context, fn func(*db.Queries) error) error {
	return s.inTx(ctx, func(tx pgx.Tx, q *db.Queries) error {
		if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
			return fmt.Errorf("setting platform admin context: %w", err)
		}
		return fn(q)
	})
}

func (s *Service) inTenantTx(ctx context.Context, organisationID, locationID uuid.UUID, fn func(*db.Queries) error) error {
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
		return fn(q)
	})
}

func normalizeTenantActor(actor TenantActor) TenantActor {
	actor.ActorRef = strings.TrimSpace(actor.ActorRef)
	return actor
}

func validateTenantActor(actor TenantActor, requireLocation bool) error {
	if actor.OrganisationID == uuid.Nil || (requireLocation && actor.LocationID == uuid.Nil) {
		return ErrValidation
	}
	if actor.ActorRef == "" {
		return ErrForbidden
	}
	return nil
}

func requireOrganisationPermission(ctx context.Context, q *db.Queries, actor TenantActor, permission string) error {
	allowed, err := q.ActorHasOrganisationPermission(ctx, db.ActorHasOrganisationPermissionParams{
		OrganisationID: actor.OrganisationID,
		MemberRef:      actor.ActorRef,
		PermissionName: permission,
	})
	if err != nil {
		return fmt.Errorf("checking organisation permission: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func requireLocationPermission(ctx context.Context, q *db.Queries, actor TenantActor, permission string) error {
	if actor.LocationID == uuid.Nil {
		return ErrForbidden
	}
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

func (s *Service) inTx(ctx context.Context, fn func(pgx.Tx, *db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	queries := db.New(tx)
	if err := fn(tx, queries); err != nil {
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

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func nullableUUID(value uuid.UUID) uuid.NullUUID {
	if value == uuid.Nil {
		return uuid.NullUUID{}
	}
	return uuid.NullUUID{UUID: value, Valid: true}
}

func nullableTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func nullableUUIDValue(value uuid.NullUUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	return value.UUID
}

func normalizeRegisterDeviceParams(arg RegisterDeviceParams) RegisterDeviceParams {
	arg.DeviceType = strings.TrimSpace(arg.DeviceType)
	arg.Platform = strings.TrimSpace(arg.Platform)
	arg.AppVersion = strings.TrimSpace(arg.AppVersion)
	arg.Name = strings.TrimSpace(arg.Name)
	arg.Capabilities = defaultJSONObject(arg.Capabilities)
	arg.Configuration = defaultJSONObject(arg.Configuration)
	if arg.HeartbeatEvery <= 0 {
		arg.HeartbeatEvery = time.Minute
	}
	return arg
}

func validateRegisterDeviceParams(arg RegisterDeviceParams) error {
	if arg.OrganisationID == uuid.Nil || !validDeviceType(arg.DeviceType) || arg.Platform == "" || arg.AppVersion == "" {
		return ErrValidation
	}
	if !jsonObject(arg.Capabilities) || !jsonObject(arg.Configuration) {
		return ErrValidation
	}
	if arg.HeartbeatEvery < time.Second || arg.HeartbeatEvery > time.Hour {
		return ErrValidation
	}
	return nil
}

func validDeviceType(value string) bool {
	switch value {
	case DeviceTypeWaiterMobile, DeviceTypeManagerTablet, DeviceTypeDisplay, DeviceTypeHostDevice:
		return true
	default:
		return false
	}
}

func defaultJSONObject(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

func jsonObject(value json.RawMessage) bool {
	var decoded map[string]any
	return json.Unmarshal(value, &decoded) == nil
}

func normalizeOnboardOwnerParams(arg OnboardOwnerParams) (OnboardOwnerParams, error) {
	arg.OrganisationName = strings.TrimSpace(arg.OrganisationName)
	arg.OrganisationSlug = strings.TrimSpace(arg.OrganisationSlug)
	arg.LocationName = strings.TrimSpace(arg.LocationName)
	arg.LocationSlug = strings.TrimSpace(arg.LocationSlug)
	arg.Timezone = strings.TrimSpace(arg.Timezone)
	if arg.Timezone == "" {
		arg.Timezone = "UTC"
	}
	arg.Floor.Name = strings.TrimSpace(arg.Floor.Name)
	if arg.Floor.Name == "" {
		arg.Floor.Name = "Main floor"
	}
	arg.Floor.Slug = strings.TrimSpace(arg.Floor.Slug)
	if arg.Floor.Slug == "" {
		arg.Floor.Slug = "main-floor"
	}
	arg.Floor.Canvas = defaultJSONObject(arg.Floor.Canvas)
	if len(arg.Floor.Zones) == 0 {
		arg.Floor.Zones = []OnboardingZone{{Name: "Dining room"}}
	}
	for i := range arg.Floor.Zones {
		arg.Floor.Zones[i].Name = strings.TrimSpace(arg.Floor.Zones[i].Name)
	}
	for i := range arg.Floor.Tables {
		arg.Floor.Tables[i].Label = strings.TrimSpace(arg.Floor.Tables[i].Label)
		arg.Floor.Tables[i].CapacityLabel = strings.TrimSpace(arg.Floor.Tables[i].CapacityLabel)
		arg.Floor.Tables[i].Shape = strings.TrimSpace(arg.Floor.Tables[i].Shape)
		arg.Floor.Tables[i].ZoneName = strings.TrimSpace(arg.Floor.Tables[i].ZoneName)
		if arg.Floor.Tables[i].Shape == "" {
			arg.Floor.Tables[i].Shape = "rectangle"
		}
		if arg.Floor.Tables[i].CapacityLabel == "" {
			arg.Floor.Tables[i].CapacityLabel = "2-4"
		}
	}
	for i := range arg.ServicePeriods {
		arg.ServicePeriods[i].Name = strings.TrimSpace(arg.ServicePeriods[i].Name)
		arg.ServicePeriods[i].StartTime = strings.TrimSpace(arg.ServicePeriods[i].StartTime)
		arg.ServicePeriods[i].EndTime = strings.TrimSpace(arg.ServicePeriods[i].EndTime)
	}
	for i := range arg.Staff {
		arg.Staff[i].Email = strings.TrimSpace(arg.Staff[i].Email)
		arg.Staff[i].Name = strings.TrimSpace(arg.Staff[i].Name)
		arg.Staff[i].Role = strings.TrimSpace(arg.Staff[i].Role)
	}
	return arg, nil
}

func validateOnboardOwnerParams(arg OnboardOwnerParams) error {
	if arg.UserProfileID == uuid.Nil ||
		arg.OrganisationName == "" ||
		arg.LocationName == "" ||
		!validSlug(arg.OrganisationSlug) ||
		!validSlug(arg.LocationSlug) ||
		!validSlug(arg.Floor.Slug) ||
		!jsonObject(arg.Floor.Canvas) {
		return ErrValidation
	}
	if _, err := time.LoadLocation(arg.Timezone); err != nil {
		return ErrValidation
	}
	zoneNames := map[string]bool{}
	for _, zone := range arg.Floor.Zones {
		if zone.Name == "" || zoneNames[zone.Name] {
			return ErrValidation
		}
		zoneNames[zone.Name] = true
	}
	for _, table := range arg.Floor.Tables {
		if table.Label == "" ||
			table.CapacityLabel == "" ||
			!validTableShape(table.Shape) ||
			!jsonObject(table.Geometry) {
			return ErrValidation
		}
		if table.ZoneName != "" && !zoneNames[table.ZoneName] {
			return ErrValidation
		}
	}
	for _, period := range arg.ServicePeriods {
		if _, _, _, _, err := parseOnboardingServicePeriod(period); err != nil {
			return err
		}
	}
	for _, staff := range arg.Staff {
		if staff.Email == "" && staff.Name == "" {
			return ErrValidation
		}
		if staff.Role != "" && !validStaffRole(staff.Role) {
			return ErrValidation
		}
	}
	return nil
}

func createOnboardingLayout(
	ctx context.Context,
	q *db.Queries,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	floorInput OnboardingFloor,
) (*db.Floor, []db.Zone, []db.Table, error) {
	floor, err := q.CreateFloor(ctx, db.CreateFloorParams{
		OrganisationID: organisationID,
		LocationID:     locationID,
		Slug:           floorInput.Slug,
		Name:           floorInput.Name,
		SortOrder:      floorInput.SortOrder,
		Canvas:         floorInput.Canvas,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating onboarding floor: %w", err)
	}
	zones := make([]db.Zone, 0, len(floorInput.Zones))
	zoneByName := map[string]db.Zone{}
	for _, zoneInput := range floorInput.Zones {
		zone, err := q.CreateZone(ctx, db.CreateZoneParams{
			OrganisationID: organisationID,
			LocationID:     locationID,
			FloorID:        floor.ID,
			Name:           zoneInput.Name,
			SortOrder:      zoneInput.SortOrder,
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("creating onboarding zone: %w", err)
		}
		zones = append(zones, zone)
		zoneByName[zone.Name] = zone
	}
	tables := make([]db.Table, 0, len(floorInput.Tables))
	defaultZone := zones[0]
	for _, tableInput := range floorInput.Tables {
		zone := defaultZone
		if tableInput.ZoneName != "" {
			zone = zoneByName[tableInput.ZoneName]
		}
		table, err := q.CreateTable(ctx, db.CreateTableParams{
			OrganisationID: organisationID,
			LocationID:     locationID,
			FloorID:        floor.ID,
			ZoneID:         zone.ID,
			Label:          tableInput.Label,
			CapacityLabel:  tableInput.CapacityLabel,
			Shape:          tableInput.Shape,
			Geometry:       tableInput.Geometry,
		})
		if err != nil {
			return nil, nil, nil, fmt.Errorf("creating onboarding table: %w", err)
		}
		if _, err := q.CreateTableOccupancy(ctx, db.CreateTableOccupancyParams{
			TableID:        table.ID,
			OrganisationID: organisationID,
			LocationID:     locationID,
		}); err != nil {
			return nil, nil, nil, fmt.Errorf("creating onboarding table occupancy: %w", err)
		}
		tables = append(tables, table)
	}
	return &floor, zones, tables, nil
}

func createOnboardingServicePeriods(
	ctx context.Context,
	q *db.Queries,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	inputs []OnboardingServicePeriod,
) ([]db.ServicePeriod, error) {
	periods := make([]db.ServicePeriod, 0, len(inputs))
	for _, input := range inputs {
		name, days, start, end, err := parseOnboardingServicePeriod(input)
		if err != nil {
			return nil, err
		}
		period, err := q.CreateServicePeriod(ctx, db.CreateServicePeriodParams{
			OrganisationID: organisationID,
			LocationID:     locationID,
			Name:           name,
			DaysOfWeek:     days,
			StartTime:      start,
			EndTime:        end,
		})
		if err != nil {
			return nil, fmt.Errorf("creating onboarding service period: %w", err)
		}
		periods = append(periods, period)
	}
	return periods, nil
}

func parseOnboardingServicePeriod(input OnboardingServicePeriod) (string, []int16, pgtype.Time, pgtype.Time, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(input.DaysOfWeek) == 0 || len(input.DaysOfWeek) > 7 {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	seen := map[int16]bool{}
	days := make([]int16, 0, len(input.DaysOfWeek))
	for _, day := range input.DaysOfWeek {
		if day < 0 || day > 6 || seen[day] {
			return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
		}
		seen[day] = true
		days = append(days, day)
	}
	start, err := parseOnboardingClock(input.StartTime)
	if err != nil {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	end, err := parseOnboardingClock(input.EndTime)
	if err != nil {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	return name, days, start, end, nil
}

func parseOnboardingClock(value string) (pgtype.Time, error) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		parsed, err = time.Parse("15:04:05", strings.TrimSpace(value))
	}
	if err != nil {
		return pgtype.Time{}, err
	}
	micros := int64(parsed.Hour()) * int64(time.Hour/time.Microsecond)
	micros += int64(parsed.Minute()) * int64(time.Minute/time.Microsecond)
	micros += int64(parsed.Second()) * int64(time.Second/time.Microsecond)
	return pgtype.Time{Microseconds: micros, Valid: true}, nil
}

func validSlug(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func validTableShape(value string) bool {
	switch value {
	case "rectangle", "circle", "square", "custom":
		return true
	default:
		return false
	}
}

func validStaffRole(value string) bool {
	switch value {
	case RoleLocationManager, RoleWaiter, RoleReadOnly:
		return true
	default:
		return false
	}
}

func userProfileActorRef(id uuid.UUID) string {
	if id == uuid.Nil {
		return "user:"
	}
	return "user:" + id.String()
}

func mapIdentityConstraintError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrAlreadyExists
	}
	return err
}
