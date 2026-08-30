DROP POLICY IF EXISTS outbox_records_tenant_isolation ON outbox_records;
DROP POLICY IF EXISTS operational_events_tenant_isolation ON operational_events;
DROP POLICY IF EXISTS api_idempotency_records_tenant_isolation ON api_idempotency_records;

ALTER TABLE outbox_records DISABLE ROW LEVEL SECURITY;
ALTER TABLE operational_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE api_idempotency_records DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS outbox_records;
DROP TABLE IF EXISTS operational_events;
DROP TABLE IF EXISTS api_idempotency_records;
