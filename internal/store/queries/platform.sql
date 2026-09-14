-- name: SearchPlatformTenants :many
WITH tenant_locations AS (
    SELECT
        organisation_id,
        count(*) AS location_count,
        count(*) FILTER (WHERE status = 'active') AS active_location_count,
        count(*) FILTER (WHERE status = 'disabled') AS disabled_location_count
    FROM locations
    GROUP BY organisation_id
),
tenant_devices AS (
    SELECT
        organisation_id,
        count(*) AS device_count,
        count(*) FILTER (WHERE trust_state = 'trusted') AS trusted_device_count,
        max(last_seen_at)::timestamptz AS last_device_seen_at
    FROM devices
    GROUP BY organisation_id
),
tenant_activity AS (
    SELECT organisation_id, max(created_at) AS last_audit_at
    FROM audit_events
    WHERE organisation_id IS NOT NULL
    GROUP BY organisation_id
)
SELECT
    organisations.id,
    organisations.slug,
    organisations.name,
    organisations.status,
    organisations.created_at,
    organisations.updated_at,
    COALESCE(tenant_locations.location_count, 0)::integer AS location_count,
    COALESCE(tenant_locations.active_location_count, 0)::integer AS active_location_count,
    COALESCE(tenant_locations.disabled_location_count, 0)::integer AS disabled_location_count,
    COALESCE(tenant_devices.device_count, 0)::integer AS device_count,
    COALESCE(tenant_devices.trusted_device_count, 0)::integer AS trusted_device_count,
    tenant_devices.last_device_seen_at::timestamptz AS last_device_seen_at,
    tenant_activity.last_audit_at::timestamptz AS last_audit_at
FROM organisations
LEFT JOIN tenant_locations ON tenant_locations.organisation_id = organisations.id
LEFT JOIN tenant_devices ON tenant_devices.organisation_id = organisations.id
LEFT JOIN tenant_activity ON tenant_activity.organisation_id = organisations.id
WHERE
    sqlc.arg(search)::text = ''
    OR organisations.slug ILIKE '%' || sqlc.arg(search)::text || '%'
    OR organisations.name ILIKE '%' || sqlc.arg(search)::text || '%'
ORDER BY organisations.updated_at DESC, organisations.name, organisations.slug
LIMIT sqlc.arg(result_limit)::integer;

-- name: GetPlatformTenantOverview :one
WITH tenant_locations AS (
    SELECT
        organisation_id,
        count(*) AS location_count,
        count(*) FILTER (WHERE status = 'active') AS active_location_count,
        count(*) FILTER (WHERE status = 'disabled') AS disabled_location_count
    FROM locations
    WHERE organisation_id = $1
    GROUP BY organisation_id
),
tenant_devices AS (
    SELECT
        organisation_id,
        count(*) AS device_count,
        count(*) FILTER (WHERE trust_state = 'trusted') AS trusted_device_count,
        count(*) FILTER (WHERE trust_state = 'pending') AS pending_device_count,
        count(*) FILTER (WHERE trust_state = 'revoked') AS revoked_device_count,
        max(last_seen_at)::timestamptz AS last_device_seen_at
    FROM devices
    WHERE organisation_id = $1
    GROUP BY organisation_id
),
tenant_activity AS (
    SELECT organisation_id, max(created_at) AS last_audit_at
    FROM audit_events
    WHERE organisation_id = $1
    GROUP BY organisation_id
)
SELECT
    organisations.id,
    organisations.slug,
    organisations.name,
    organisations.status,
    organisations.created_at,
    organisations.updated_at,
    COALESCE(tenant_locations.location_count, 0)::integer AS location_count,
    COALESCE(tenant_locations.active_location_count, 0)::integer AS active_location_count,
    COALESCE(tenant_locations.disabled_location_count, 0)::integer AS disabled_location_count,
    COALESCE(tenant_devices.device_count, 0)::integer AS device_count,
    COALESCE(tenant_devices.trusted_device_count, 0)::integer AS trusted_device_count,
    COALESCE(tenant_devices.pending_device_count, 0)::integer AS pending_device_count,
    COALESCE(tenant_devices.revoked_device_count, 0)::integer AS revoked_device_count,
    tenant_devices.last_device_seen_at::timestamptz AS last_device_seen_at,
    tenant_activity.last_audit_at::timestamptz AS last_audit_at
FROM organisations
LEFT JOIN tenant_locations ON tenant_locations.organisation_id = organisations.id
LEFT JOIN tenant_devices ON tenant_devices.organisation_id = organisations.id
LEFT JOIN tenant_activity ON tenant_activity.organisation_id = organisations.id
WHERE organisations.id = $1;

-- name: ListPlatformTenantLocations :many
SELECT id, organisation_id, slug, name, timezone, status, created_at, updated_at
FROM locations
WHERE organisation_id = $1
ORDER BY status, name, slug;

-- name: ListPlatformTenantMembershipRoleCounts :many
SELECT role, count(*)::integer AS member_count
FROM (
    SELECT organisation_memberships.role
    FROM organisation_memberships
    WHERE organisation_memberships.organisation_id = $1
      AND organisation_memberships.disabled_at IS NULL
    UNION ALL
    SELECT location_memberships.role
    FROM location_memberships
    WHERE location_memberships.organisation_id = $1
      AND location_memberships.disabled_at IS NULL
) memberships
GROUP BY role
ORDER BY role;

-- name: ListPlatformTenantDeviceCounts :many
SELECT
    trust_state,
    device_type,
    count(*)::integer AS device_count,
    max(last_seen_at)::timestamptz AS last_seen_at
FROM devices
WHERE organisation_id = $1
GROUP BY trust_state, device_type
ORDER BY trust_state, device_type;

-- name: GetPlatformTenantDiagnostics :one
WITH location_counts AS (
    SELECT
        count(*) FILTER (WHERE status = 'disabled') AS disabled_location_count
    FROM locations
    WHERE locations.organisation_id = $1
),
device_counts AS (
    SELECT
        count(*) FILTER (WHERE trust_state = 'trusted' AND last_heartbeat_at IS NULL) AS never_heartbeat_device_count,
        count(*) FILTER (
            WHERE trust_state = 'trusted'
              AND last_heartbeat_at IS NOT NULL
              AND last_heartbeat_at < now() - make_interval(secs => greatest(heartbeat_interval_seconds * 3, 180))
        ) AS stale_device_count,
        count(*) FILTER (WHERE trust_state = 'pending') AS pending_device_count,
        count(*) FILTER (WHERE trust_state = 'revoked') AS revoked_device_count
    FROM devices
    WHERE organisation_id = $1
),
integration_counts AS (
    SELECT
        count(*) AS integration_count,
        count(*) FILTER (WHERE status = 'connected') AS connected_integration_count,
        count(*) FILTER (WHERE status = 'degraded') AS degraded_integration_count,
        count(*) FILTER (WHERE status = 'disconnected') AS disconnected_integration_count,
        max(last_successful_sync_at)::timestamptz AS last_successful_sync_at
    FROM integration_connections
    WHERE organisation_id = $1
),
audit_counts AS (
    SELECT count(*) AS recent_platform_audit_count
    FROM audit_events
    WHERE organisation_id = $1
      AND action LIKE 'platform.%'
      AND created_at >= now() - interval '7 days'
),
checkpoint AS (
    SELECT lag_seconds, rebuild_status, updated_at
    FROM analytics_projector_checkpoints
    ORDER BY updated_at DESC
    LIMIT 1
)
SELECT
    COALESCE(location_counts.disabled_location_count, 0)::integer AS disabled_location_count,
    COALESCE(device_counts.never_heartbeat_device_count, 0)::integer AS never_heartbeat_device_count,
    COALESCE(device_counts.stale_device_count, 0)::integer AS stale_device_count,
    COALESCE(device_counts.pending_device_count, 0)::integer AS pending_device_count,
    COALESCE(device_counts.revoked_device_count, 0)::integer AS revoked_device_count,
    COALESCE(integration_counts.integration_count, 0)::integer AS integration_count,
    COALESCE(integration_counts.connected_integration_count, 0)::integer AS connected_integration_count,
    COALESCE(integration_counts.degraded_integration_count, 0)::integer AS degraded_integration_count,
    COALESCE(integration_counts.disconnected_integration_count, 0)::integer AS disconnected_integration_count,
    integration_counts.last_successful_sync_at::timestamptz AS last_successful_sync_at,
    COALESCE(audit_counts.recent_platform_audit_count, 0)::integer AS recent_platform_audit_count,
    checkpoint.lag_seconds AS analytics_lag_seconds,
    checkpoint.rebuild_status AS analytics_rebuild_status,
    checkpoint.updated_at AS analytics_checkpoint_updated_at
FROM location_counts, device_counts, integration_counts, audit_counts
LEFT JOIN checkpoint ON true;

-- name: SetOrganisationStatus :one
UPDATE organisations
SET status = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, slug, name, status, legacy_restaurant_id, created_at, updated_at;

-- name: ListPlatformMemberships :many
SELECT
    pm.id,
    pm.user_profile_id,
    up.display_name,
    up.email,
    pm.role,
    pm.granted_by_actor_ref,
    pm.granted_at,
    pm.disabled_at,
    pm.disabled_by_actor_ref,
    pm.created_at,
    pm.updated_at
FROM platform_memberships pm
JOIN user_profiles up ON up.id = pm.user_profile_id
ORDER BY pm.disabled_at NULLS FIRST, pm.role, up.display_name, up.email;

-- name: ListPlatformAdminGrants :many
SELECT *
FROM platform_admin_grants
ORDER BY revoked_at NULLS FIRST, consumed_at NULLS FIRST, created_at DESC;

-- name: CreatePlatformAdminGrant :one
INSERT INTO platform_admin_grants (email, role, invited_by_actor_ref)
VALUES (lower(btrim($1)), $2, $3)
ON CONFLICT (lower(btrim(email))) WHERE consumed_at IS NULL AND revoked_at IS NULL
DO UPDATE SET
    role = EXCLUDED.role,
    invited_by_actor_ref = EXCLUDED.invited_by_actor_ref
RETURNING *;

-- name: RevokePlatformAdminGrant :one
UPDATE platform_admin_grants
SET revoked_at = now(),
    revoked_by_actor_ref = $2
WHERE id = $1
  AND consumed_at IS NULL
  AND revoked_at IS NULL
RETURNING *;

-- name: ConsumePlatformAdminGrant :one
UPDATE platform_admin_grants
SET consumed_at = now(),
    consumed_by_user_profile_id = $1
WHERE lower(btrim(email)) = lower(btrim($2))
  AND consumed_at IS NULL
  AND revoked_at IS NULL
RETURNING role, invited_by_actor_ref;

-- name: CreatePlatformMembership :one
INSERT INTO platform_memberships (user_profile_id, role, granted_by_actor_ref)
VALUES ($1, $2, $3)
ON CONFLICT (user_profile_id) WHERE disabled_at IS NULL DO UPDATE
SET role = EXCLUDED.role,
    updated_at = now()
RETURNING *;

-- name: DisablePlatformMembership :one
UPDATE platform_memberships
SET disabled_at = now(),
    disabled_by_actor_ref = $2,
    updated_at = now()
WHERE id = $1
  AND disabled_at IS NULL
RETURNING *;

-- name: ReactivatePlatformMembership :one
UPDATE platform_memberships
SET disabled_at = NULL,
    disabled_by_actor_ref = NULL,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountActivePlatformAdmins :one
SELECT count(*)::integer
FROM platform_memberships
WHERE role = 'platform_admin'
  AND disabled_at IS NULL;

-- name: GetPlatformMembership :one
SELECT *
FROM platform_memberships
WHERE id = $1;

-- name: SetOrganisationOnboarded :one
UPDATE organisations
SET onboarded_at = now(),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListPlatformTenantOwners :many
SELECT om.id, 'organisation' AS scope, om.organisation_id, NULL::uuid AS location_id, om.user_profile_id,
       up.display_name, up.email, om.member_ref, om.role, om.disabled_at
FROM organisation_memberships om
LEFT JOIN user_profiles up ON up.id = om.user_profile_id
WHERE om.organisation_id = $1
  AND om.role = 'organisation_owner'
  AND ($2::boolean OR om.disabled_at IS NULL)
ORDER BY om.disabled_at NULLS FIRST, up.display_name, om.member_ref;

-- name: ListPlatformTenantAudit :many
SELECT *
FROM audit_events
WHERE organisation_id = $1
ORDER BY created_at DESC
LIMIT $2;
