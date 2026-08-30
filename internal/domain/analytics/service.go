package analytics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/seatd/seatd/internal/domain/events"
	"github.com/seatd/seatd/internal/domain/identity"
	"github.com/seatd/seatd/internal/store/db"
)

const (
	ProjectorName    = "analytics-projector"
	ProjectorVersion = int32(1)

	GrainHour = "hour"
	GrainDay  = "day"

	GroupByZone          = "zone"
	GroupByFloor         = "floor"
	GroupByServicePeriod = "service_period"
)

var (
	ErrForbidden  = errors.New("forbidden")
	ErrValidation = errors.New("validation failed")
	ErrNotFound   = errors.New("not found")
)

type TenantActor struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	ActorRef       string
}

type WindowMetric struct {
	ID          uuid.UUID
	Name        string
	WindowStart time.Time
	WindowEnd   time.Time
	Metric
}

type Summary struct {
	Location             db.Location
	Date                 time.Time
	Today                WindowMetric
	CurrentServicePeriod *WindowMetric
	Checkpoint           *db.AnalyticsProjectorCheckpoint
	DataQuality          DataQuality
}

type TimePoint struct {
	BucketStart time.Time
	BucketEnd   time.Time
	Metric
}

type ComparisonPoint struct {
	GroupID     uuid.UUID
	GroupName   string
	MetricDate  time.Time
	WindowStart time.Time
	WindowEnd   time.Time
	Metric
}

type DataQuality struct {
	OpenSessionCount       int32
	LongOpenSessionCount   int32
	ImpossibleSessionCount int32
	MissingOccupancyCount  int32
	StaleDeviceCount       int32
	ProjectorLagSeconds    float64
	RebuildStatus          string
}

type ServicePeriodInput struct {
	Name            string
	DaysOfWeek      []int16
	StartTime       string
	EndTime         string
	ExpectedVersion int32
}

type RebuildParams struct {
	TenantActor
	LocationID uuid.UUID
	From       time.Time
	To         time.Time
}

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func (s *Service) Summary(ctx context.Context, actor TenantActor, locationID uuid.UUID, date time.Time, asOf time.Time) (Summary, error) {
	if err := actor.validate(); err != nil {
		return Summary{}, err
	}
	actor.LocationID = locationID
	var result Summary
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireRead(ctx, q, actor); err != nil {
			return err
		}
		location, err := q.GetLocation(ctx, db.GetLocationParams{ID: locationID, OrganisationID: actor.OrganisationID})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("getting analytics location: %w", err)
		}
		tz, err := time.LoadLocation(location.Timezone)
		if err != nil {
			return fmt.Errorf("loading location timezone: %w", err)
		}
		localDate := calendarDateInLocation(date, tz)
		start, end := localCalendarDateWindow(localDate, tz)
		metric, err := calculateWindowMetric(ctx, tx, actor.OrganisationID, locationID, uuid.Nil, "location", start, end, minTime(asOf, end))
		if err != nil {
			return err
		}
		result = Summary{
			Location: location,
			Date:     localDate,
			Today: WindowMetric{
				WindowStart: start,
				WindowEnd:   end,
				Metric:      metric,
			},
		}
		current, err := currentServicePeriodMetric(ctx, tx, q, actor, tz, localDate, asOf)
		if err != nil {
			return err
		}
		result.CurrentServicePeriod = current
		quality, err := dataQuality(ctx, tx, q, actor)
		if err != nil {
			return err
		}
		result.DataQuality = quality
		checkpoint, err := q.GetAnalyticsProjectorCheckpoint(ctx, ProjectorName)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("getting analytics checkpoint: %w", err)
		}
		if err == nil {
			result.Checkpoint = &checkpoint
			result.DataQuality.ProjectorLagSeconds = checkpoint.LagSeconds
			result.DataQuality.RebuildStatus = checkpoint.RebuildStatus
		}
		return nil
	})
	return result, err
}

func (s *Service) Timeseries(ctx context.Context, actor TenantActor, from time.Time, to time.Time, grain string) ([]TimePoint, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	if !from.Before(to) || (grain != GrainHour && grain != GrainDay) {
		return nil, ErrValidation
	}
	points := []TimePoint{}
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireRead(ctx, q, actor); err != nil {
			return err
		}
		if grain == GrainDay {
			rows, err := q.ListDailyLocationMetrics(ctx, db.ListDailyLocationMetricsParams{
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				MetricDate:     pgDate(from),
				MetricDate_2:   pgDate(to),
			})
			if err != nil {
				return fmt.Errorf("listing daily location metrics: %w", err)
			}
			for _, row := range rows {
				points = append(points, TimePoint{BucketStart: row.BucketStart.Time, BucketEnd: row.BucketEnd.Time, Metric: metricFromLocation(row)})
			}
			return nil
		}
		rows, err := tx.Query(ctx, `
SELECT bucket_start,
       bucket_end,
       count(DISTINCT table_id)::integer AS active_table_count,
       sum(occupancy_seconds),
       sum(utilisation_basis_seconds),
       CASE WHEN sum(utilisation_basis_seconds) > 0 THEN sum(occupancy_seconds) / sum(utilisation_basis_seconds) ELSE 0 END,
       sum(completed_session_count)::integer,
       CASE WHEN count(DISTINCT table_id) > 0 THEN sum(completed_session_count)::double precision / count(DISTINCT table_id) ELSE 0 END,
       avg(NULLIF(avg_session_seconds, 0)),
       percentile_cont(0.5) WITHIN GROUP (ORDER BY NULLIF(p50_session_seconds, 0)),
       percentile_cont(0.9) WITHIN GROUP (ORDER BY NULLIF(p90_session_seconds, 0)),
       sum(assist_request_count)::integer,
       avg(NULLIF(avg_assist_response_seconds, 0)),
       avg(NULLIF(avg_assist_resolution_seconds, 0)),
       sum(anomaly_count)::integer
FROM analytics_hourly_table_metrics
WHERE organisation_id = $1 AND location_id = $2 AND bucket_start >= $3 AND bucket_start < $4
GROUP BY bucket_start, bucket_end
ORDER BY bucket_start
`, actor.OrganisationID, actor.LocationID, from, to)
		if err != nil {
			return fmt.Errorf("listing hourly metrics: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			point, err := scanTimePoint(rows)
			if err != nil {
				return err
			}
			points = append(points, point)
		}
		return rows.Err()
	})
	return points, err
}

func (s *Service) Comparison(ctx context.Context, actor TenantActor, from time.Time, to time.Time, groupBy string) ([]ComparisonPoint, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	if !from.Before(to) {
		return nil, ErrValidation
	}
	points := []ComparisonPoint{}
	err := s.inTenantTx(ctx, actor, func(_ pgx.Tx, q *db.Queries) error {
		if err := requireRead(ctx, q, actor); err != nil {
			return err
		}
		switch groupBy {
		case GroupByZone:
			rows, err := q.ListDailyZoneMetrics(ctx, db.ListDailyZoneMetricsParams{
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				MetricDate:     pgDate(from),
				MetricDate_2:   pgDate(to),
			})
			if err != nil {
				return fmt.Errorf("listing zone metrics: %w", err)
			}
			for _, row := range rows {
				points = append(points, ComparisonPoint{GroupID: row.ZoneID, GroupName: row.GroupName, MetricDate: row.MetricDate.Time, WindowStart: row.BucketStart.Time, WindowEnd: row.BucketEnd.Time, Metric: metricFromZone(row)})
			}
		case GroupByFloor:
			rows, err := q.ListDailyFloorMetrics(ctx, db.ListDailyFloorMetricsParams{
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				MetricDate:     pgDate(from),
				MetricDate_2:   pgDate(to),
			})
			if err != nil {
				return fmt.Errorf("listing floor metrics: %w", err)
			}
			for _, row := range rows {
				points = append(points, ComparisonPoint{GroupID: row.FloorID, GroupName: row.GroupName, MetricDate: row.MetricDate.Time, WindowStart: row.BucketStart.Time, WindowEnd: row.BucketEnd.Time, Metric: metricFromFloor(row)})
			}
		case GroupByServicePeriod:
			rows, err := q.ListServicePeriodMetrics(ctx, db.ListServicePeriodMetricsParams{
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				MetricDate:     pgDate(from),
				MetricDate_2:   pgDate(to),
			})
			if err != nil {
				return fmt.Errorf("listing service period metrics: %w", err)
			}
			for _, row := range rows {
				points = append(points, ComparisonPoint{GroupID: row.ServicePeriodID, GroupName: row.GroupName, MetricDate: row.MetricDate.Time, WindowStart: row.BucketStart.Time, WindowEnd: row.BucketEnd.Time, Metric: metricFromServicePeriod(row)})
			}
		default:
			return ErrValidation
		}
		return nil
	})
	return points, err
}

func (s *Service) DataQuality(ctx context.Context, actor TenantActor) (DataQuality, error) {
	if err := actor.validate(); err != nil {
		return DataQuality{}, err
	}
	var result DataQuality
	err := s.inTenantTx(ctx, actor, func(tx pgx.Tx, q *db.Queries) error {
		if err := requireRead(ctx, q, actor); err != nil {
			return err
		}
		quality, err := dataQuality(ctx, tx, q, actor)
		if err != nil {
			return err
		}
		result = quality
		checkpoint, err := q.GetAnalyticsProjectorCheckpoint(ctx, ProjectorName)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("getting analytics checkpoint: %w", err)
		}
		if err == nil {
			result.ProjectorLagSeconds = checkpoint.LagSeconds
			result.RebuildStatus = checkpoint.RebuildStatus
		}
		return nil
	})
	return result, err
}

func (s *Service) ListServicePeriods(ctx context.Context, actor TenantActor) ([]db.ServicePeriod, error) {
	if err := actor.validate(); err != nil {
		return nil, err
	}
	var periods []db.ServicePeriod
	err := s.inTenantTx(ctx, actor, func(_ pgx.Tx, q *db.Queries) error {
		if err := requireRead(ctx, q, actor); err != nil {
			return err
		}
		var err error
		periods, err = q.ListServicePeriodsByLocation(ctx, db.ListServicePeriodsByLocationParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
		})
		if err != nil {
			return fmt.Errorf("listing service periods: %w", err)
		}
		return nil
	})
	return periods, err
}

func (s *Service) CreateServicePeriod(ctx context.Context, actor TenantActor, input ServicePeriodInput) (db.ServicePeriod, error) {
	if err := actor.validate(); err != nil {
		return db.ServicePeriod{}, err
	}
	name, days, start, end, err := validateServicePeriodInput(input, false)
	if err != nil {
		return db.ServicePeriod{}, err
	}
	var period db.ServicePeriod
	err = s.inTenantTx(ctx, actor, func(_ pgx.Tx, q *db.Queries) error {
		allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			MemberRef:      actor.ActorRef,
			PermissionName: identity.PermissionLocationManage,
		})
		if err != nil {
			return fmt.Errorf("checking location permission: %w", err)
		}
		if !allowed {
			return ErrForbidden
		}
		period, err = q.CreateServicePeriod(ctx, db.CreateServicePeriodParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			Name:           name,
			DaysOfWeek:     days,
			StartTime:      start,
			EndTime:        end,
		})
		if err != nil {
			return fmt.Errorf("creating service period: %w", err)
		}
		return nil
	})
	return period, err
}

func (s *Service) UpdateServicePeriod(ctx context.Context, actor TenantActor, id uuid.UUID, input ServicePeriodInput) (db.ServicePeriod, error) {
	if err := actor.validate(); err != nil || id == uuid.Nil {
		return db.ServicePeriod{}, ErrValidation
	}
	name, days, start, end, err := validateServicePeriodInput(input, true)
	if err != nil {
		return db.ServicePeriod{}, err
	}
	var period db.ServicePeriod
	err = s.inTenantTx(ctx, actor, func(_ pgx.Tx, q *db.Queries) error {
		allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			MemberRef:      actor.ActorRef,
			PermissionName: identity.PermissionLocationManage,
		})
		if err != nil {
			return fmt.Errorf("checking location permission: %w", err)
		}
		if !allowed {
			return ErrForbidden
		}
		period, err = q.UpdateServicePeriod(ctx, db.UpdateServicePeriodParams{
			ID:             id,
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			Name:           name,
			DaysOfWeek:     days,
			StartTime:      start,
			EndTime:        end,
			Version:        input.ExpectedVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("updating service period: %w", err)
		}
		return nil
	})
	return period, err
}

func (s *Service) ArchiveServicePeriod(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32) (db.ServicePeriod, error) {
	if err := actor.validate(); err != nil || id == uuid.Nil || expectedVersion < 1 {
		return db.ServicePeriod{}, ErrValidation
	}
	var period db.ServicePeriod
	err := s.inTenantTx(ctx, actor, func(_ pgx.Tx, q *db.Queries) error {
		allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			MemberRef:      actor.ActorRef,
			PermissionName: identity.PermissionLocationManage,
		})
		if err != nil {
			return fmt.Errorf("checking location permission: %w", err)
		}
		if !allowed {
			return ErrForbidden
		}
		period, err = q.ArchiveServicePeriod(ctx, db.ArchiveServicePeriodParams{
			ID:             id,
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			Version:        expectedVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("archiving service period: %w", err)
		}
		return nil
	})
	return period, err
}

func (s *Service) Rebuild(ctx context.Context, arg RebuildParams) error {
	if err := arg.TenantActor.validate(); err != nil {
		return err
	}
	if arg.LocationID != arg.TenantActor.LocationID || !arg.From.Before(arg.To) {
		return ErrValidation
	}
	return s.inTenantTx(ctx, arg.TenantActor, func(tx pgx.Tx, q *db.Queries) error {
		allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			MemberRef:      arg.ActorRef,
			PermissionName: identity.PermissionLocationManage,
		})
		if err != nil {
			return fmt.Errorf("checking location permission: %w", err)
		}
		if !allowed {
			return ErrForbidden
		}
		return rebuildRange(ctx, tx, q, arg.OrganisationID, arg.LocationID, arg.From, arg.To)
	})
}

func (actor TenantActor) validate() error {
	if actor.OrganisationID == uuid.Nil || actor.LocationID == uuid.Nil || strings.TrimSpace(actor.ActorRef) == "" {
		return ErrValidation
	}
	return nil
}

func requireRead(ctx context.Context, q *db.Queries, actor TenantActor) error {
	allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
		OrganisationID: actor.OrganisationID,
		LocationID:     actor.LocationID,
		MemberRef:      actor.ActorRef,
		PermissionName: identity.PermissionAnalyticsRead,
	})
	if err != nil {
		return fmt.Errorf("checking analytics permission: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) inTenantTx(ctx context.Context, actor TenantActor, fn func(pgx.Tx, *db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, actor.OrganisationID.String(), actor.LocationID.String()); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	if err := fn(tx, db.New(tx)); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func validateServicePeriodInput(input ServicePeriodInput, requireVersion bool) (string, []int16, pgtype.Time, pgtype.Time, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(input.DaysOfWeek) == 0 || len(input.DaysOfWeek) > 7 {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	if requireVersion && input.ExpectedVersion < 1 {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	seen := map[int16]bool{}
	days := make([]int16, 0, len(input.DaysOfWeek))
	for _, day := range input.DaysOfWeek {
		if day < 0 || day > 6 || seen[day] {
			return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
		}
		seen[day] = true
		days = append(days, day)
	}
	start, err := ParseClock(input.StartTime)
	if err != nil {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	end, err := ParseClock(input.EndTime)
	if err != nil {
		return "", nil, pgtype.Time{}, pgtype.Time{}, ErrValidation
	}
	return name, days, start, end, nil
}

func localDateWindow(date time.Time, location *time.Location) (time.Time, time.Time) {
	local := date.In(location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return start.UTC(), start.AddDate(0, 0, 1).UTC()
}

func localCalendarDateWindow(date time.Time, location *time.Location) (time.Time, time.Time) {
	local := calendarDateInLocation(date, location)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, location)
	return start.UTC(), start.AddDate(0, 0, 1).UTC()
}

func calendarDateInLocation(date time.Time, location *time.Location) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, location)
}

func pgDate(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}

func metricFromLocation(row db.AnalyticsDailyLocationMetric) Metric {
	return Metric{
		ActiveTableCount:           row.ActiveTableCount,
		OccupancySeconds:           row.OccupancySeconds,
		UtilisationBasisSeconds:    row.UtilisationBasisSeconds,
		UtilisationRate:            row.UtilisationRate,
		CompletedSessionCount:      row.CompletedSessionCount,
		TurnoverRate:               row.TurnoverRate,
		AvgSessionSeconds:          row.AvgSessionSeconds,
		P50SessionSeconds:          row.P50SessionSeconds,
		P90SessionSeconds:          row.P90SessionSeconds,
		AssistRequestCount:         row.AssistRequestCount,
		AvgAssistResponseSeconds:   row.AvgAssistResponseSeconds,
		AvgAssistResolutionSeconds: row.AvgAssistResolutionSeconds,
		AnomalyCount:               row.AnomalyCount,
	}
}

func metricFromZone(row db.ListDailyZoneMetricsRow) Metric {
	return metricFromValues(
		row.ActiveTableCount,
		row.OccupancySeconds,
		row.UtilisationBasisSeconds,
		row.UtilisationRate,
		row.CompletedSessionCount,
		row.TurnoverRate,
		row.AvgSessionSeconds,
		row.P50SessionSeconds,
		row.P90SessionSeconds,
		row.AssistRequestCount,
		row.AvgAssistResponseSeconds,
		row.AvgAssistResolutionSeconds,
		row.AnomalyCount,
	)
}

func metricFromFloor(row db.ListDailyFloorMetricsRow) Metric {
	return metricFromValues(
		row.ActiveTableCount,
		row.OccupancySeconds,
		row.UtilisationBasisSeconds,
		row.UtilisationRate,
		row.CompletedSessionCount,
		row.TurnoverRate,
		row.AvgSessionSeconds,
		row.P50SessionSeconds,
		row.P90SessionSeconds,
		row.AssistRequestCount,
		row.AvgAssistResponseSeconds,
		row.AvgAssistResolutionSeconds,
		row.AnomalyCount,
	)
}

func metricFromServicePeriod(row db.ListServicePeriodMetricsRow) Metric {
	return metricFromValues(
		row.ActiveTableCount,
		row.OccupancySeconds,
		row.UtilisationBasisSeconds,
		row.UtilisationRate,
		row.CompletedSessionCount,
		row.TurnoverRate,
		row.AvgSessionSeconds,
		row.P50SessionSeconds,
		row.P90SessionSeconds,
		row.AssistRequestCount,
		row.AvgAssistResponseSeconds,
		row.AvgAssistResolutionSeconds,
		row.AnomalyCount,
	)
}

func metricFromValues(activeTableCount int32, occupancySeconds, basisSeconds, utilisationRate float64, completedSessionCount int32, turnoverRate, avgSessionSeconds, p50SessionSeconds, p90SessionSeconds float64, assistRequestCount int32, avgAssistResponseSeconds, avgAssistResolutionSeconds float64, anomalyCount int32) Metric {
	return Metric{
		ActiveTableCount:           activeTableCount,
		OccupancySeconds:           occupancySeconds,
		UtilisationBasisSeconds:    basisSeconds,
		UtilisationRate:            utilisationRate,
		CompletedSessionCount:      completedSessionCount,
		TurnoverRate:               turnoverRate,
		AvgSessionSeconds:          avgSessionSeconds,
		P50SessionSeconds:          p50SessionSeconds,
		P90SessionSeconds:          p90SessionSeconds,
		AssistRequestCount:         assistRequestCount,
		AvgAssistResponseSeconds:   avgAssistResponseSeconds,
		AvgAssistResolutionSeconds: avgAssistResolutionSeconds,
		AnomalyCount:               anomalyCount,
	}
}

func _eventTypesUsedByProjector() []string {
	return []string{
		events.TypeTableOccupied,
		events.TypeTableCleared,
		events.TypeAssistRequested,
		events.TypeAssistAcknowledged,
		events.TypeAssistResolved,
		events.TypeAssistCancelled,
	}
}
