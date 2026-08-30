package analytics

import (
	"testing"
	"time"
)

func TestServicePeriodWindowForDate(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("Africa/Johannesburg")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	start, err := ParseClock("18:00")
	if err != nil {
		t.Fatalf("parse start: %v", err)
	}
	end, err := ParseClock("22:30")
	if err != nil {
		t.Fatalf("parse end: %v", err)
	}
	window, ok, err := ServicePeriodWindowForDate(
		time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC),
		location,
		[]int16{1},
		start,
		end,
	)
	if err != nil {
		t.Fatalf("window: %v", err)
	}
	if !ok {
		t.Fatalf("window not active")
	}
	if got := window.Start.In(location).Format("2006-01-02 15:04"); got != "2026-08-31 18:00" {
		t.Fatalf("start = %s", got)
	}
	if got := window.End.In(location).Format("2006-01-02 15:04"); got != "2026-08-31 22:30" {
		t.Fatalf("end = %s", got)
	}
}

func TestServicePeriodWindowForDateOvernight(t *testing.T) {
	t.Parallel()

	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	start, err := ParseClock("22:00")
	if err != nil {
		t.Fatalf("parse start: %v", err)
	}
	end, err := ParseClock("02:00")
	if err != nil {
		t.Fatalf("parse end: %v", err)
	}
	window, ok, err := ServicePeriodWindowForDate(
		time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC),
		location,
		[]int16{0},
		start,
		end,
	)
	if err != nil {
		t.Fatalf("window: %v", err)
	}
	if !ok {
		t.Fatalf("window not active")
	}
	if !window.End.After(window.Start) {
		t.Fatalf("overnight end did not advance")
	}
	if got := window.Start.In(location).Format("2006-01-02 15:04"); got != "2026-11-01 22:00" {
		t.Fatalf("start = %s", got)
	}
	if got := window.End.In(location).Format("2006-01-02 15:04"); got != "2026-11-02 02:00" {
		t.Fatalf("end = %s", got)
	}
}
