-- name: SetPlatformAdminContext :exec
SELECT set_config('seatd.platform_admin', 'true', true);

-- name: SetTenantContext :exec
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', COALESCE($2::text, ''), true);

-- name: ClearTenantContext :exec
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', '', true),
       set_config('seatd.current_location_id', '', true);

-- name: CreateUserProfile :one
INSERT INTO user_profiles (display_name, email)
VALUES ($1, $2)
RETURNING *;

-- name: GetActiveUserProfileByEmail :one
SELECT *
FROM user_profiles
WHERE lower(btrim(email)) = $1
  AND status = 'active';

-- name: LinkExternalIdentity :one
INSERT INTO external_identities (user_profile_id, issuer, subject, email)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByExternalIdentity :one
SELECT sqlc.embed(user_profiles), sqlc.embed(external_identities)
FROM external_identities
JOIN user_profiles ON user_profiles.id = external_identities.user_profile_id
WHERE external_identities.issuer = $1
  AND external_identities.subject = $2
  AND user_profiles.status = 'active';

-- name: CreateWebSession :one
INSERT INTO web_sessions (
    user_profile_id,
    lookup_prefix,
    session_hash,
    expires_at,
    user_agent,
    ip_address,
    rotated_from_session_id
)
VALUES ($1, $2, $3, $4, $5, sqlc.narg(ip_address)::inet, sqlc.narg(rotated_from_session_id)::uuid)
RETURNING *;

-- name: GetWebSessionByPrefix :one
SELECT sqlc.embed(web_sessions), sqlc.embed(user_profiles)
FROM web_sessions
JOIN user_profiles ON user_profiles.id = web_sessions.user_profile_id
WHERE web_sessions.lookup_prefix = $1
  AND web_sessions.revoked_at IS NULL
  AND web_sessions.expires_at > now()
  AND user_profiles.status = 'active';

-- name: TouchWebSession :exec
UPDATE web_sessions
SET last_used_at = now(),
    updated_at = now()
WHERE id = $1
  AND revoked_at IS NULL
  AND expires_at > now();

-- name: RevokeWebSession :exec
UPDATE web_sessions
SET revoked_at = COALESCE(revoked_at, now()),
    updated_at = now()
WHERE id = $1;

-- name: ListUserOrganisationMemberships :many
SELECT *
FROM organisation_memberships
WHERE user_profile_id = $1
  AND disabled_at IS NULL
ORDER BY role, member_ref;

-- name: ListUserLocationMemberships :many
SELECT *
FROM location_memberships
WHERE user_profile_id = $1
  AND disabled_at IS NULL
ORDER BY role, member_ref;

-- name: GetRolePermissions :many
SELECT permission_name
FROM role_permissions
WHERE role_name = $1
ORDER BY permission_name;

-- name: UserHasOrganisationPermission :one
SELECT EXISTS (
    SELECT 1
    FROM organisation_memberships om
    JOIN role_permissions rp ON rp.role_name = om.role
    WHERE om.organisation_id = $1
      AND om.user_profile_id = $2
      AND om.disabled_at IS NULL
      AND rp.permission_name = $3
) AS has_permission;

-- name: UserHasLocationPermission :one
SELECT EXISTS (
    SELECT 1
    FROM location_memberships lm
    JOIN role_permissions rp ON rp.role_name = lm.role
    WHERE lm.organisation_id = $1
      AND lm.location_id = $2
      AND lm.user_profile_id = $3
      AND lm.disabled_at IS NULL
      AND rp.permission_name = $4
    UNION ALL
    SELECT 1
    FROM organisation_memberships om
    JOIN role_permissions rp ON rp.role_name = om.role
    WHERE om.organisation_id = $1
      AND om.user_profile_id = $3
      AND om.disabled_at IS NULL
      AND rp.permission_name = $4
) AS has_permission;

-- name: ActorHasOrganisationPermission :one
SELECT EXISTS (
    SELECT 1
    FROM organisation_memberships om
    JOIN role_permissions rp ON rp.role_name = om.role
    WHERE om.organisation_id = $1
      AND om.member_ref = $2
      AND om.disabled_at IS NULL
      AND rp.permission_name = $3
) AS has_permission;

-- name: ActorHasLocationPermission :one
SELECT EXISTS (
    SELECT 1
    FROM location_memberships lm
    JOIN role_permissions rp ON rp.role_name = lm.role
    WHERE lm.organisation_id = $1
      AND lm.location_id = $2
      AND lm.member_ref = $3
      AND lm.disabled_at IS NULL
      AND rp.permission_name = $4
    UNION ALL
    SELECT 1
    FROM organisation_memberships om
    JOIN role_permissions rp ON rp.role_name = om.role
    WHERE om.organisation_id = $1
      AND om.member_ref = $3
      AND om.disabled_at IS NULL
      AND rp.permission_name = $4
) AS has_permission;

-- name: ActorHasPlatformPermission :one
SELECT EXISTS (
    SELECT 1
    FROM organisation_memberships om
    JOIN role_permissions rp ON rp.role_name = om.role
    WHERE om.member_ref = $1
      AND om.disabled_at IS NULL
      AND rp.permission_name = $2
) AS has_permission;

-- name: CreateUserOrganisationMembership :one
INSERT INTO organisation_memberships (organisation_id, user_profile_id, member_ref, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateUserLocationMembership :one
INSERT INTO location_memberships (organisation_id, location_id, user_profile_id, member_ref, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: RegisterDevice :one
INSERT INTO devices (
    organisation_id,
    location_id,
    user_profile_id,
    device_type,
    platform,
    app_version,
    name,
    capabilities,
    configuration,
    assigned_at,
    heartbeat_interval_seconds
)
VALUES (
    sqlc.arg(organisation_id)::uuid,
    sqlc.narg(location_id)::uuid,
    sqlc.narg(user_profile_id)::uuid,
    sqlc.arg(device_type)::text,
    sqlc.arg(platform)::text,
    sqlc.arg(app_version)::text,
    sqlc.narg(name)::text,
    sqlc.arg(capabilities)::jsonb,
    sqlc.arg(configuration)::jsonb,
    CASE WHEN sqlc.narg(location_id)::uuid IS NULL THEN NULL ELSE now() END,
    sqlc.arg(heartbeat_interval_seconds)::integer
)
RETURNING *;

-- name: TrustDevice :one
UPDATE devices
SET trust_state = 'trusted',
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND revoked_at IS NULL
RETURNING *;

-- name: MarkDeviceSeen :one
UPDATE devices
SET last_seen_at = now(),
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND trust_state = 'trusted'
  AND revoked_at IS NULL
RETURNING *;

-- name: RecordDeviceHeartbeat :one
UPDATE devices
SET last_seen_at = now(),
    last_heartbeat_at = now(),
    app_version = $3,
    capabilities = $4,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND trust_state = 'trusted'
  AND revoked_at IS NULL
RETURNING *;

-- name: RevokeDevice :one
UPDATE devices
SET trust_state = 'revoked',
    revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND revoked_at IS NULL
RETURNING *;

-- name: CreateDeviceCredential :one
INSERT INTO device_credentials (organisation_id, device_id, lookup_prefix, credential_hash, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDeviceCredentialByPrefix :one
SELECT sqlc.embed(device_credentials), sqlc.embed(devices)
FROM device_credentials
JOIN devices ON devices.id = device_credentials.device_id
WHERE device_credentials.lookup_prefix = $1
  AND device_credentials.revoked_at IS NULL
  AND (device_credentials.expires_at IS NULL OR device_credentials.expires_at > now())
  AND devices.trust_state = 'trusted'
  AND devices.revoked_at IS NULL;

-- name: TouchDeviceCredential :exec
UPDATE device_credentials
SET last_used_at = now(),
    updated_at = now()
WHERE id = $1
  AND revoked_at IS NULL;

-- name: RevokeDeviceCredentials :exec
UPDATE device_credentials
SET revoked_at = now(),
    updated_at = now()
WHERE organisation_id = $1
  AND device_id = $2
  AND revoked_at IS NULL;

-- name: CreateTableQRCapabilityHashed :one
INSERT INTO table_qr_capabilities (
    organisation_id,
    location_id,
    table_id,
    token,
    token_lookup_prefix,
    token_hash,
    label,
    expires_at
)
VALUES (
    sqlc.arg(organisation_id)::uuid,
    sqlc.arg(location_id)::uuid,
    sqlc.arg(table_id)::uuid,
    sqlc.arg(token_lookup_prefix)::text,
    sqlc.arg(token_lookup_prefix)::text,
    sqlc.arg(token_hash)::bytea,
    sqlc.arg(label)::text,
    sqlc.arg(expires_at)::timestamptz
)
RETURNING *;

-- name: GetActiveTableQRCapabilityByHash :one
SELECT table_qr_capabilities.*
FROM table_qr_capabilities
JOIN locations ON locations.id = table_qr_capabilities.location_id
    AND locations.organisation_id = table_qr_capabilities.organisation_id
JOIN tables ON tables.id = table_qr_capabilities.table_id
    AND tables.organisation_id = table_qr_capabilities.organisation_id
    AND tables.location_id = table_qr_capabilities.location_id
WHERE table_qr_capabilities.token_lookup_prefix = $1
  AND table_qr_capabilities.token_hash = $2
  AND table_qr_capabilities.revoked_at IS NULL
  AND (table_qr_capabilities.expires_at IS NULL OR table_qr_capabilities.expires_at > now())
  AND locations.status = 'active'
  AND tables.is_active
  AND tables.deleted_at IS NULL;

-- name: ListActiveTableQRCapabilitiesByPrefix :many
SELECT table_qr_capabilities.*
FROM table_qr_capabilities
JOIN locations ON locations.id = table_qr_capabilities.location_id
    AND locations.organisation_id = table_qr_capabilities.organisation_id
JOIN tables ON tables.id = table_qr_capabilities.table_id
    AND tables.organisation_id = table_qr_capabilities.organisation_id
    AND tables.location_id = table_qr_capabilities.location_id
WHERE table_qr_capabilities.token_lookup_prefix = $1
  AND table_qr_capabilities.revoked_at IS NULL
  AND (table_qr_capabilities.expires_at IS NULL OR table_qr_capabilities.expires_at > now())
  AND locations.status = 'active'
  AND tables.is_active
  AND tables.deleted_at IS NULL
ORDER BY table_qr_capabilities.issued_at DESC;

-- name: TouchTableQRCapability :exec
UPDATE table_qr_capabilities
SET last_used_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: ListTableQRCapabilitiesByTable :many
SELECT *
FROM table_qr_capabilities
WHERE organisation_id = $1
  AND location_id = $2
  AND table_id = $3
ORDER BY issued_at DESC;

-- name: RotateTableQRCapability :one
UPDATE table_qr_capabilities
SET revoked_at = COALESCE(revoked_at, now()),
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND table_id = $4
RETURNING *;

-- name: RecordGuestAbuseEvent :exec
INSERT INTO guest_abuse_events (
    organisation_id,
    location_id,
    table_id,
    token_lookup_prefix,
    action_key,
    ip_hash,
    decision
)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: RevokeTableQRCapability :one
UPDATE table_qr_capabilities
SET revoked_at = now(),
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND revoked_at IS NULL
RETURNING *;

-- name: RecordAuditEvent :one
INSERT INTO audit_events (organisation_id, location_id, actor_ref, action, target_type, target_id, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListDevicesByOrganisation :many
SELECT *
FROM devices
WHERE organisation_id = $1
ORDER BY registered_at DESC;

-- name: ListDevicesByLocation :many
SELECT *
FROM devices
WHERE organisation_id = $1
  AND location_id = $2
ORDER BY registered_at DESC;

-- name: GetDevice :one
SELECT *
FROM devices
WHERE id = $1
  AND organisation_id = $2;

-- name: CreateDevicePairingCode :one
INSERT INTO device_pairing_codes (
    organisation_id,
    location_id,
    created_by,
    device_type,
    pairing_code_lookup_prefix,
    pairing_code_hash,
    expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetDevicePairingCodeByPrefix :one
SELECT device_pairing_codes.*
FROM device_pairing_codes
JOIN locations ON locations.id = device_pairing_codes.location_id
    AND locations.organisation_id = device_pairing_codes.organisation_id
WHERE device_pairing_codes.pairing_code_lookup_prefix = $1
  AND device_pairing_codes.consumed_at IS NULL
  AND device_pairing_codes.expires_at > now()
  AND locations.status = 'active';

-- name: ConsumeDevicePairingCode :one
UPDATE device_pairing_codes
SET consumed_at = now(),
    consumed_by_device_id = $2,
    updated_at = now()
WHERE id = $1
  AND consumed_at IS NULL
  AND expires_at > now()
RETURNING *;

-- name: ListOrganisationMemberships :many
SELECT *
FROM organisation_memberships
WHERE organisation_id = $1
  AND disabled_at IS NULL
ORDER BY role, member_ref;

-- name: ListLocationMemberships :many
SELECT *
FROM location_memberships
WHERE organisation_id = $1
  AND location_id = $2
  AND disabled_at IS NULL
ORDER BY role, member_ref;
