ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS name text,
    ADD COLUMN IF NOT EXISTS capabilities jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS configuration jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS assigned_at timestamptz,
    ADD COLUMN IF NOT EXISTS last_heartbeat_at timestamptz,
    ADD COLUMN IF NOT EXISTS heartbeat_interval_seconds integer NOT NULL DEFAULT 60;

ALTER TABLE devices
    DROP CONSTRAINT IF EXISTS devices_name_not_empty,
    ADD CONSTRAINT devices_name_not_empty CHECK (name IS NULL OR name <> ''),
    DROP CONSTRAINT IF EXISTS devices_capabilities_object,
    ADD CONSTRAINT devices_capabilities_object CHECK (jsonb_typeof(capabilities) = 'object'),
    DROP CONSTRAINT IF EXISTS devices_configuration_object,
    ADD CONSTRAINT devices_configuration_object CHECK (jsonb_typeof(configuration) = 'object'),
    DROP CONSTRAINT IF EXISTS devices_heartbeat_interval_positive,
    ADD CONSTRAINT devices_heartbeat_interval_positive CHECK (heartbeat_interval_seconds > 0);

CREATE TABLE IF NOT EXISTS device_pairing_codes (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    location_id uuid NOT NULL,
    created_by text NOT NULL,
    device_type text NOT NULL,
    pairing_code_lookup_prefix text NOT NULL,
    pairing_code_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    consumed_by_device_id uuid,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    FOREIGN KEY (consumed_by_device_id) REFERENCES devices (id) ON DELETE SET NULL,
    UNIQUE (pairing_code_lookup_prefix),
    CHECK (created_by <> ''),
    CHECK (device_type IN ('waiter_mobile', 'manager_tablet', 'display', 'host_device')),
    CHECK (pairing_code_lookup_prefix <> ''),
    CHECK (length(pairing_code_hash) >= 32),
    CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS devices_by_last_heartbeat
    ON devices (organisation_id, location_id, last_heartbeat_at DESC);

CREATE INDEX IF NOT EXISTS device_pairing_codes_active
    ON device_pairing_codes (organisation_id, location_id, device_type, expires_at)
    WHERE consumed_at IS NULL;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON device_pairing_codes TO seatd_app;
    END IF;
END $$;

ALTER TABLE device_pairing_codes ENABLE ROW LEVEL SECURITY;
ALTER TABLE device_pairing_codes FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS device_pairing_codes_tenant_isolation ON device_pairing_codes;
CREATE POLICY device_pairing_codes_tenant_isolation ON device_pairing_codes
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));
