package events

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const SchemaVersion = 1

const (
	TypeTableOccupied      = "table.occupied"
	TypeTableCleared       = "table.cleared"
	TypeSessionOpened      = "session.opened"
	TypeSessionClosed      = "session.closed"
	TypeAssistRequested    = "assist.requested"
	TypeAssistAcknowledged = "assist.acknowledged"
	TypeAssistResolved     = "assist.resolved"
	TypeAssistCancelled    = "assist.cancelled"
	TypeDeviceRegistered   = "device.registered"
	TypeDeviceRevoked      = "device.revoked"
)

type Envelope struct {
	ID             uuid.UUID       `json:"id"`
	Type           string          `json:"type"`
	SchemaVersion  int32           `json:"schemaVersion"`
	OccurredAt     time.Time       `json:"occurredAt"`
	OrganisationID uuid.UUID       `json:"organisationId"`
	LocationID     uuid.UUID       `json:"locationId"`
	ActorRef       string          `json:"actorRef,omitempty"`
	DeviceID       *uuid.UUID      `json:"deviceId,omitempty"`
	EntityType     string          `json:"entityType"`
	EntityID       uuid.UUID       `json:"entityId"`
	EntityVersion  *int32          `json:"entityVersion,omitempty"`
	CommandID      *uuid.UUID      `json:"commandId,omitempty"`
	CorrelationID  *uuid.UUID      `json:"correlationId,omitempty"`
	Traceparent    string          `json:"traceparent,omitempty"`
	Tracestate     string          `json:"tracestate,omitempty"`
	Data           json.RawMessage `json:"data"`
}

type Data map[string]any

func MarshalData(data Data) ([]byte, error) {
	if data == nil {
		return []byte(`{}`), nil
	}
	return json.Marshal(data)
}
