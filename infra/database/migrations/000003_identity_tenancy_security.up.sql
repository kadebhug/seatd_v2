CREATE TABLE IF NOT EXISTS user_profiles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name text NOT NULL,
    email text,
    status text NOT NULL DEFAULT 'active',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (display_name <> ''),
    CHECK (email IS NULL OR email <> ''),
    CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS external_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_profile_id uuid NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    issuer text NOT NULL,
    subject text NOT NULL,
    email text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject),
    CHECK (issuer <> ''),
    CHECK (subject <> ''),
    CHECK (email IS NULL OR email <> '')
);

CREATE TABLE IF NOT EXISTS roles (
    name text PRIMARY KEY,
    scope text NOT NULL,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (name <> ''),
    CHECK (scope IN ('platform', 'organisation', 'location')),
    CHECK (description <> '')
);

CREATE TABLE IF NOT EXISTS permissions (
    name text PRIMARY KEY,
    description text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (name <> ''),
    CHECK (description <> '')
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_name text NOT NULL REFERENCES roles (name) ON DELETE CASCADE,
    permission_name text NOT NULL REFERENCES permissions (name) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (role_name, permission_name)
);

ALTER TABLE organisation_memberships
    ADD COLUMN IF NOT EXISTS user_profile_id uuid REFERENCES user_profiles (id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS disabled_at timestamptz;

ALTER TABLE location_memberships
    ADD COLUMN IF NOT EXISTS user_profile_id uuid REFERENCES user_profiles (id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS disabled_at timestamptz;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'organisation_memberships_role_fk'
    ) THEN
        ALTER TABLE organisation_memberships
            ADD CONSTRAINT organisation_memberships_role_fk
            FOREIGN KEY (role) REFERENCES roles (name) ON DELETE RESTRICT;
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'location_memberships_role_fk'
    ) THEN
        ALTER TABLE location_memberships
            ADD CONSTRAINT location_memberships_role_fk
            FOREIGN KEY (role) REFERENCES roles (name) ON DELETE RESTRICT;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS organisation_memberships_active_user
    ON organisation_memberships (organisation_id, user_profile_id)
    WHERE user_profile_id IS NOT NULL AND disabled_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS location_memberships_active_user
    ON location_memberships (location_id, user_profile_id)
    WHERE user_profile_id IS NOT NULL AND disabled_at IS NULL;

CREATE TABLE IF NOT EXISTS devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    location_id uuid REFERENCES locations (id) ON DELETE SET NULL,
    user_profile_id uuid REFERENCES user_profiles (id) ON DELETE SET NULL,
    device_type text NOT NULL,
    platform text NOT NULL,
    app_version text NOT NULL,
    trust_state text NOT NULL DEFAULT 'pending',
    registered_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE RESTRICT,
    CHECK (device_type IN ('waiter_mobile', 'manager_tablet', 'display', 'host_device')),
    CHECK (platform <> ''),
    CHECK (app_version <> ''),
    CHECK (trust_state IN ('pending', 'trusted', 'revoked')),
    CHECK ((trust_state = 'revoked' AND revoked_at IS NOT NULL) OR (trust_state <> 'revoked' AND revoked_at IS NULL))
);

CREATE TABLE IF NOT EXISTS device_credentials (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    device_id uuid NOT NULL,
    lookup_prefix text NOT NULL,
    credential_hash bytea NOT NULL,
    issued_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    expires_at timestamptz,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE,
    UNIQUE (lookup_prefix),
    CHECK (lookup_prefix <> ''),
    CHECK (length(credential_hash) >= 32),
    CHECK (expires_at IS NULL OR expires_at > issued_at)
);

ALTER TABLE table_qr_capabilities
    ADD COLUMN IF NOT EXISTS token_lookup_prefix text,
    ADD COLUMN IF NOT EXISTS token_hash bytea,
    ADD COLUMN IF NOT EXISTS last_used_at timestamptz;

UPDATE table_qr_capabilities
SET token_lookup_prefix = COALESCE(token_lookup_prefix, left(token, 16)),
    token_hash = COALESCE(token_hash, digest(token, 'sha256'))
WHERE token_hash IS NULL OR token_lookup_prefix IS NULL;

UPDATE table_qr_capabilities
SET token = token_lookup_prefix;

ALTER TABLE table_qr_capabilities
    DROP CONSTRAINT IF EXISTS table_qr_capabilities_token_key;

ALTER TABLE table_qr_capabilities
    ALTER COLUMN token_lookup_prefix SET NOT NULL,
    ALTER COLUMN token_hash SET NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS table_qr_capabilities_token_hash_unique
    ON table_qr_capabilities (token_hash);

ALTER TABLE table_qr_capabilities
    DROP CONSTRAINT IF EXISTS table_qr_capabilities_token_hash_length;

ALTER TABLE table_qr_capabilities
    ADD CONSTRAINT table_qr_capabilities_token_hash_length
    CHECK (length(token_hash) >= 32);

CREATE TABLE IF NOT EXISTS audit_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid REFERENCES organisations (id) ON DELETE SET NULL,
    location_id uuid,
    actor_ref text NOT NULL,
    action text NOT NULL,
    target_type text NOT NULL,
    target_id text NOT NULL,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (actor_ref <> ''),
    CHECK (action <> ''),
    CHECK (target_type <> ''),
    CHECK (target_id <> ''),
    CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS external_identities_user_profile ON external_identities (user_profile_id);
CREATE INDEX IF NOT EXISTS organisation_memberships_user_profile ON organisation_memberships (user_profile_id) WHERE disabled_at IS NULL;
CREATE INDEX IF NOT EXISTS location_memberships_user_profile ON location_memberships (user_profile_id) WHERE disabled_at IS NULL;
CREATE INDEX IF NOT EXISTS devices_by_organisation_location ON devices (organisation_id, location_id, trust_state);
CREATE INDEX IF NOT EXISTS device_credentials_by_device ON device_credentials (device_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS audit_events_by_target ON audit_events (target_type, target_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_by_actor ON audit_events (actor_ref, created_at DESC);

INSERT INTO roles (name, scope, description)
VALUES
    ('platform_admin', 'platform', 'Seatd platform administrator'),
    ('organisation_owner', 'organisation', 'Organisation owner with administrative control'),
    ('location_manager', 'location', 'Manager for a specific location'),
    ('waiter', 'location', 'Waiter operational access for table and assist workflows'),
    ('read_only', 'organisation', 'Read-only organisation visibility'),
    ('support', 'platform', 'Seatd support operator')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (name, description)
VALUES
    ('platform.admin', 'Administer Seatd platform controls'),
    ('organisation.manage', 'Manage organisation configuration and members'),
    ('location.manage', 'Manage location configuration'),
    ('layout.read', 'Read floor, zone, and table layout'),
    ('layout.write', 'Create and modify floor, zone, and table layout'),
    ('operations.read', 'Read table occupancy and assist workflows'),
    ('operations.write', 'Modify table occupancy and assist workflows'),
    ('device.manage', 'Register, trust, and revoke devices'),
    ('audit.read', 'Read audit events')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES
    ('platform_admin', 'platform.admin'),
    ('platform_admin', 'audit.read'),
    ('organisation_owner', 'organisation.manage'),
    ('organisation_owner', 'location.manage'),
    ('organisation_owner', 'layout.read'),
    ('organisation_owner', 'layout.write'),
    ('organisation_owner', 'operations.read'),
    ('organisation_owner', 'operations.write'),
    ('organisation_owner', 'device.manage'),
    ('organisation_owner', 'audit.read'),
    ('location_manager', 'location.manage'),
    ('location_manager', 'layout.read'),
    ('location_manager', 'layout.write'),
    ('location_manager', 'operations.read'),
    ('location_manager', 'operations.write'),
    ('location_manager', 'device.manage'),
    ('waiter', 'layout.read'),
    ('waiter', 'operations.read'),
    ('waiter', 'operations.write'),
    ('read_only', 'layout.read'),
    ('read_only', 'operations.read'),
    ('support', 'audit.read')
ON CONFLICT (role_name, permission_name) DO NOTHING;

CREATE OR REPLACE FUNCTION seatd_is_platform_admin()
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT current_setting('seatd.platform_admin', true) = 'true';
$$;

CREATE OR REPLACE FUNCTION seatd_current_organisation_id()
RETURNS uuid
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('seatd.current_organisation_id', true), '')::uuid;
$$;

CREATE OR REPLACE FUNCTION seatd_current_location_id()
RETURNS uuid
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('seatd.current_location_id', true), '')::uuid;
$$;

CREATE OR REPLACE FUNCTION seatd_has_organisation_access(target_organisation_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT seatd_is_platform_admin()
        OR target_organisation_id = seatd_current_organisation_id();
$$;

CREATE OR REPLACE FUNCTION seatd_has_location_access(target_organisation_id uuid, target_location_id uuid)
RETURNS boolean
LANGUAGE sql
STABLE
AS $$
    SELECT seatd_is_platform_admin()
        OR (
            target_organisation_id = seatd_current_organisation_id()
            AND (
                seatd_current_location_id() IS NULL
                OR target_location_id = seatd_current_location_id()
            )
        );
$$;

DO $$
BEGIN
    CREATE ROLE seatd_app NOLOGIN NOSUPERUSER NOBYPASSRLS;
EXCEPTION
    WHEN duplicate_object THEN
        NULL;
    WHEN insufficient_privilege THEN
        NULL;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT USAGE ON SCHEMA public TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO seatd_app;
        GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO seatd_app;
    END IF;
END $$;

ALTER TABLE locations ENABLE ROW LEVEL SECURITY;
ALTER TABLE locations FORCE ROW LEVEL SECURITY;
ALTER TABLE organisation_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE organisation_memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE location_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE location_memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE floors ENABLE ROW LEVEL SECURITY;
ALTER TABLE floors FORCE ROW LEVEL SECURITY;
ALTER TABLE zones ENABLE ROW LEVEL SECURITY;
ALTER TABLE zones FORCE ROW LEVEL SECURITY;
ALTER TABLE tables ENABLE ROW LEVEL SECURITY;
ALTER TABLE tables FORCE ROW LEVEL SECURITY;
ALTER TABLE table_qr_capabilities ENABLE ROW LEVEL SECURITY;
ALTER TABLE table_qr_capabilities FORCE ROW LEVEL SECURITY;
ALTER TABLE table_occupancy ENABLE ROW LEVEL SECURITY;
ALTER TABLE table_occupancy FORCE ROW LEVEL SECURITY;
ALTER TABLE table_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE table_sessions FORCE ROW LEVEL SECURITY;
ALTER TABLE assist_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE assist_requests FORCE ROW LEVEL SECURITY;
ALTER TABLE devices ENABLE ROW LEVEL SECURITY;
ALTER TABLE devices FORCE ROW LEVEL SECURITY;
ALTER TABLE device_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_credentials FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS locations_tenant_isolation ON locations;
CREATE POLICY locations_tenant_isolation ON locations
    USING (seatd_has_location_access(organisation_id, id))
    WITH CHECK (seatd_has_location_access(organisation_id, id));

DROP POLICY IF EXISTS organisation_memberships_tenant_isolation ON organisation_memberships;
CREATE POLICY organisation_memberships_tenant_isolation ON organisation_memberships
    USING (seatd_has_organisation_access(organisation_id))
    WITH CHECK (seatd_has_organisation_access(organisation_id));

DROP POLICY IF EXISTS location_memberships_tenant_isolation ON location_memberships;
CREATE POLICY location_memberships_tenant_isolation ON location_memberships
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS floors_tenant_isolation ON floors;
CREATE POLICY floors_tenant_isolation ON floors
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS zones_tenant_isolation ON zones;
CREATE POLICY zones_tenant_isolation ON zones
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS tables_tenant_isolation ON tables;
CREATE POLICY tables_tenant_isolation ON tables
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS table_qr_capabilities_tenant_isolation ON table_qr_capabilities;
CREATE POLICY table_qr_capabilities_tenant_isolation ON table_qr_capabilities
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS table_occupancy_tenant_isolation ON table_occupancy;
CREATE POLICY table_occupancy_tenant_isolation ON table_occupancy
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS table_sessions_tenant_isolation ON table_sessions;
CREATE POLICY table_sessions_tenant_isolation ON table_sessions
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS assist_requests_tenant_isolation ON assist_requests;
CREATE POLICY assist_requests_tenant_isolation ON assist_requests
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS devices_tenant_isolation ON devices;
CREATE POLICY devices_tenant_isolation ON devices
    USING (seatd_has_organisation_access(organisation_id))
    WITH CHECK (seatd_has_organisation_access(organisation_id));

DROP POLICY IF EXISTS device_credentials_tenant_isolation ON device_credentials;
CREATE POLICY device_credentials_tenant_isolation ON device_credentials
    USING (seatd_has_organisation_access(organisation_id))
    WITH CHECK (seatd_has_organisation_access(organisation_id));

DROP POLICY IF EXISTS audit_events_tenant_isolation ON audit_events;
CREATE POLICY audit_events_tenant_isolation ON audit_events
    USING (seatd_is_platform_admin() OR organisation_id IS NULL OR organisation_id = seatd_current_organisation_id())
    WITH CHECK (seatd_is_platform_admin() OR organisation_id IS NULL OR organisation_id = seatd_current_organisation_id());
