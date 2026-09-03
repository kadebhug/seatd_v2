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

	PermissionPlatformAdmin      = "platform.admin"
	PermissionOrganisationManage = "organisation.manage"
	PermissionLocationManage     = "location.manage"
	PermissionLayoutRead         = "layout.read"
	PermissionLayoutWrite        = "layout.write"
	PermissionOperationsRead     = "operations.read"
	PermissionOperationsWrite    = "operations.write"
	PermissionDeviceManage       = "device.manage"
	PermissionAuditRead          = "audit.read"
	PermissionAnalyticsRead      = "analytics.read"
	PermissionIntegrationsRead   = "integrations.read"
	PermissionIntegrationsManage = "integrations.manage"

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

type ResolveExternalIdentityParams struct {
	DisplayName string
	Email       string
	External    ExternalIdentity
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
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("getting external identity: %w", err)
		}

		created, err := q.CreateUserProfile(ctx, db.CreateUserProfileParams{
			DisplayName: arg.DisplayName,
			Email:       nullableText(arg.Email),
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
		return nil
	})
	return result, err
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
		result.Memberships = make([]Membership, 0, len(orgs)+len(locations))
		for _, membership := range orgs {
			result.Memberships = append(result.Memberships, Membership{
				OrganisationID: membership.OrganisationID,
				MemberRef:      membership.MemberRef,
				Role:           membership.Role,
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
