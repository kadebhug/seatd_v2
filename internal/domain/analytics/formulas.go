package analytics

import (
	"math"
	"sort"
	"time"
)

type Metric struct {
	ActiveTableCount           int32
	OccupancySeconds           float64
	UtilisationBasisSeconds    float64
	UtilisationRate            float64
	CompletedSessionCount      int32
	TurnoverRate               float64
	AvgSessionSeconds          float64
	P50SessionSeconds          float64
	P90SessionSeconds          float64
	AssistRequestCount         int32
	AvgAssistResponseSeconds   float64
	AvgAssistResolutionSeconds float64
	AnomalyCount               int32
}

func FinalizeMetric(metric Metric) Metric {
	if metric.UtilisationBasisSeconds > 0 {
		metric.UtilisationRate = metric.OccupancySeconds / metric.UtilisationBasisSeconds
	}
	if metric.ActiveTableCount > 0 {
		metric.TurnoverRate = float64(metric.CompletedSessionCount) / float64(metric.ActiveTableCount)
	}
	return metric
}

func OverlapSeconds(start time.Time, end *time.Time, windowStart time.Time, windowEnd time.Time, asOf time.Time) (float64, bool) {
	sessionEnd := asOf
	if end != nil {
		sessionEnd = *end
	}
	if sessionEnd.Before(start) {
		return 0, false
	}
	overlapStart := maxTime(start, windowStart)
	overlapEnd := minTime(sessionEnd, windowEnd)
	if !overlapEnd.After(overlapStart) {
		return 0, true
	}
	return overlapEnd.Sub(overlapStart).Seconds(), true
}

func Percentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := percentile * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	weight := position - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func maxTime(a time.Time, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minTime(a time.Time, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
