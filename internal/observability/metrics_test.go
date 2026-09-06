package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/domain/events"
	"github.com/kadebhug/seatd_v2/internal/realtime"
)

func TestHTTPMiddlewareUsesRoutePattern(t *testing.T) {
	t.Parallel()

	metrics := NewServiceMetrics(testConfig("api"), nil)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/tables/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/tables/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()
	metrics.Middleware(mux).ServeHTTP(rec, req)

	metricsRec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := metricsRec.Body.String()
	if !strings.Contains(body, `route="GET /v1/tables/{id}"`) {
		t.Fatalf("metrics did not use route pattern:\n%s", body)
	}
	if strings.Contains(body, "11111111-1111-1111-1111-111111111111") {
		t.Fatalf("metrics leaked raw path value:\n%s", body)
	}
	if !strings.Contains(body, `status="418"`) {
		t.Fatalf("metrics did not include response status:\n%s", body)
	}
}

func TestRealtimeHubCollector(t *testing.T) {
	t.Parallel()

	metrics := NewServiceMetrics(testConfig("realtime"), nil)
	hub := realtime.NewHub(nil)
	metrics.RegisterRealtimeHub(hub)

	scope := realtime.Scope{OrganisationID: uuid.New(), LocationID: uuid.New()}
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
	<-client.Done()

	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	for _, want := range []string{
		"seatd_realtime_connections_accepted_total",
		"seatd_realtime_messages_published_total",
		"seatd_realtime_messages_dropped_total",
		"seatd_realtime_backpressure_closes_total",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics missing %s:\n%s", want, body)
		}
	}
}

func testConfig(service string) app.Config {
	return app.Config{
		Name:        service,
		Environment: app.EnvTest,
		Version:     app.BuildInfo{Version: "dev", Commit: "test", Date: "test"},
	}
}
