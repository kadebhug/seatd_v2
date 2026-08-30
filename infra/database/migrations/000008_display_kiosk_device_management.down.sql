DROP POLICY IF EXISTS device_pairing_codes_tenant_isolation ON device_pairing_codes;
ALTER TABLE device_pairing_codes DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS device_pairing_codes;

DROP INDEX IF EXISTS devices_by_last_heartbeat;

ALTER TABLE devices
    DROP CONSTRAINT IF EXISTS devices_heartbeat_interval_positive,
    DROP CONSTRAINT IF EXISTS devices_configuration_object,
    DROP CONSTRAINT IF EXISTS devices_capabilities_object,
    DROP CONSTRAINT IF EXISTS devices_name_not_empty,
    DROP COLUMN IF EXISTS heartbeat_interval_seconds,
    DROP COLUMN IF EXISTS last_heartbeat_at,
    DROP COLUMN IF EXISTS assigned_at,
    DROP COLUMN IF EXISTS configuration,
    DROP COLUMN IF EXISTS capabilities,
    DROP COLUMN IF EXISTS name;
