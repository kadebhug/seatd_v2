DROP POLICY IF EXISTS analytics_projector_checkpoints_admin ON analytics_projector_checkpoints;
DROP POLICY IF EXISTS analytics_service_period_metrics_tenant_isolation ON analytics_service_period_metrics;
DROP POLICY IF EXISTS analytics_daily_location_metrics_tenant_isolation ON analytics_daily_location_metrics;
DROP POLICY IF EXISTS analytics_daily_floor_metrics_tenant_isolation ON analytics_daily_floor_metrics;
DROP POLICY IF EXISTS analytics_daily_zone_metrics_tenant_isolation ON analytics_daily_zone_metrics;
DROP POLICY IF EXISTS analytics_daily_table_metrics_tenant_isolation ON analytics_daily_table_metrics;
DROP POLICY IF EXISTS analytics_hourly_table_metrics_tenant_isolation ON analytics_hourly_table_metrics;
DROP POLICY IF EXISTS service_periods_tenant_isolation ON service_periods;

DROP TABLE IF EXISTS analytics_projector_checkpoints;
DROP TABLE IF EXISTS analytics_service_period_metrics;
DROP TABLE IF EXISTS analytics_daily_location_metrics;
DROP TABLE IF EXISTS analytics_daily_floor_metrics;
DROP TABLE IF EXISTS analytics_daily_zone_metrics;
DROP TABLE IF EXISTS analytics_daily_table_metrics;
DROP TABLE IF EXISTS analytics_hourly_table_metrics;
DROP TABLE IF EXISTS service_periods;

DELETE FROM role_permissions WHERE permission_name = 'analytics.read';
DELETE FROM permissions WHERE name = 'analytics.read';
