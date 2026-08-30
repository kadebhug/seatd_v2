package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/kadebhug/seatd_v2/internal/domain/events"
)

const DestinationOperations = "realtime.operations"

type Scope struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
}

type Message struct {
	Type           string          `json:"type"`
	EventID        string          `json:"eventId"`
	OrganisationID string          `json:"organisationId"`
	LocationID     string          `json:"locationId"`
	EntityType     string          `json:"entityType"`
	EntityID       string          `json:"entityId"`
	Version        *int32          `json:"version,omitempty"`
	OccurredAt     string          `json:"occurredAt"`
	Data           json.RawMessage `json:"data"`
}

type Hub struct {
	logger *slog.Logger

	mu      sync.RWMutex
	clients map[Scope]map[*Client]struct{}
	metrics Metrics
}

type Metrics struct {
	ConnectionsAccepted atomic.Int64
	ConnectionsClosed   atomic.Int64
	MessagesPublished   atomic.Int64
	MessagesDropped     atomic.Int64
	BackpressureCloses  atomic.Int64
}

type Snapshot struct {
	ActiveConnections   int   `json:"activeConnections"`
	ConnectionsAccepted int64 `json:"connectionsAccepted"`
	ConnectionsClosed   int64 `json:"connectionsClosed"`
	MessagesPublished   int64 `json:"messagesPublished"`
	MessagesDropped     int64 `json:"messagesDropped"`
	BackpressureCloses  int64 `json:"backpressureCloses"`
}

func NewHub(logger *slog.Logger) *Hub {
	if logger == nil {
		logger = slog.Default()
	}
	return &Hub{
		logger:  logger,
		clients: make(map[Scope]map[*Client]struct{}),
	}
}

func (h *Hub) Subscribe(scope Scope, actorRef string, deviceID uuid.UUID, buffer int) *Client {
	if buffer <= 0 {
		buffer = 32
	}
	client := &Client{
		scope:    scope,
		actorRef: actorRef,
		deviceID: deviceID,
		send:     make(chan Message, buffer),
		done:     make(chan struct{}),
	}
	h.mu.Lock()
	if h.clients[scope] == nil {
		h.clients[scope] = make(map[*Client]struct{})
	}
	h.clients[scope][client] = struct{}{}
	h.mu.Unlock()
	h.metrics.ConnectionsAccepted.Add(1)
	return client
}

func (h *Hub) Unsubscribe(client *Client) {
	client.closeOnce.Do(func() {
		h.mu.Lock()
		if scoped := h.clients[client.scope]; scoped != nil {
			delete(scoped, client)
			if len(scoped) == 0 {
				delete(h.clients, client.scope)
			}
		}
		h.mu.Unlock()
		client.close()
		h.metrics.ConnectionsClosed.Add(1)
	})
}

func (h *Hub) Publish(ctx context.Context, event events.Envelope) {
	message := MessageFromEvent(event)
	scope := Scope{OrganisationID: event.OrganisationID, LocationID: event.LocationID}

	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients[scope]))
	for client := range h.clients[scope] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	h.metrics.MessagesPublished.Add(1)
	for _, client := range clients {
		if !client.trySend(message) {
			h.metrics.MessagesDropped.Add(1)
			h.metrics.BackpressureCloses.Add(1)
			h.logger.WarnContext(ctx, "closing realtime connection for backpressure",
				"organisation_id", scope.OrganisationID,
				"location_id", scope.LocationID,
				"actor_ref", client.actorRef,
				"device_id", client.deviceID,
			)
			h.Unsubscribe(client)
		}
	}
}

func (h *Hub) MetricsSnapshot() Snapshot {
	h.mu.RLock()
	active := 0
	for _, scoped := range h.clients {
		active += len(scoped)
	}
	h.mu.RUnlock()
	return Snapshot{
		ActiveConnections:   active,
		ConnectionsAccepted: h.metrics.ConnectionsAccepted.Load(),
		ConnectionsClosed:   h.metrics.ConnectionsClosed.Load(),
		MessagesPublished:   h.metrics.MessagesPublished.Load(),
		MessagesDropped:     h.metrics.MessagesDropped.Load(),
		BackpressureCloses:  h.metrics.BackpressureCloses.Load(),
	}
}

func MessageFromEvent(event events.Envelope) Message {
	return Message{
		Type:           event.Type,
		EventID:        event.ID.String(),
		OrganisationID: event.OrganisationID.String(),
		LocationID:     event.LocationID.String(),
		EntityType:     event.EntityType,
		EntityID:       event.EntityID.String(),
		Version:        event.EntityVersion,
		OccurredAt:     event.OccurredAt.UTC().Format(time.RFC3339Nano),
		Data:           event.Data,
	}
}

type Client struct {
	scope     Scope
	actorRef  string
	deviceID  uuid.UUID
	send      chan Message
	done      chan struct{}
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) trySend(message Message) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- message:
		return true
	default:
		return false
	}
}

func (c *Client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	close(c.done)
	close(c.send)
}
