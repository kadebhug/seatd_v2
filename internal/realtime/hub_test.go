package realtime

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kadebhug/seatd_v2/internal/domain/events"
)

func TestMessageFromEvent(t *testing.T) {
	t.Parallel()

	version := int32(7)
	event := events.Envelope{
		ID:             uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		Type:           events.TypeTableOccupied,
		OrganisationID: uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		LocationID:     uuid.MustParse("33333333-3333-3333-3333-333333333333"),
		EntityType:     "table",
		EntityID:       uuid.MustParse("44444444-4444-4444-4444-444444444444"),
		EntityVersion:  &version,
		OccurredAt:     time.Date(2026, 8, 29, 10, 30, 0, 0, time.UTC),
		Data:           json.RawMessage(`{"status":"occupied"}`),
	}

	got := MessageFromEvent(event)
	if got.Type != events.TypeTableOccupied {
		t.Fatalf("type = %q, want %q", got.Type, events.TypeTableOccupied)
	}
	if got.EventID != event.ID.String() {
		t.Fatalf("event ID = %q, want %q", got.EventID, event.ID)
	}
	if got.Version == nil || *got.Version != version {
		t.Fatalf("version = %v, want %d", got.Version, version)
	}
	if string(got.Data) != `{"status":"occupied"}` {
		t.Fatalf("data = %s", got.Data)
	}
}

func TestHubPublishScopesByLocation(t *testing.T) {
	t.Parallel()

	hub := NewHub(nil)
	orgID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	locationID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherLocationID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	client := hub.Subscribe(Scope{OrganisationID: orgID, LocationID: locationID}, "user:waiter", uuid.New(), 1)
	other := hub.Subscribe(Scope{OrganisationID: orgID, LocationID: otherLocationID}, "user:waiter", uuid.New(), 1)
	defer hub.Unsubscribe(client)
	defer hub.Unsubscribe(other)

	hub.Publish(context.Background(), events.Envelope{
		ID:             uuid.New(),
		Type:           events.TypeTableCleared,
		OrganisationID: orgID,
		LocationID:     locationID,
		EntityType:     "table",
		EntityID:       uuid.New(),
		OccurredAt:     time.Now(),
		Data:           json.RawMessage(`{}`),
	})

	select {
	case <-client.send:
	default:
		t.Fatalf("scoped client did not receive message")
	}
	select {
	case <-other.send:
		t.Fatalf("client for another location received message")
	default:
	}
}

func TestHubBackpressureClosesSlowClient(t *testing.T) {
	t.Parallel()

	hub := NewHub(nil)
	scope := Scope{OrganisationID: uuid.New(), LocationID: uuid.New()}
	client := hub.Subscribe(scope, "user:waiter", uuid.New(), 1)

	event := events.Envelope{
		ID:             uuid.New(),
		Type:           events.TypeAssistRequested,
		OrganisationID: scope.OrganisationID,
		LocationID:     scope.LocationID,
		EntityType:     "assist",
		EntityID:       uuid.New(),
		OccurredAt:     time.Now(),
		Data:           json.RawMessage(`{}`),
	}
	hub.Publish(context.Background(), event)
	hub.Publish(context.Background(), event)

	select {
	case <-client.Done():
	default:
		t.Fatalf("slow client was not closed")
	}
	metrics := hub.MetricsSnapshot()
	if metrics.BackpressureCloses != 1 {
		t.Fatalf("backpressure closes = %d, want 1", metrics.BackpressureCloses)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	t.Parallel()

	want := Cursor{
		OccurredAt: time.Date(2026, 8, 29, 12, 0, 0, 123, time.UTC),
		EventID:    uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	}
	got, err := DecodeCursor(want.Encode())
	if err != nil {
		t.Fatalf("decode cursor: %v", err)
	}
	if !got.OccurredAt.Equal(want.OccurredAt) || got.EventID != want.EventID {
		t.Fatalf("cursor = %+v, want %+v", got, want)
	}
}
