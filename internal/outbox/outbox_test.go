package outbox

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/kadebhug/seatd_v2/internal/domain/events"
)

func TestWorkerBackoff(t *testing.T) {
	t.Parallel()

	worker := NewWorker(nil, slog.Default(), nil, Config{
		BackoffBase: time.Second,
		BackoffMax:  10 * time.Second,
	})

	tests := []struct {
		name    string
		attempt int32
		want    time.Duration
	}{
		{name: "first attempt uses base", attempt: 1, want: time.Second},
		{name: "second attempt doubles", attempt: 2, want: 2 * time.Second},
		{name: "large attempt caps", attempt: 12, want: 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := worker.backoff(tt.attempt)
			if got != tt.want {
				t.Fatalf("backoff(%d) = %s, want %s", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestConsumerFunc(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("consumer failed")
	consumer := ConsumerFunc{
		ConsumerName: "analytics-projector",
		Fn: func(context.Context, events.Envelope) error {
			return wantErr
		},
	}

	if consumer.Name() != "analytics-projector" {
		t.Fatalf("name = %q, want analytics-projector", consumer.Name())
	}
	if err := consumer.Handle(context.Background(), events.Envelope{}); !errors.Is(err, wantErr) {
		t.Fatalf("handle error = %v, want %v", err, wantErr)
	}
}

func TestWorkerTracksOwnedDestinations(t *testing.T) {
	t.Parallel()

	worker := NewWorker(nil, slog.Default(), map[string][]Consumer{
		"realtime.operations": {
			ConsumerFunc{ConsumerName: "realtime-gateway"},
		},
		"analytics.operations": {
			ConsumerFunc{ConsumerName: "analytics-projector"},
		},
	}, Config{})

	got := map[string]bool{}
	for _, destination := range worker.destinations {
		got[destination] = true
	}
	for _, want := range []string{"realtime.operations", "analytics.operations"} {
		if !got[want] {
			t.Fatalf("worker destinations = %v, missing %q", worker.destinations, want)
		}
	}
}

func TestWorkerRecordsClaimedMetric(t *testing.T) {
	t.Parallel()

	metrics := &fakeMetricsRecorder{}
	worker := NewWorker(nil, slog.Default(), nil, Config{Metrics: metrics})
	worker.recordClaimed(Record{Destination: "analytics.operations", Topic: "table.occupied"})

	if metrics.claimedDestination != "analytics.operations" || metrics.claimedTopic != "table.occupied" {
		t.Fatalf("claimed metric = %q/%q, want analytics.operations/table.occupied", metrics.claimedDestination, metrics.claimedTopic)
	}
}

type fakeMetricsRecorder struct {
	claimedDestination string
	claimedTopic       string
}

func (r *fakeMetricsRecorder) RecordClaimed(destination, topic string) {
	r.claimedDestination = destination
	r.claimedTopic = topic
}

func (r *fakeMetricsRecorder) RecordProcessed(string, string) {}
func (r *fakeMetricsRecorder) RecordRetried(string, string)   {}
func (r *fakeMetricsRecorder) RecordFailed(string, string)    {}
func (r *fakeMetricsRecorder) RecordConsumerFailed(string)    {}
