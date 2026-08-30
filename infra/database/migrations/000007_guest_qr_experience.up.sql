ALTER TABLE assist_requests
    ADD COLUMN IF NOT EXISTS action_key text;

UPDATE assist_requests
SET action_key = 'request_service'
WHERE action_key IS NULL
  AND source = 'guest_qr';

ALTER TABLE assist_requests
    DROP CONSTRAINT IF EXISTS assist_requests_guest_action_key_required,
    ADD CONSTRAINT assist_requests_guest_action_key_required
    CHECK (source <> 'guest_qr' OR (action_key IS NOT NULL AND action_key <> ''));

CREATE INDEX IF NOT EXISTS assist_requests_guest_pending_action
    ON assist_requests (organisation_id, location_id, table_id, table_session_id, action_key, status, requested_at)
    WHERE source = 'guest_qr' AND status = 'pending';

CREATE INDEX IF NOT EXISTS assist_requests_guest_terminal_action
    ON assist_requests (organisation_id, location_id, table_id, table_session_id, action_key, requested_at DESC)
    WHERE source = 'guest_qr' AND status IN ('resolved', 'cancelled');

CREATE TABLE IF NOT EXISTS guest_abuse_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid,
    location_id uuid,
    table_id uuid,
    token_lookup_prefix text NOT NULL,
    action_key text,
    ip_hash bytea NOT NULL,
    decision text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (token_lookup_prefix <> ''),
    CHECK (action_key IS NULL OR action_key <> ''),
    CHECK (length(ip_hash) >= 32),
    CHECK (decision <> '')
);

CREATE INDEX IF NOT EXISTS guest_abuse_events_by_prefix
    ON guest_abuse_events (token_lookup_prefix, created_at DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON guest_abuse_events TO seatd_app;
    END IF;
END $$;

ALTER TABLE guest_abuse_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE guest_abuse_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS guest_abuse_events_insert_public ON guest_abuse_events;
CREATE POLICY guest_abuse_events_insert_public ON guest_abuse_events
    FOR INSERT
    WITH CHECK (true);

DROP POLICY IF EXISTS guest_abuse_events_tenant_read ON guest_abuse_events;
CREATE POLICY guest_abuse_events_tenant_read ON guest_abuse_events
    FOR SELECT
    USING (
        organisation_id IS NULL
        OR seatd_has_location_access(organisation_id, location_id)
    );
