package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/seatd/seatd/internal/domain/events"
	"github.com/seatd/seatd/internal/store/db"
)

type Projector struct {
	pool *pgxpool.Pool
}

func NewProjector(pool *pgxpool.Pool) *Projector {
	return &Projector{pool: pool}
}

func (p *Projector) Name() string {
	return ProjectorName
}

func (p *Projector) Handle(ctx context.Context, event events.Envelope) error {
	tx, err := p.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning analytics projection transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting analytics projector context: %w", err))
	}
	q := db.New(tx)
	location, err := q.GetLocation(ctx, db.GetLocationParams{
		ID:             event.LocationID,
		OrganisationID: event.OrganisationID,
	})
	if err != nil {
		return rollback(tx, ctx, fmt.Errorf("getting projector location: %w", err))
	}
	tz, err := time.LoadLocation(location.Timezone)
	if err != nil {
		return rollback(tx, ctx, fmt.Errorf("loading location timezone: %w", err))
	}
	start, end := localDateWindow(event.OccurredAt, tz)
	if err := rebuildRange(ctx, tx, q, event.OrganisationID, event.LocationID, start, end); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := upsertCheckpoint(ctx, tx, event); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing analytics projection transaction: %w", err)
	}
	return nil
}

func rebuildRange(
	ctx context.Context,
	tx pgx.Tx,
	q *db.Queries,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	from time.Time,
	to time.Time,
) error {
	location, err := q.GetLocation(ctx, db.GetLocationParams{ID: locationID, OrganisationID: organisationID})
	if err != nil {
		return fmt.Errorf("getting rebuild location: %w", err)
	}
	tz, err := time.LoadLocation(location.Timezone)
	if err != nil {
		return fmt.Errorf("loading rebuild timezone: %w", err)
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO analytics_projector_checkpoints (projector_name, projector_version, rebuild_status, rebuild_started_at, updated_at)
VALUES ($1, $2, 'running', now(), now())
ON CONFLICT (projector_name) DO UPDATE
SET projector_version = EXCLUDED.projector_version,
    rebuild_status = 'running',
    rebuild_started_at = now(),
    updated_at = now()
`, ProjectorName, ProjectorVersion); err != nil {
		return fmt.Errorf("marking analytics rebuild running: %w", err)
	}

	for day := calendarDateInLocation(from, tz); day.Before(calendarDateInLocation(to, tz)); day = day.AddDate(0, 0, 1) {
		dayStart, dayEnd := localCalendarDateWindow(day, tz)
		if err := clearProjectionDay(ctx, tx, organisationID, locationID, day); err != nil {
			return err
		}
		if err := projectDay(ctx, tx, q, organisationID, locationID, day, dayStart, dayEnd); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(ctx, `
UPDATE analytics_projector_checkpoints
SET rebuild_status = 'idle',
    rebuild_completed_at = now(),
    updated_at = now()
WHERE projector_name = $1
`, ProjectorName); err != nil {
		return fmt.Errorf("marking analytics rebuild complete: %w", err)
	}
	return nil
}

func clearProjectionDay(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, locationID uuid.UUID, day time.Time) error {
	start, end := day, day.AddDate(0, 0, 1)
	if _, err := tx.Exec(ctx, `
DELETE FROM analytics_hourly_table_metrics
WHERE organisation_id = $1 AND location_id = $2 AND bucket_start >= $3 AND bucket_start < $4
`, organisationID, locationID, start, end); err != nil {
		return fmt.Errorf("clearing hourly analytics projection: %w", err)
	}
	for _, statement := range []string{
		`DELETE FROM analytics_daily_table_metrics WHERE organisation_id = $1 AND location_id = $2 AND metric_date = $3`,
		`DELETE FROM analytics_daily_zone_metrics WHERE organisation_id = $1 AND location_id = $2 AND metric_date = $3`,
		`DELETE FROM analytics_daily_floor_metrics WHERE organisation_id = $1 AND location_id = $2 AND metric_date = $3`,
		`DELETE FROM analytics_daily_location_metrics WHERE organisation_id = $1 AND location_id = $2 AND metric_date = $3`,
		`DELETE FROM analytics_service_period_metrics WHERE organisation_id = $1 AND location_id = $2 AND metric_date = $3`,
	} {
		if _, err := tx.Exec(ctx, statement, organisationID, locationID, day); err != nil {
			return fmt.Errorf("clearing analytics projection: %w", err)
		}
	}
	return nil
}

func projectDay(
	ctx context.Context,
	tx pgx.Tx,
	q *db.Queries,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	day time.Time,
	dayStart time.Time,
	dayEnd time.Time,
) error {
	if err := projectLocation(ctx, tx, organisationID, locationID, day, dayStart, dayEnd); err != nil {
		return err
	}
	if err := projectGrouped(ctx, tx, organisationID, locationID, day, dayStart, dayEnd, GroupByFloor); err != nil {
		return err
	}
	if err := projectGrouped(ctx, tx, organisationID, locationID, day, dayStart, dayEnd, GroupByZone); err != nil {
		return err
	}
	if err := projectTables(ctx, tx, organisationID, locationID, day, dayStart, dayEnd, "day"); err != nil {
		return err
	}
	for hourStart := dayStart; hourStart.Before(dayEnd); hourStart = hourStart.Add(time.Hour) {
		hourEnd := hourStart.Add(time.Hour)
		if hourEnd.After(dayEnd) {
			hourEnd = dayEnd
		}
		if err := projectTables(ctx, tx, organisationID, locationID, day, hourStart, hourEnd, "hour"); err != nil {
			return err
		}
	}
	periods, err := q.ListActiveServicePeriodsByLocation(ctx, db.ListActiveServicePeriodsByLocationParams{
		OrganisationID: organisationID,
		LocationID:     locationID,
	})
	if err != nil {
		return fmt.Errorf("listing projection service periods: %w", err)
	}
	location, err := q.GetLocation(ctx, db.GetLocationParams{ID: locationID, OrganisationID: organisationID})
	if err != nil {
		return fmt.Errorf("getting service period location: %w", err)
	}
	tz, err := time.LoadLocation(location.Timezone)
	if err != nil {
		return fmt.Errorf("loading service period timezone: %w", err)
	}
	for _, period := range periods {
		window, ok, err := ServicePeriodWindowForDate(day, tz, period.DaysOfWeek, period.StartTime, period.EndTime)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		metric, err := calculateWindowMetric(ctx, tx, organisationID, locationID, uuid.Nil, "location", window.Start, window.End, window.End)
		if err != nil {
			return err
		}
		if err := insertServicePeriodMetric(ctx, tx, organisationID, locationID, period.ID, day, window.Start, window.End, metric); err != nil {
			return err
		}
	}
	return nil
}

func calculateWindowMetric(
	ctx context.Context,
	tx pgx.Tx,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	groupID uuid.UUID,
	scope string,
	windowStart time.Time,
	windowEnd time.Time,
	asOf time.Time,
) (Metric, error) {
	filter, args := scopeFilter(scope, groupID)
	query := fmt.Sprintf(`
WITH scoped_tables AS (
    SELECT t.id
    FROM tables t
    WHERE t.organisation_id = $1
      AND t.location_id = $2
      AND t.is_active
      AND t.deleted_at IS NULL
      %s
),
table_count AS (
    SELECT count(*)::integer AS value FROM scoped_tables
),
session_overlap AS (
    SELECT coalesce(sum(greatest(0, extract(epoch FROM least(coalesce(ts.ended_at, $5), $4) - greatest(ts.started_at, $3)))), 0)::double precision AS value
    FROM table_sessions ts
    JOIN scoped_tables st ON st.id = ts.table_id
    WHERE ts.started_at < $4
      AND coalesce(ts.ended_at, $5) > $3
      AND coalesce(ts.ended_at, $5) >= ts.started_at
),
completed_sessions AS (
    SELECT extract(epoch FROM ts.ended_at - ts.started_at)::double precision AS duration
    FROM table_sessions ts
    JOIN scoped_tables st ON st.id = ts.table_id
    WHERE ts.ended_at >= $3
      AND ts.ended_at < $4
      AND ts.ended_at >= ts.started_at
),
assist_metrics AS (
    SELECT count(*)::integer AS request_count,
           coalesce(avg(extract(epoch FROM ar.acknowledged_at - ar.requested_at)) FILTER (
               WHERE ar.acknowledged_at IS NOT NULL AND ar.acknowledged_at >= ar.requested_at
           ), 0)::double precision AS avg_response,
           coalesce(avg(extract(epoch FROM ar.resolved_at - ar.requested_at)) FILTER (
               WHERE ar.resolved_at IS NOT NULL AND ar.resolved_at >= ar.requested_at
           ), 0)::double precision AS avg_resolution
    FROM assist_requests ar
    JOIN scoped_tables st ON st.id = ar.table_id
    WHERE ar.requested_at >= $3
      AND ar.requested_at < $4
),
anomalies AS (
    SELECT count(*)::integer AS value
    FROM table_sessions ts
    JOIN scoped_tables st ON st.id = ts.table_id
    WHERE ts.ended_at IS NOT NULL
      AND ts.ended_at < ts.started_at
)
SELECT table_count.value,
       session_overlap.value,
       table_count.value::double precision * extract(epoch FROM $4::timestamptz - $3::timestamptz),
       count(completed_sessions.duration)::integer,
       coalesce(avg(completed_sessions.duration), 0)::double precision,
       coalesce(percentile_cont(0.5) WITHIN GROUP (ORDER BY completed_sessions.duration), 0)::double precision,
       coalesce(percentile_cont(0.9) WITHIN GROUP (ORDER BY completed_sessions.duration), 0)::double precision,
       assist_metrics.request_count,
       assist_metrics.avg_response,
       assist_metrics.avg_resolution,
       anomalies.value
FROM table_count, session_overlap, assist_metrics, anomalies
LEFT JOIN completed_sessions ON true
GROUP BY table_count.value, session_overlap.value, assist_metrics.request_count, assist_metrics.avg_response, assist_metrics.avg_resolution, anomalies.value
`, filter)
	allArgs := []any{organisationID, locationID, windowStart, windowEnd, asOf}
	allArgs = append(allArgs, args...)
	var metric Metric
	err := tx.QueryRow(ctx, query, allArgs...).Scan(
		&metric.ActiveTableCount,
		&metric.OccupancySeconds,
		&metric.UtilisationBasisSeconds,
		&metric.CompletedSessionCount,
		&metric.AvgSessionSeconds,
		&metric.P50SessionSeconds,
		&metric.P90SessionSeconds,
		&metric.AssistRequestCount,
		&metric.AvgAssistResponseSeconds,
		&metric.AvgAssistResolutionSeconds,
		&metric.AnomalyCount,
	)
	if err != nil {
		return Metric{}, fmt.Errorf("calculating analytics metric: %w", err)
	}
	return FinalizeMetric(metric), nil
}

func scopeFilter(scope string, groupID uuid.UUID) (string, []any) {
	switch scope {
	case "table":
		return "AND t.id = $6", []any{groupID}
	case GroupByZone:
		return "AND t.zone_id = $6", []any{groupID}
	case GroupByFloor:
		return "AND t.floor_id = $6", []any{groupID}
	default:
		return "", nil
	}
}

func projectLocation(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, locationID uuid.UUID, day time.Time, start time.Time, end time.Time) error {
	metric, err := calculateWindowMetric(ctx, tx, organisationID, locationID, uuid.Nil, "location", start, end, end)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO analytics_daily_location_metrics (
    organisation_id, location_id, metric_date, bucket_start, bucket_end, active_table_count,
    occupancy_seconds, utilisation_basis_seconds, utilisation_rate, completed_session_count,
    turnover_rate, avg_session_seconds, p50_session_seconds, p90_session_seconds,
    assist_request_count, avg_assist_response_seconds, avg_assist_resolution_seconds, anomaly_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
`,
		organisationID,
		locationID,
		day,
		start,
		end,
		metric.ActiveTableCount,
		metric.OccupancySeconds,
		metric.UtilisationBasisSeconds,
		metric.UtilisationRate,
		metric.CompletedSessionCount,
		metric.TurnoverRate,
		metric.AvgSessionSeconds,
		metric.P50SessionSeconds,
		metric.P90SessionSeconds,
		metric.AssistRequestCount,
		metric.AvgAssistResponseSeconds,
		metric.AvgAssistResolutionSeconds,
		metric.AnomalyCount,
	)
	if err != nil {
		return fmt.Errorf("inserting daily location metric: %w", err)
	}
	return nil
}

func projectGrouped(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, locationID uuid.UUID, day time.Time, start time.Time, end time.Time, groupBy string) error {
	idColumn := "floor_id"
	tableName := "analytics_daily_floor_metrics"
	if groupBy == GroupByZone {
		idColumn = "zone_id"
		tableName = "analytics_daily_zone_metrics"
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`
SELECT DISTINCT %s
FROM tables
WHERE organisation_id = $1
  AND location_id = $2
  AND is_active
  AND deleted_at IS NULL
ORDER BY %s
`, idColumn, idColumn), organisationID, locationID)
	if err != nil {
		return fmt.Errorf("listing analytics groups: %w", err)
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scanning analytics group: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating analytics groups: %w", err)
	}
	for _, id := range ids {
		metric, err := calculateWindowMetric(ctx, tx, organisationID, locationID, id, groupBy, start, end, end)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, fmt.Sprintf(`
INSERT INTO %s (
    organisation_id, location_id, %s, metric_date, bucket_start, bucket_end, active_table_count,
    occupancy_seconds, utilisation_basis_seconds, utilisation_rate, completed_session_count,
    turnover_rate, avg_session_seconds, p50_session_seconds, p90_session_seconds,
    assist_request_count, avg_assist_response_seconds, avg_assist_resolution_seconds, anomaly_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
`, tableName, idColumn), metricArgs(organisationID, locationID, id, day, start, end, metric)...)
		if err != nil {
			return fmt.Errorf("inserting grouped analytics metric: %w", err)
		}
	}
	return nil
}

func projectTables(ctx context.Context, tx pgx.Tx, organisationID uuid.UUID, locationID uuid.UUID, day time.Time, start time.Time, end time.Time, grain string) error {
	rows, err := tx.Query(ctx, `
SELECT id
FROM tables
WHERE organisation_id = $1
  AND location_id = $2
  AND is_active
  AND deleted_at IS NULL
ORDER BY label
`, organisationID, locationID)
	if err != nil {
		return fmt.Errorf("listing analytics tables: %w", err)
	}
	defer rows.Close()
	tableIDs := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scanning analytics table: %w", err)
		}
		tableIDs = append(tableIDs, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating analytics tables: %w", err)
	}
	for _, tableID := range tableIDs {
		metric, err := calculateWindowMetric(ctx, tx, organisationID, locationID, tableID, "table", start, end, end)
		if err != nil {
			return err
		}
		if grain == "hour" {
			_, err = tx.Exec(ctx, `
INSERT INTO analytics_hourly_table_metrics (
    organisation_id, location_id, table_id, bucket_start, bucket_end, active_table_count,
    occupancy_seconds, utilisation_basis_seconds, utilisation_rate, completed_session_count,
    turnover_rate, avg_session_seconds, p50_session_seconds, p90_session_seconds,
    assist_request_count, avg_assist_response_seconds, avg_assist_resolution_seconds, anomaly_count
) VALUES ($1, $2, $3, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
`, metricArgs(organisationID, locationID, tableID, day, start, end, metric)...)
		} else {
			_, err = tx.Exec(ctx, `
INSERT INTO analytics_daily_table_metrics (
    organisation_id, location_id, table_id, metric_date, bucket_start, bucket_end, active_table_count,
    occupancy_seconds, utilisation_basis_seconds, utilisation_rate, completed_session_count,
    turnover_rate, avg_session_seconds, p50_session_seconds, p90_session_seconds,
    assist_request_count, avg_assist_response_seconds, avg_assist_resolution_seconds, anomaly_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
`, metricArgs(organisationID, locationID, tableID, day, start, end, metric)...)
		}
		if err != nil {
			return fmt.Errorf("inserting table analytics metric: %w", err)
		}
	}
	return nil
}

func insertServicePeriodMetric(
	ctx context.Context,
	tx pgx.Tx,
	organisationID uuid.UUID,
	locationID uuid.UUID,
	periodID uuid.UUID,
	day time.Time,
	start time.Time,
	end time.Time,
	metric Metric,
) error {
	_, err := tx.Exec(ctx, `
INSERT INTO analytics_service_period_metrics (
    organisation_id, location_id, service_period_id, metric_date, bucket_start, bucket_end,
    active_table_count, occupancy_seconds, utilisation_basis_seconds, utilisation_rate,
    completed_session_count, turnover_rate, avg_session_seconds, p50_session_seconds,
    p90_session_seconds, assist_request_count, avg_assist_response_seconds,
    avg_assist_resolution_seconds, anomaly_count
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
`, metricArgs(organisationID, locationID, periodID, day, start, end, metric)...)
	if err != nil {
		return fmt.Errorf("inserting service period analytics metric: %w", err)
	}
	return nil
}

func upsertCheckpoint(ctx context.Context, tx pgx.Tx, event events.Envelope) error {
	lag := time.Since(event.OccurredAt)
	if lag < 0 {
		lag = 0
	}
	_, err := tx.Exec(ctx, `
INSERT INTO analytics_projector_checkpoints (
    projector_name, projector_version, cursor_occurred_at, cursor_event_id, lag_seconds, rebuild_status, updated_at
) VALUES ($1, $2, $3, $4, $5, 'idle', now())
ON CONFLICT (projector_name) DO UPDATE
SET projector_version = EXCLUDED.projector_version,
    cursor_occurred_at = EXCLUDED.cursor_occurred_at,
    cursor_event_id = EXCLUDED.cursor_event_id,
    lag_seconds = EXCLUDED.lag_seconds,
    rebuild_status = 'idle',
    updated_at = now()
`, ProjectorName, ProjectorVersion, event.OccurredAt, event.ID, lag.Seconds())
	if err != nil {
		return fmt.Errorf("upserting analytics checkpoint: %w", err)
	}
	return nil
}

func dataQuality(ctx context.Context, tx pgx.Tx, _ *db.Queries, actor TenantActor) (DataQuality, error) {
	var quality DataQuality
	err := tx.QueryRow(ctx, `
SELECT
    count(*) FILTER (WHERE ended_at IS NULL)::integer,
    count(*) FILTER (WHERE ended_at IS NULL AND started_at < now() - interval '6 hours')::integer,
    count(*) FILTER (WHERE ended_at IS NOT NULL AND ended_at < started_at)::integer
FROM table_sessions
WHERE organisation_id = $1 AND location_id = $2
`, actor.OrganisationID, actor.LocationID).Scan(
		&quality.OpenSessionCount,
		&quality.LongOpenSessionCount,
		&quality.ImpossibleSessionCount,
	)
	if err != nil {
		return DataQuality{}, fmt.Errorf("querying session data quality: %w", err)
	}
	err = tx.QueryRow(ctx, `
SELECT count(*)::integer
FROM tables t
LEFT JOIN table_occupancy o ON o.table_id = t.id
WHERE t.organisation_id = $1
  AND t.location_id = $2
  AND t.is_active
  AND t.deleted_at IS NULL
  AND o.table_id IS NULL
`, actor.OrganisationID, actor.LocationID).Scan(&quality.MissingOccupancyCount)
	if err != nil {
		return DataQuality{}, fmt.Errorf("querying occupancy data quality: %w", err)
	}
	err = tx.QueryRow(ctx, `
SELECT count(*)::integer
FROM devices
WHERE organisation_id = $1
  AND location_id = $2
  AND trust_state = 'trusted'
  AND revoked_at IS NULL
  AND (
      last_heartbeat_at IS NULL
      OR last_heartbeat_at < now() - (heartbeat_interval_seconds::bigint * 2 * interval '1 second')
  )
`, actor.OrganisationID, actor.LocationID).Scan(&quality.StaleDeviceCount)
	if err != nil {
		return DataQuality{}, fmt.Errorf("querying device data quality: %w", err)
	}
	return quality, nil
}

func currentServicePeriodMetric(
	ctx context.Context,
	tx pgx.Tx,
	q *db.Queries,
	actor TenantActor,
	location *time.Location,
	date time.Time,
	asOf time.Time,
) (*WindowMetric, error) {
	periods, err := q.ListActiveServicePeriodsByLocation(ctx, db.ListActiveServicePeriodsByLocationParams{
		OrganisationID: actor.OrganisationID,
		LocationID:     actor.LocationID,
	})
	if err != nil {
		return nil, fmt.Errorf("listing active service periods: %w", err)
	}
	for _, period := range periods {
		window, ok, err := ServicePeriodWindowForDate(date, location, period.DaysOfWeek, period.StartTime, period.EndTime)
		if err != nil {
			return nil, err
		}
		if !ok || asOf.Before(window.Start) || !asOf.Before(window.End) {
			continue
		}
		metric, err := calculateWindowMetric(ctx, tx, actor.OrganisationID, actor.LocationID, uuid.Nil, "location", window.Start, window.End, asOf)
		if err != nil {
			return nil, err
		}
		return &WindowMetric{
			ID:          period.ID,
			Name:        period.Name,
			WindowStart: window.Start,
			WindowEnd:   window.End,
			Metric:      metric,
		}, nil
	}
	return nil, nil
}

func scanTimePoint(rows pgx.Rows) (TimePoint, error) {
	var point TimePoint
	var avgSession *float64
	var p50 *float64
	var p90 *float64
	var avgResponse *float64
	var avgResolution *float64
	err := rows.Scan(
		&point.BucketStart,
		&point.BucketEnd,
		&point.ActiveTableCount,
		&point.OccupancySeconds,
		&point.UtilisationBasisSeconds,
		&point.UtilisationRate,
		&point.CompletedSessionCount,
		&point.TurnoverRate,
		&avgSession,
		&p50,
		&p90,
		&point.AssistRequestCount,
		&avgResponse,
		&avgResolution,
		&point.AnomalyCount,
	)
	if err != nil {
		return TimePoint{}, fmt.Errorf("scanning analytics time point: %w", err)
	}
	point.AvgSessionSeconds = optionalFloat(avgSession)
	point.P50SessionSeconds = optionalFloat(p50)
	point.P90SessionSeconds = optionalFloat(p90)
	point.AvgAssistResponseSeconds = optionalFloat(avgResponse)
	point.AvgAssistResolutionSeconds = optionalFloat(avgResolution)
	return point, nil
}

func metricArgs(organisationID uuid.UUID, locationID uuid.UUID, groupID uuid.UUID, day time.Time, start time.Time, end time.Time, metric Metric) []any {
	return []any{
		organisationID,
		locationID,
		groupID,
		day,
		start,
		end,
		metric.ActiveTableCount,
		metric.OccupancySeconds,
		metric.UtilisationBasisSeconds,
		metric.UtilisationRate,
		metric.CompletedSessionCount,
		metric.TurnoverRate,
		metric.AvgSessionSeconds,
		metric.P50SessionSeconds,
		metric.P90SessionSeconds,
		metric.AssistRequestCount,
		metric.AvgAssistResponseSeconds,
		metric.AvgAssistResolutionSeconds,
		metric.AnomalyCount,
	}
}

func optionalFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
