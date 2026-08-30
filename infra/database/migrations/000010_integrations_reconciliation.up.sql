CREATE TABLE IF NOT EXISTS integration_connections (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    vendor text NOT NULL,
    display_name text NOT NULL,
    credential_ref text NOT NULL,
    status text NOT NULL DEFAULT 'connected',
    capabilities jsonb NOT NULL DEFAULT '{}'::jsonb,
    config jsonb NOT NULL DEFAULT '{}'::jsonb,
    last_successful_sync_at timestamptz,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    UNIQUE (location_id, vendor),
    UNIQUE (id, organisation_id, location_id),
    CHECK (vendor <> ''),
    CHECK (display_name <> ''),
    CHECK (credential_ref <> ''),
    CHECK (status IN ('connected', 'disconnected', 'degraded')),
    CHECK (jsonb_typeof(capabilities) = 'object'),
    CHECK (jsonb_typeof(config) = 'object')
);

CREATE TABLE IF NOT EXISTS integration_table_mappings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    external_table_id text NOT NULL,
    table_id uuid,
    external_label text,
    status text NOT NULL DEFAULT 'mapped',
    last_seen_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connection_id, organisation_id, location_id) REFERENCES integration_connections (id, organisation_id, location_id) ON DELETE CASCADE,
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE RESTRICT,
    UNIQUE (connection_id, external_table_id),
    UNIQUE (connection_id, table_id),
    CHECK (external_table_id <> ''),
    CHECK (external_label IS NULL OR external_label <> ''),
    CHECK (status IN ('mapped', 'unmapped', 'ignored')),
    CHECK ((status = 'mapped' AND table_id IS NOT NULL) OR (status <> 'mapped'))
);

CREATE TABLE IF NOT EXISTS integration_webhook_inbox (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    vendor text NOT NULL,
    external_event_id text NOT NULL,
    received_at timestamptz NOT NULL DEFAULT now(),
    signature_valid boolean NOT NULL,
    payload jsonb NOT NULL,
    processing_state text NOT NULL DEFAULT 'received',
    attempts integer NOT NULL DEFAULT 0,
    last_error text,
    processed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connection_id, organisation_id, location_id) REFERENCES integration_connections (id, organisation_id, location_id) ON DELETE CASCADE,
    UNIQUE (vendor, external_event_id),
    CHECK (vendor <> ''),
    CHECK (external_event_id <> ''),
    CHECK (jsonb_typeof(payload) = 'object'),
    CHECK (processing_state IN ('received', 'processing', 'processed', 'failed', 'skipped')),
    CHECK (attempts >= 0)
);

CREATE TABLE IF NOT EXISTS integration_reconciliation_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    status text NOT NULL DEFAULT 'running',
    checked_count integer NOT NULL DEFAULT 0,
    discrepancy_count integer NOT NULL DEFAULT 0,
    auto_corrected_count integer NOT NULL DEFAULT 0,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connection_id, organisation_id, location_id) REFERENCES integration_connections (id, organisation_id, location_id) ON DELETE CASCADE,
    CHECK (status IN ('running', 'completed', 'failed')),
    CHECK (checked_count >= 0),
    CHECK (discrepancy_count >= 0),
    CHECK (auto_corrected_count >= 0)
);

CREATE TABLE IF NOT EXISTS integration_discrepancies (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    connection_id uuid NOT NULL,
    reconciliation_run_id uuid NOT NULL,
    mapping_id uuid,
    table_id uuid,
    external_table_id text NOT NULL,
    discrepancy_type text NOT NULL,
    seatd_state text,
    external_state text,
    resolution_state text NOT NULL DEFAULT 'review_required',
    details jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (connection_id, organisation_id, location_id) REFERENCES integration_connections (id, organisation_id, location_id) ON DELETE CASCADE,
    FOREIGN KEY (reconciliation_run_id) REFERENCES integration_reconciliation_runs (id) ON DELETE CASCADE,
    FOREIGN KEY (mapping_id) REFERENCES integration_table_mappings (id) ON DELETE SET NULL,
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE RESTRICT,
    CHECK (external_table_id <> ''),
    CHECK (discrepancy_type IN ('unmapped_table', 'occupancy_mismatch', 'ambiguous_conflict')),
    CHECK (seatd_state IS NULL OR seatd_state IN ('available', 'occupied')),
    CHECK (external_state IS NULL OR external_state IN ('available', 'occupied')),
    CHECK (resolution_state IN ('auto_corrected', 'review_required', 'ignored')),
    CHECK (jsonb_typeof(details) = 'object')
);

CREATE INDEX IF NOT EXISTS integration_connections_by_location
    ON integration_connections (organisation_id, location_id, status);

CREATE INDEX IF NOT EXISTS integration_table_mappings_by_connection
    ON integration_table_mappings (connection_id, status, external_table_id);

CREATE INDEX IF NOT EXISTS integration_webhook_inbox_by_connection
    ON integration_webhook_inbox (connection_id, received_at DESC);

CREATE INDEX IF NOT EXISTS integration_webhook_inbox_failed
    ON integration_webhook_inbox (updated_at DESC)
    WHERE processing_state = 'failed';

CREATE INDEX IF NOT EXISTS integration_discrepancies_open
    ON integration_discrepancies (connection_id, created_at DESC)
    WHERE resolution_state = 'review_required';

INSERT INTO permissions (name, description)
VALUES
    ('integrations.read', 'Read integration health, mappings, webhook inbox, and reconciliation records'),
    ('integrations.manage', 'Manage integration mappings, replay webhooks, and trigger reconciliation')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES
    ('platform_admin', 'integrations.read'),
    ('platform_admin', 'integrations.manage'),
    ('organisation_owner', 'integrations.read'),
    ('organisation_owner', 'integrations.manage'),
    ('location_manager', 'integrations.read'),
    ('location_manager', 'integrations.manage'),
    ('support', 'integrations.read')
ON CONFLICT (role_name, permission_name) DO NOTHING;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON integration_connections TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON integration_table_mappings TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON integration_webhook_inbox TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON integration_reconciliation_runs TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON integration_discrepancies TO seatd_app;
    END IF;
END $$;

ALTER TABLE integration_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_connections FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_table_mappings ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_table_mappings FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_webhook_inbox ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_webhook_inbox FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_reconciliation_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_reconciliation_runs FORCE ROW LEVEL SECURITY;
ALTER TABLE integration_discrepancies ENABLE ROW LEVEL SECURITY;
ALTER TABLE integration_discrepancies FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS integration_connections_tenant_isolation ON integration_connections;
CREATE POLICY integration_connections_tenant_isolation ON integration_connections
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS integration_table_mappings_tenant_isolation ON integration_table_mappings;
CREATE POLICY integration_table_mappings_tenant_isolation ON integration_table_mappings
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS integration_webhook_inbox_tenant_isolation ON integration_webhook_inbox;
CREATE POLICY integration_webhook_inbox_tenant_isolation ON integration_webhook_inbox
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS integration_reconciliation_runs_tenant_isolation ON integration_reconciliation_runs;
CREATE POLICY integration_reconciliation_runs_tenant_isolation ON integration_reconciliation_runs
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS integration_discrepancies_tenant_isolation ON integration_discrepancies;
CREATE POLICY integration_discrepancies_tenant_isolation ON integration_discrepancies
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));
