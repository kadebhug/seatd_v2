-- name: ListServicePeriodsByLocation :many
SELECT *
FROM service_periods
WHERE organisation_id = $1
  AND location_id = $2
ORDER BY is_active DESC, name;

-- name: ListActiveServicePeriodsByLocation :many
SELECT *
FROM service_periods
WHERE organisation_id = $1
  AND location_id = $2
  AND is_active
ORDER BY name;

-- name: CreateServicePeriod :one
INSERT INTO service_periods (
    organisation_id,
    location_id,
    name,
    days_of_week,
    start_time,
    end_time
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateServicePeriod :one
UPDATE service_periods
SET name = $4,
    days_of_week = $5,
    start_time = $6,
    end_time = $7,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $8
RETURNING *;

-- name: ArchiveServicePeriod :one
UPDATE service_periods
SET is_active = false,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $4
RETURNING *;

-- name: GetDailyLocationMetric :one
SELECT *
FROM analytics_daily_location_metrics
WHERE organisation_id = $1
  AND location_id = $2
  AND metric_date = $3;

-- name: ListHourlyTableMetrics :many
SELECT *
FROM analytics_hourly_table_metrics
WHERE organisation_id = $1
  AND location_id = $2
  AND bucket_start >= $3
  AND bucket_start < $4
ORDER BY bucket_start, table_id;

-- name: ListDailyLocationMetrics :many
SELECT *
FROM analytics_daily_location_metrics
WHERE organisation_id = $1
  AND location_id = $2
  AND metric_date >= $3
  AND metric_date < $4
ORDER BY metric_date;

-- name: ListDailyZoneMetrics :many
SELECT m.*, z.name AS group_name
FROM analytics_daily_zone_metrics m
JOIN zones z ON z.id = m.zone_id
WHERE m.organisation_id = $1
  AND m.location_id = $2
  AND m.metric_date >= $3
  AND m.metric_date < $4
ORDER BY z.sort_order, z.name, m.metric_date;

-- name: ListDailyFloorMetrics :many
SELECT m.*, f.name AS group_name
FROM analytics_daily_floor_metrics m
JOIN floors f ON f.id = m.floor_id
WHERE m.organisation_id = $1
  AND m.location_id = $2
  AND m.metric_date >= $3
  AND m.metric_date < $4
ORDER BY f.sort_order, f.name, m.metric_date;

-- name: ListServicePeriodMetrics :many
SELECT m.*, sp.name AS group_name
FROM analytics_service_period_metrics m
JOIN service_periods sp ON sp.id = m.service_period_id
WHERE m.organisation_id = $1
  AND m.location_id = $2
  AND m.metric_date >= $3
  AND m.metric_date < $4
ORDER BY sp.name, m.metric_date;

-- name: GetAnalyticsProjectorCheckpoint :one
SELECT *
FROM analytics_projector_checkpoints
WHERE projector_name = $1;
