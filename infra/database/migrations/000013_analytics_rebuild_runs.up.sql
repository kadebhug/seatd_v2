CREATE TABLE IF NOT EXISTS analytics_rebuild_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    requested_by text NOT NULL,
    range_from date NOT NULL,
    range_to date NOT NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz,
    status text NOT NULL DEFAULT 'running',
    days_processed integer NOT NULL DEFAULT 0,
    last_error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    CHECK (requested_by <> ''),
    CHECK (range_to > range_from),
    CHECK (status IN ('running', 'completed', 'failed')),
    CHECK (days_processed >= 0)
);

CREATE INDEX IF NOT EXISTS analytics_rebuild_runs_by_location
    ON analytics_rebuild_runs (organisation_id, location_id, started_at DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_rebuild_runs TO seatd_app;
    END IF;
END $$;

ALTER TABLE analytics_rebuild_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_rebuild_runs FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS analytics_rebuild_runs_tenant_isolation ON analytics_rebuild_runs;
CREATE POLICY analytics_rebuild_runs_tenant_isolation ON analytics_rebuild_runs
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));
