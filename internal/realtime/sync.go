package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type tableDTO struct {
	ID            string          `json:"id"`
	FloorID       string          `json:"floorId"`
	ZoneID        string          `json:"zoneId"`
	Label         string          `json:"label"`
	CapacityLabel string          `json:"capacityLabel"`
	Shape         string          `json:"shape"`
	Geometry      json.RawMessage `json:"geometry"`
	Version       int32           `json:"version"`
}

type occupancyDTO struct {
	Status           string  `json:"status"`
	CurrentSessionID *string `json:"currentSessionId,omitempty"`
	Version          int32   `json:"version"`
	UpdatedAt        string  `json:"updatedAt"`
}

type tableStateDTO struct {
	Table     tableDTO     `json:"table"`
	Occupancy occupancyDTO `json:"occupancy"`
}

type assistDTO struct {
	ID             string  `json:"id"`
	TableID        string  `json:"tableId"`
	TableSessionID *string `json:"tableSessionId,omitempty"`
	Status         string  `json:"status"`
	RequestedAt    string  `json:"requestedAt"`
	Version        int32   `json:"version"`
	Note           *string `json:"note,omitempty"`
}

func loadCurrentCursor(ctx context.Context, tx pgx.Tx, req requestContext) (Cursor, error) {
	var eventID uuid.UUID
	var occurredAt pgtype.Timestamptz
	err := tx.QueryRow(ctx, `
SELECT id, occurred_at
FROM operational_events
WHERE organisation_id = $1 AND location_id = $2
ORDER BY occurred_at DESC, id DESC
LIMIT 1
`, req.OrganisationID, req.LocationID).Scan(&eventID, &occurredAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Cursor{}, nil
		}
		return Cursor{}, fmt.Errorf("loading current event cursor: %w", err)
	}
	return cursorFrom(occurredAt.Time, eventID), nil
}

func loadTableStates(ctx context.Context, tx pgx.Tx, req requestContext) ([]tableStateDTO, error) {
	rows, err := tx.Query(ctx, `
SELECT t.id, t.floor_id, t.zone_id, t.label, t.capacity_label, t.shape, t.geometry, t.version,
       o.status, o.current_session_id, o.version, o.updated_at
FROM tables t
JOIN table_occupancy o ON o.table_id = t.id
WHERE t.organisation_id = $1
  AND t.location_id = $2
  AND t.is_active
  AND t.deleted_at IS NULL
ORDER BY t.label
`, req.OrganisationID, req.LocationID)
	if err != nil {
		return nil, fmt.Errorf("loading location table states: %w", err)
	}
	defer rows.Close()

	states := []tableStateDTO{}
	for rows.Next() {
		var tableID, floorID, zoneID uuid.UUID
		var label, capacityLabel, shape string
		var geometry []byte
		var tableVersion int32
		var status string
		var sessionID uuid.NullUUID
		var occupancyVersion int32
		var updatedAt pgtype.Timestamptz
		if err := rows.Scan(&tableID, &floorID, &zoneID, &label, &capacityLabel, &shape, &geometry, &tableVersion, &status, &sessionID, &occupancyVersion, &updatedAt); err != nil {
			return nil, fmt.Errorf("scanning location table state: %w", err)
		}
		states = append(states, tableStateDTO{
			Table: tableDTO{
				ID:            tableID.String(),
				FloorID:       floorID.String(),
				ZoneID:        zoneID.String(),
				Label:         label,
				CapacityLabel: capacityLabel,
				Shape:         shape,
				Geometry:      json.RawMessage(geometry),
				Version:       tableVersion,
			},
			Occupancy: occupancyDTO{
				Status:           status,
				CurrentSessionID: uuidPtr(sessionID),
				Version:          occupancyVersion,
				UpdatedAt:        timeString(updatedAt),
			},
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating location table states: %w", err)
	}
	return states, nil
}

func loadActiveAssists(ctx context.Context, tx pgx.Tx, req requestContext) ([]assistDTO, error) {
	rows, err := tx.Query(ctx, `
SELECT id, table_id, table_session_id, status, requested_at, version, note
FROM assist_requests
WHERE organisation_id = $1
  AND location_id = $2
  AND status IN ('pending', 'acknowledged')
ORDER BY requested_at, id
`, req.OrganisationID, req.LocationID)
	if err != nil {
		return nil, fmt.Errorf("loading active assists: %w", err)
	}
	defer rows.Close()

	assists := []assistDTO{}
	for rows.Next() {
		var assistID, tableID uuid.UUID
		var sessionID uuid.NullUUID
		var status string
		var requestedAt pgtype.Timestamptz
		var version int32
		var note pgtype.Text
		if err := rows.Scan(&assistID, &tableID, &sessionID, &status, &requestedAt, &version, &note); err != nil {
			return nil, fmt.Errorf("scanning active assist: %w", err)
		}
		assists = append(assists, assistDTO{
			ID:             assistID.String(),
			TableID:        tableID.String(),
			TableSessionID: uuidPtr(sessionID),
			Status:         status,
			RequestedAt:    timeString(requestedAt),
			Version:        version,
			Note:           textPtr(note),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating active assists: %w", err)
	}
	return assists, nil
}

func loadEventsAfter(ctx context.Context, tx pgx.Tx, req requestContext, after Cursor, limit int32) ([]Message, Cursor, error) {
	rows, err := tx.Query(ctx, `
SELECT id, event_type, occurred_at, entity_type, entity_id, entity_version, event_data
FROM operational_events
WHERE organisation_id = $1
  AND location_id = $2
  AND (
      $3::timestamptz IS NULL
      OR (occurred_at, id) > ($3::timestamptz, $4::uuid)
  )
ORDER BY occurred_at, id
LIMIT $5
`, req.OrganisationID, req.LocationID, cursorTime(after), cursorUUID(after), limit)
	if err != nil {
		return nil, Cursor{}, fmt.Errorf("loading realtime events after cursor: %w", err)
	}
	defer rows.Close()

	messages := []Message{}
	next := after
	for rows.Next() {
		var eventID, entityID uuid.UUID
		var eventType, entityType string
		var occurredAt pgtype.Timestamptz
		var entityVersion pgtype.Int4
		var data []byte
		if err := rows.Scan(&eventID, &eventType, &occurredAt, &entityType, &entityID, &entityVersion, &data); err != nil {
			return nil, Cursor{}, fmt.Errorf("scanning realtime event: %w", err)
		}
		var version *int32
		if entityVersion.Valid {
			value := entityVersion.Int32
			version = &value
		}
		messages = append(messages, Message{
			Type:           eventType,
			EventID:        eventID.String(),
			OrganisationID: req.OrganisationID.String(),
			LocationID:     req.LocationID.String(),
			EntityType:     entityType,
			EntityID:       entityID.String(),
			Version:        version,
			OccurredAt:     timeString(occurredAt),
			Data:           json.RawMessage(data),
		})
		next = cursorFrom(occurredAt.Time, eventID)
	}
	if err := rows.Err(); err != nil {
		return nil, Cursor{}, fmt.Errorf("iterating realtime events: %w", err)
	}
	return messages, next, nil
}

func cursorTime(cursor Cursor) any {
	if cursor.OccurredAt.IsZero() {
		return nil
	}
	return cursor.OccurredAt
}

func cursorUUID(cursor Cursor) any {
	if cursor.EventID == uuid.Nil {
		return nil
	}
	return cursor.EventID
}

func uuidPtr(value uuid.NullUUID) *string {
	if !value.Valid {
		return nil
	}
	out := value.UUID.String()
	return &out
}

func textPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	out := value.String
	return &out
}
