DROP POLICY IF EXISTS audit_events_tenant_isolation ON audit_events;
DROP POLICY IF EXISTS device_credentials_tenant_isolation ON device_credentials;
DROP POLICY IF EXISTS devices_tenant_isolation ON devices;
DROP POLICY IF EXISTS assist_requests_tenant_isolation ON assist_requests;
DROP POLICY IF EXISTS table_sessions_tenant_isolation ON table_sessions;
DROP POLICY IF EXISTS table_occupancy_tenant_isolation ON table_occupancy;
DROP POLICY IF EXISTS table_qr_capabilities_tenant_isolation ON table_qr_capabilities;
DROP POLICY IF EXISTS tables_tenant_isolation ON tables;
DROP POLICY IF EXISTS zones_tenant_isolation ON zones;
DROP POLICY IF EXISTS floors_tenant_isolation ON floors;
DROP POLICY IF EXISTS location_memberships_tenant_isolation ON location_memberships;
DROP POLICY IF EXISTS organisation_memberships_tenant_isolation ON organisation_memberships;
DROP POLICY IF EXISTS locations_tenant_isolation ON locations;

ALTER TABLE audit_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE device_credentials DISABLE ROW LEVEL SECURITY;
ALTER TABLE devices DISABLE ROW LEVEL SECURITY;
ALTER TABLE assist_requests DISABLE ROW LEVEL SECURITY;
ALTER TABLE table_sessions DISABLE ROW LEVEL SECURITY;
ALTER TABLE table_occupancy DISABLE ROW LEVEL SECURITY;
ALTER TABLE table_qr_capabilities DISABLE ROW LEVEL SECURITY;
ALTER TABLE tables DISABLE ROW LEVEL SECURITY;
ALTER TABLE zones DISABLE ROW LEVEL SECURITY;
ALTER TABLE floors DISABLE ROW LEVEL SECURITY;
ALTER TABLE location_memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE organisation_memberships DISABLE ROW LEVEL SECURITY;
ALTER TABLE locations DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS seatd_has_location_access(uuid, uuid);
DROP FUNCTION IF EXISTS seatd_has_organisation_access(uuid);
DROP FUNCTION IF EXISTS seatd_current_location_id();
DROP FUNCTION IF EXISTS seatd_current_organisation_id();
DROP FUNCTION IF EXISTS seatd_is_platform_admin();

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        REVOKE EXECUTE ON ALL FUNCTIONS IN SCHEMA public FROM seatd_app;
        REVOKE SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public FROM seatd_app;
        REVOKE USAGE ON SCHEMA public FROM seatd_app;
    END IF;
END $$;

DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS device_credentials;
DROP TABLE IF EXISTS devices;

ALTER TABLE table_qr_capabilities
    DROP COLUMN IF EXISTS last_used_at,
    DROP COLUMN IF EXISTS token_hash,
    DROP COLUMN IF EXISTS token_lookup_prefix;

ALTER TABLE location_memberships
    DROP CONSTRAINT IF EXISTS location_memberships_role_fk,
    DROP COLUMN IF EXISTS disabled_at,
    DROP COLUMN IF EXISTS user_profile_id;

ALTER TABLE organisation_memberships
    DROP CONSTRAINT IF EXISTS organisation_memberships_role_fk,
    DROP COLUMN IF EXISTS disabled_at,
    DROP COLUMN IF EXISTS user_profile_id;

DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS external_identities;
DROP TABLE IF EXISTS user_profiles;
