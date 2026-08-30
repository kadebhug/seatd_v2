DROP POLICY IF EXISTS consumer_event_records_tenant_isolation ON consumer_event_records;
ALTER TABLE consumer_event_records DISABLE ROW LEVEL SECURITY;
DROP TABLE IF EXISTS consumer_event_records;

DROP INDEX IF EXISTS outbox_records_dead_letters;
DROP INDEX IF EXISTS outbox_records_claimable;
ALTER TABLE outbox_records
    DROP CONSTRAINT IF EXISTS outbox_records_destination_not_empty,
    DROP COLUMN IF EXISTS last_error,
    DROP COLUMN IF EXISTS destination;

DROP INDEX IF EXISTS operational_events_timeline;
ALTER TABLE operational_events
    DROP CONSTRAINT IF EXISTS operational_events_event_data_object,
    DROP CONSTRAINT IF EXISTS operational_events_entity_version_positive,
    DROP CONSTRAINT IF EXISTS operational_events_entity_type_not_empty,
    DROP CONSTRAINT IF EXISTS operational_events_schema_version_positive,
    DROP COLUMN IF EXISTS event_data,
    DROP COLUMN IF EXISTS correlation_id,
    DROP COLUMN IF EXISTS entity_version,
    DROP COLUMN IF EXISTS entity_id,
    DROP COLUMN IF EXISTS entity_type,
    DROP COLUMN IF EXISTS schema_version;
