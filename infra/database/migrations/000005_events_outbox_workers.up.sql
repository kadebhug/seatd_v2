ALTER TABLE operational_events
    ADD COLUMN IF NOT EXISTS schema_version integer NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS entity_type text,
    ADD COLUMN IF NOT EXISTS entity_id uuid,
    ADD COLUMN IF NOT EXISTS entity_version integer,
    ADD COLUMN IF NOT EXISTS correlation_id uuid,
    ADD COLUMN IF NOT EXISTS event_data jsonb NOT NULL DEFAULT '{}'::jsonb;

UPDATE operational_events
SET entity_type = COALESCE(entity_type, aggregate_type),
    entity_id = COALESCE(entity_id, aggregate_id),
    entity_version = COALESCE(entity_version, aggregate_version),
    correlation_id = COALESCE(correlation_id, command_id),
    event_data = CASE
        WHEN event_data = '{}'::jsonb THEN payload
        ELSE event_data
    END
WHERE entity_type IS NULL
   OR entity_id IS NULL
   OR entity_version IS NULL
   OR correlation_id IS NULL
   OR event_data = '{}'::jsonb;

ALTER TABLE operational_events
    ALTER COLUMN entity_type SET NOT NULL,
    ALTER COLUMN entity_id SET NOT NULL,
    DROP CONSTRAINT IF EXISTS operational_events_schema_version_positive,
    DROP CONSTRAINT IF EXISTS operational_events_entity_type_not_empty,
    DROP CONSTRAINT IF EXISTS operational_events_entity_version_positive,
    DROP CONSTRAINT IF EXISTS operational_events_event_data_object,
    ADD CONSTRAINT operational_events_schema_version_positive CHECK (schema_version >= 1),
    ADD CONSTRAINT operational_events_entity_type_not_empty CHECK (entity_type <> ''),
    ADD CONSTRAINT operational_events_entity_version_positive CHECK (entity_version IS NULL OR entity_version >= 1),
    ADD CONSTRAINT operational_events_event_data_object CHECK (jsonb_typeof(event_data) = 'object');

CREATE INDEX IF NOT EXISTS operational_events_timeline
    ON operational_events (organisation_id, location_id, entity_type, entity_id, occurred_at, id);

ALTER TABLE outbox_records
    ADD COLUMN IF NOT EXISTS destination text,
    ADD COLUMN IF NOT EXISTS last_error text;

UPDATE outbox_records
SET destination = COALESCE(destination, topic)
WHERE destination IS NULL;

ALTER TABLE outbox_records
    ALTER COLUMN destination SET NOT NULL,
    DROP CONSTRAINT IF EXISTS outbox_records_destination_not_empty,
    ADD CONSTRAINT outbox_records_destination_not_empty CHECK (destination <> ''),
    DROP CONSTRAINT IF EXISTS outbox_records_status_check;

ALTER TABLE outbox_records
    ADD CONSTRAINT outbox_records_status_check CHECK (status IN ('pending', 'processing', 'processed', 'failed'));

CREATE INDEX IF NOT EXISTS outbox_records_claimable
    ON outbox_records (available_at, created_at, id)
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS outbox_records_dead_letters
    ON outbox_records (updated_at DESC)
    WHERE status = 'failed';

CREATE TABLE IF NOT EXISTS consumer_event_records (
    consumer_name text NOT NULL,
    event_id uuid NOT NULL REFERENCES operational_events (id) ON DELETE CASCADE,
    outbox_record_id uuid NOT NULL REFERENCES outbox_records (id) ON DELETE CASCADE,
    status text NOT NULL DEFAULT 'processing',
    first_seen_at timestamptz NOT NULL DEFAULT now(),
    processed_at timestamptz,
    last_error text,
    PRIMARY KEY (consumer_name, event_id),
    CHECK (consumer_name <> ''),
    CHECK (status IN ('processing', 'processed', 'failed'))
);

CREATE INDEX IF NOT EXISTS consumer_event_records_by_event
    ON consumer_event_records (event_id);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        REVOKE UPDATE, DELETE ON operational_events FROM seatd_app;
        GRANT SELECT, INSERT ON operational_events TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON outbox_records TO seatd_app;
        GRANT SELECT, INSERT, UPDATE ON consumer_event_records TO seatd_app;
    END IF;
END $$;

ALTER TABLE consumer_event_records ENABLE ROW LEVEL SECURITY;
ALTER TABLE consumer_event_records FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS consumer_event_records_tenant_isolation ON consumer_event_records;
CREATE POLICY consumer_event_records_tenant_isolation ON consumer_event_records
    USING (
        seatd_is_platform_admin()
        OR EXISTS (
            SELECT 1
            FROM operational_events oe
            WHERE oe.id = consumer_event_records.event_id
              AND seatd_has_location_access(oe.organisation_id, oe.location_id)
        )
    )
    WITH CHECK (
        seatd_is_platform_admin()
        OR EXISTS (
            SELECT 1
            FROM operational_events oe
            WHERE oe.id = consumer_event_records.event_id
              AND seatd_has_location_access(oe.organisation_id, oe.location_id)
        )
    );
