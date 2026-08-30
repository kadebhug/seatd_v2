DROP POLICY IF EXISTS guest_abuse_events_tenant_read ON guest_abuse_events;
DROP POLICY IF EXISTS guest_abuse_events_insert_public ON guest_abuse_events;
DROP TABLE IF EXISTS guest_abuse_events;

DROP INDEX IF EXISTS assist_requests_guest_terminal_action;
DROP INDEX IF EXISTS assist_requests_guest_pending_action;

ALTER TABLE assist_requests
    DROP CONSTRAINT IF EXISTS assist_requests_guest_action_key_required,
    DROP COLUMN IF EXISTS action_key;
