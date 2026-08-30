CREATE TABLE IF NOT EXISTS service_periods (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    name text NOT NULL,
    days_of_week smallint[] NOT NULL,
    start_time time NOT NULL,
    end_time time NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    UNIQUE (location_id, name),
    CHECK (name <> ''),
    CHECK (cardinality(days_of_week) BETWEEN 1 AND 7),
    CHECK (days_of_week <@ ARRAY[0,1,2,3,4,5,6]::smallint[]),
    CHECK (version >= 1)
);

CREATE TABLE IF NOT EXISTS analytics_hourly_table_metrics (
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    table_id uuid NOT NULL,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    active_table_count integer NOT NULL DEFAULT 1,
    occupancy_seconds double precision NOT NULL DEFAULT 0,
    utilisation_basis_seconds double precision NOT NULL DEFAULT 0,
    utilisation_rate double precision NOT NULL DEFAULT 0,
    completed_session_count integer NOT NULL DEFAULT 0,
    turnover_rate double precision NOT NULL DEFAULT 0,
    avg_session_seconds double precision NOT NULL DEFAULT 0,
    p50_session_seconds double precision NOT NULL DEFAULT 0,
    p90_session_seconds double precision NOT NULL DEFAULT 0,
    assist_request_count integer NOT NULL DEFAULT 0,
    avg_assist_response_seconds double precision NOT NULL DEFAULT 0,
    avg_assist_resolution_seconds double precision NOT NULL DEFAULT 0,
    anomaly_count integer NOT NULL DEFAULT 0,
    projected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, location_id, table_id, bucket_start),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE CASCADE,
    CHECK (bucket_end > bucket_start),
    CHECK (active_table_count >= 0),
    CHECK (occupancy_seconds >= 0),
    CHECK (utilisation_basis_seconds >= 0),
    CHECK (utilisation_rate >= 0),
    CHECK (completed_session_count >= 0),
    CHECK (turnover_rate >= 0),
    CHECK (assist_request_count >= 0),
    CHECK (anomaly_count >= 0)
);

CREATE TABLE IF NOT EXISTS analytics_daily_table_metrics (
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    table_id uuid NOT NULL,
    metric_date date NOT NULL,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    active_table_count integer NOT NULL DEFAULT 1,
    occupancy_seconds double precision NOT NULL DEFAULT 0,
    utilisation_basis_seconds double precision NOT NULL DEFAULT 0,
    utilisation_rate double precision NOT NULL DEFAULT 0,
    completed_session_count integer NOT NULL DEFAULT 0,
    turnover_rate double precision NOT NULL DEFAULT 0,
    avg_session_seconds double precision NOT NULL DEFAULT 0,
    p50_session_seconds double precision NOT NULL DEFAULT 0,
    p90_session_seconds double precision NOT NULL DEFAULT 0,
    assist_request_count integer NOT NULL DEFAULT 0,
    avg_assist_response_seconds double precision NOT NULL DEFAULT 0,
    avg_assist_resolution_seconds double precision NOT NULL DEFAULT 0,
    anomaly_count integer NOT NULL DEFAULT 0,
    projected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, location_id, table_id, metric_date),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE CASCADE,
    CHECK (bucket_end > bucket_start)
);

CREATE TABLE IF NOT EXISTS analytics_daily_zone_metrics (
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    zone_id uuid NOT NULL,
    metric_date date NOT NULL,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    active_table_count integer NOT NULL DEFAULT 0,
    occupancy_seconds double precision NOT NULL DEFAULT 0,
    utilisation_basis_seconds double precision NOT NULL DEFAULT 0,
    utilisation_rate double precision NOT NULL DEFAULT 0,
    completed_session_count integer NOT NULL DEFAULT 0,
    turnover_rate double precision NOT NULL DEFAULT 0,
    avg_session_seconds double precision NOT NULL DEFAULT 0,
    p50_session_seconds double precision NOT NULL DEFAULT 0,
    p90_session_seconds double precision NOT NULL DEFAULT 0,
    assist_request_count integer NOT NULL DEFAULT 0,
    avg_assist_response_seconds double precision NOT NULL DEFAULT 0,
    avg_assist_resolution_seconds double precision NOT NULL DEFAULT 0,
    anomaly_count integer NOT NULL DEFAULT 0,
    projected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, location_id, zone_id, metric_date),
    FOREIGN KEY (zone_id) REFERENCES zones (id) ON DELETE CASCADE,
    CHECK (bucket_end > bucket_start)
);

CREATE TABLE IF NOT EXISTS analytics_daily_floor_metrics (
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    floor_id uuid NOT NULL,
    metric_date date NOT NULL,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    active_table_count integer NOT NULL DEFAULT 0,
    occupancy_seconds double precision NOT NULL DEFAULT 0,
    utilisation_basis_seconds double precision NOT NULL DEFAULT 0,
    utilisation_rate double precision NOT NULL DEFAULT 0,
    completed_session_count integer NOT NULL DEFAULT 0,
    turnover_rate double precision NOT NULL DEFAULT 0,
    avg_session_seconds double precision NOT NULL DEFAULT 0,
    p50_session_seconds double precision NOT NULL DEFAULT 0,
    p90_session_seconds double precision NOT NULL DEFAULT 0,
    assist_request_count integer NOT NULL DEFAULT 0,
    avg_assist_response_seconds double precision NOT NULL DEFAULT 0,
    avg_assist_resolution_seconds double precision NOT NULL DEFAULT 0,
    anomaly_count integer NOT NULL DEFAULT 0,
    projected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, location_id, floor_id, metric_date),
    FOREIGN KEY (floor_id, organisation_id, location_id) REFERENCES floors (id, organisation_id, location_id) ON DELETE CASCADE,
    CHECK (bucket_end > bucket_start)
);

CREATE TABLE IF NOT EXISTS analytics_daily_location_metrics (
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    metric_date date NOT NULL,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    active_table_count integer NOT NULL DEFAULT 0,
    occupancy_seconds double precision NOT NULL DEFAULT 0,
    utilisation_basis_seconds double precision NOT NULL DEFAULT 0,
    utilisation_rate double precision NOT NULL DEFAULT 0,
    completed_session_count integer NOT NULL DEFAULT 0,
    turnover_rate double precision NOT NULL DEFAULT 0,
    avg_session_seconds double precision NOT NULL DEFAULT 0,
    p50_session_seconds double precision NOT NULL DEFAULT 0,
    p90_session_seconds double precision NOT NULL DEFAULT 0,
    assist_request_count integer NOT NULL DEFAULT 0,
    avg_assist_response_seconds double precision NOT NULL DEFAULT 0,
    avg_assist_resolution_seconds double precision NOT NULL DEFAULT 0,
    anomaly_count integer NOT NULL DEFAULT 0,
    projected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, location_id, metric_date),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    CHECK (bucket_end > bucket_start)
);

CREATE TABLE IF NOT EXISTS analytics_service_period_metrics (
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    service_period_id uuid NOT NULL,
    metric_date date NOT NULL,
    bucket_start timestamptz NOT NULL,
    bucket_end timestamptz NOT NULL,
    active_table_count integer NOT NULL DEFAULT 0,
    occupancy_seconds double precision NOT NULL DEFAULT 0,
    utilisation_basis_seconds double precision NOT NULL DEFAULT 0,
    utilisation_rate double precision NOT NULL DEFAULT 0,
    completed_session_count integer NOT NULL DEFAULT 0,
    turnover_rate double precision NOT NULL DEFAULT 0,
    avg_session_seconds double precision NOT NULL DEFAULT 0,
    p50_session_seconds double precision NOT NULL DEFAULT 0,
    p90_session_seconds double precision NOT NULL DEFAULT 0,
    assist_request_count integer NOT NULL DEFAULT 0,
    avg_assist_response_seconds double precision NOT NULL DEFAULT 0,
    avg_assist_resolution_seconds double precision NOT NULL DEFAULT 0,
    anomaly_count integer NOT NULL DEFAULT 0,
    projected_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organisation_id, location_id, service_period_id, metric_date),
    FOREIGN KEY (service_period_id) REFERENCES service_periods (id) ON DELETE CASCADE,
    CHECK (bucket_end > bucket_start)
);

CREATE TABLE IF NOT EXISTS analytics_projector_checkpoints (
    projector_name text PRIMARY KEY,
    projector_version integer NOT NULL,
    cursor_occurred_at timestamptz,
    cursor_event_id uuid,
    lag_seconds double precision NOT NULL DEFAULT 0,
    rebuild_status text NOT NULL DEFAULT 'idle',
    rebuild_started_at timestamptz,
    rebuild_completed_at timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (projector_name <> ''),
    CHECK (projector_version >= 1),
    CHECK (lag_seconds >= 0),
    CHECK (rebuild_status IN ('idle', 'running', 'failed'))
);

CREATE INDEX IF NOT EXISTS service_periods_by_location ON service_periods (organisation_id, location_id, is_active, name);
CREATE INDEX IF NOT EXISTS analytics_hourly_table_metrics_by_location ON analytics_hourly_table_metrics (organisation_id, location_id, bucket_start);
CREATE INDEX IF NOT EXISTS analytics_daily_table_metrics_by_location ON analytics_daily_table_metrics (organisation_id, location_id, metric_date);
CREATE INDEX IF NOT EXISTS analytics_daily_zone_metrics_by_location ON analytics_daily_zone_metrics (organisation_id, location_id, metric_date);
CREATE INDEX IF NOT EXISTS analytics_daily_floor_metrics_by_location ON analytics_daily_floor_metrics (organisation_id, location_id, metric_date);
CREATE INDEX IF NOT EXISTS analytics_service_period_metrics_by_location ON analytics_service_period_metrics (organisation_id, location_id, metric_date);

INSERT INTO permissions (name, description)
VALUES ('analytics.read', 'Read analytics dashboards and projections')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES
    ('platform_admin', 'analytics.read'),
    ('organisation_owner', 'analytics.read'),
    ('location_manager', 'analytics.read'),
    ('read_only', 'analytics.read')
ON CONFLICT (role_name, permission_name) DO NOTHING;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON service_periods TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_hourly_table_metrics TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_daily_table_metrics TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_daily_zone_metrics TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_daily_floor_metrics TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_daily_location_metrics TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_service_period_metrics TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON analytics_projector_checkpoints TO seatd_app;
    END IF;
END $$;

ALTER TABLE service_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE service_periods FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_hourly_table_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_hourly_table_metrics FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_table_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_table_metrics FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_zone_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_zone_metrics FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_floor_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_floor_metrics FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_location_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_daily_location_metrics FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_service_period_metrics ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_service_period_metrics FORCE ROW LEVEL SECURITY;
ALTER TABLE analytics_projector_checkpoints ENABLE ROW LEVEL SECURITY;
ALTER TABLE analytics_projector_checkpoints FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS service_periods_tenant_isolation ON service_periods;
CREATE POLICY service_periods_tenant_isolation ON service_periods
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_hourly_table_metrics_tenant_isolation ON analytics_hourly_table_metrics;
CREATE POLICY analytics_hourly_table_metrics_tenant_isolation ON analytics_hourly_table_metrics
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_daily_table_metrics_tenant_isolation ON analytics_daily_table_metrics;
CREATE POLICY analytics_daily_table_metrics_tenant_isolation ON analytics_daily_table_metrics
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_daily_zone_metrics_tenant_isolation ON analytics_daily_zone_metrics;
CREATE POLICY analytics_daily_zone_metrics_tenant_isolation ON analytics_daily_zone_metrics
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_daily_floor_metrics_tenant_isolation ON analytics_daily_floor_metrics;
CREATE POLICY analytics_daily_floor_metrics_tenant_isolation ON analytics_daily_floor_metrics
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_daily_location_metrics_tenant_isolation ON analytics_daily_location_metrics;
CREATE POLICY analytics_daily_location_metrics_tenant_isolation ON analytics_daily_location_metrics
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_service_period_metrics_tenant_isolation ON analytics_service_period_metrics;
CREATE POLICY analytics_service_period_metrics_tenant_isolation ON analytics_service_period_metrics
    USING (seatd_has_location_access(organisation_id, location_id))
    WITH CHECK (seatd_has_location_access(organisation_id, location_id));

DROP POLICY IF EXISTS analytics_projector_checkpoints_admin ON analytics_projector_checkpoints;
CREATE POLICY analytics_projector_checkpoints_admin ON analytics_projector_checkpoints
    USING (seatd_is_platform_admin())
    WITH CHECK (seatd_is_platform_admin());
