package analytics

import (
	"testing"
	"time"
)

func TestOverlapSeconds(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)
	windowStart := base
	windowEnd := base.Add(2 * time.Hour)
	sessionEnd := base.Add(90 * time.Minute)

	tests := []struct {
		name     string
		start    time.Time
		end      *time.Time
		asOf     time.Time
		expected float64
		valid    bool
	}{
		{
			name:     "completed overlap",
			start:    base.Add(-30 * time.Minute),
			end:      &sessionEnd,
			asOf:     windowEnd,
			expected: 5400,
			valid:    true,
		},
		{
			name:     "open session capped by as of",
			start:    base.Add(30 * time.Minute),
			asOf:     base.Add(time.Hour),
			expected: 1800,
			valid:    true,
		},
		{
			name:     "outside window",
			start:    base.Add(3 * time.Hour),
			asOf:     base.Add(4 * time.Hour),
			expected: 0,
			valid:    true,
		},
		{
			name:     "negative session",
			start:    sessionEnd,
			end:      ptr(base),
			asOf:     windowEnd,
			expected: 0,
			valid:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, valid := OverlapSeconds(tt.start, tt.end, windowStart, windowEnd, tt.asOf)
			if got != tt.expected || valid != tt.valid {
				t.Fatalf("OverlapSeconds() = %v, %v; want %v, %v", got, valid, tt.expected, tt.valid)
			}
		})
	}
}

func TestPercentile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		values     []float64
		percentile float64
		expected   float64
	}{
		{name: "empty", values: nil, percentile: 0.5, expected: 0},
		{name: "single", values: []float64{42}, percentile: 0.9, expected: 42},
		{name: "median continuous", values: []float64{10, 20, 30, 40}, percentile: 0.5, expected: 25},
		{name: "p90 continuous", values: []float64{10, 20, 30, 40}, percentile: 0.9, expected: 37},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Percentile(tt.values, tt.percentile); got != tt.expected {
				t.Fatalf("Percentile() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func ptr(value time.Time) *time.Time {
	return &value
}
