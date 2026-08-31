package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHourlyMetricArgsOmitsMetricDate(t *testing.T) {
	t.Parallel()

	organisationID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	locationID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	tableID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	day := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	start := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	metric := Metric{ActiveTableCount: 1, OccupancySeconds: 120}

	daily := metricArgs(organisationID, locationID, tableID, day, start, end, metric)
	hourly := hourlyMetricArgs(organisationID, locationID, tableID, start, end, metric)

	if got, want := len(daily), 19; got != want {
		t.Fatalf("len(metricArgs) = %d, want %d", got, want)
	}
	if got, want := len(hourly), 18; got != want {
		t.Fatalf("len(hourlyMetricArgs) = %d, want %d", got, want)
	}
	if daily[3] != day {
		t.Fatalf("metricArgs[3] = %v, want metric date %v", daily[3], day)
	}
	if hourly[3] != start {
		t.Fatalf("hourlyMetricArgs[3] = %v, want bucket start %v", hourly[3], start)
	}
}
