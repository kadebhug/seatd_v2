CREATE TABLE IF NOT EXISTS api_idempotency_records (
    command_id uuid NOT NULL,
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    location_id uuid,
    command_type text NOT NULL,
    payload_hash bytea NOT NULL,
    response_status integer NOT NULL,
    response_body jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    PRIMARY KEY (organisation_id, command_id),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    CHECK (command_type <> ''),
    CHECK (length(payload_hash) >= 32),
    CHECK (response_status >= 100 AND response_status <= 599),
    CHECK (jsonb_typeof(response_body) = 'object'),
    CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS api_idempotency_records_expiry
    ON api_idempotency_records (expires_at);

CREATE TABLE IF NOT EXISTS operational_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    location_id uuid NOT NULL,
    event_type text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id uuid NOT NULL,
    aggregate_version integer,
    actor_ref text NOT NULL,
    device_id uuid,
    command_id uuid,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE SET NULL,
    CHECK (event_type <> ''),
    CHECK (aggregate_type <> ''),
    CHECK (actor_ref <> ''),
    CHECK (aggregate_version IS NULL OR aggregate_version >= 1),
    CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS operational_events_by_location
    ON operational_events (organisation_id, location_id, occurred_at DESC);

CREATE INDEX IF NOT EXISTS operational_events_by_aggregate
    ON operational_events (aggregate_type, aggregate_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS outbox_records (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    location_id uuid NOT NULL,
    event_id uuid NOT NULL REFERENCES operational_events (id) ON DELETE CASCADE,
    topic text NOT NULL,
    payload jsonb NOT NULL,
    status text NOT NULL DEFAULT 'pending',
    attempts integer NOT NULL DEFAULT 0,
    available_at timestamptz NOT NULL DEFAULT now(),
    locked_at timestamptz,
    locked_by text,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    CHECK (topic <> ''),
    CHECK (jsonb_typeof(payload) = 'object'),
    CHECK (status IN ('pending', 'processing', 'processed', 'failed')),
    CHECK (attempts >= 0)
);

CREATE INDEX IF NOT EXISTS outbox_records_pending
    ON outbox_records (status, available_at, created_at)
    WHERE status IN ('pending', 'failed');

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON api_idempotency_records TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON operational_events TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON outbox_records TO seatd_app;
    END IF;
END $$;

ALTER TABLE api_idempotency_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE api_idempotency_records FORCE ROW LEVEL SECURITY;
ALTER TABLE operational_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE operational_events FORCE ROW LEVEL SECURITY;
ALTER TABLE outbox_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_records FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS api_idempotency_records_tenant_isolation ON api_idempotency_records;
CREATE POLICY api_idempotency_records_tenant_isolation ON api_idempotency_records
    USING (seatd_has_organisation_access(organisation_id))
    WITH CHECK (seatd_has_organisation_access(organisation_id));

DROP POLICY IF EXISTS operational_events_tenant_isolation ON operational_events;
CREATE POLICY operational_events_tenant_isolation ON operational_events
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS outbox_records_tenant_isolation ON outbox_records;
CREATE POLICY outbox_records_tenant_isolation ON outbox_records
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));
