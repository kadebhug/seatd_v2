-- name: GetTableState :one
SELECT sqlc.embed(t), sqlc.embed(o)
FROM tables t
JOIN table_occupancy o ON o.table_id = t.id
WHERE t.id = $1
  AND t.organisation_id = $2
  AND t.location_id = $3
  AND t.is_active
  AND t.deleted_at IS NULL;

-- name: LockTableOccupancy :one
SELECT o.*
FROM table_occupancy o
JOIN tables t ON t.id = o.table_id
WHERE o.table_id = $1
  AND o.organisation_id = $2
  AND o.location_id = $3
  AND t.is_active
  AND t.deleted_at IS NULL
FOR UPDATE OF o;

-- name: CreateTableSession :one
INSERT INTO table_sessions (
    organisation_id,
    location_id,
    table_id,
    party_size,
    opened_by,
    source
)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: MarkTableOccupied :one
UPDATE table_occupancy
SET status = 'occupied',
    current_session_id = $4,
    version = version + 1,
    updated_by = $5,
    updated_at = now()
WHERE table_id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND status = 'available'
  AND version = $6
RETURNING *;

-- name: CloseTableSession :one
UPDATE table_sessions
SET ended_at = now(),
    closed_by = $4,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND ended_at IS NULL
RETURNING *;

-- name: MarkTableAvailable :one
UPDATE table_occupancy
SET status = 'available',
    current_session_id = NULL,
    version = version + 1,
    updated_by = $4,
    updated_at = now()
WHERE table_id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND status = 'occupied'
  AND version = $5
RETURNING *;

-- name: CreateAssistRequest :one
INSERT INTO assist_requests (
    organisation_id,
    location_id,
    table_id,
    table_session_id,
    requested_by,
    source,
    note,
    action_key
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetPendingGuestAssistByAction :one
SELECT *
FROM assist_requests
WHERE organisation_id = $1
  AND location_id = $2
  AND table_id = $3
  AND table_session_id = $4
  AND action_key = $5
  AND source = 'guest_qr'
  AND status = 'pending'
ORDER BY requested_at
LIMIT 1
FOR UPDATE;

-- name: GetActiveGuestAssistBySession :one
SELECT *
FROM assist_requests
WHERE organisation_id = $1
  AND location_id = $2
  AND table_id = $3
  AND table_session_id = $4
  AND source = 'guest_qr'
  AND status IN ('pending', 'acknowledged')
ORDER BY requested_at DESC
LIMIT 1;

-- name: GetLatestTerminalGuestAssistByAction :one
SELECT *
FROM assist_requests
WHERE organisation_id = $1
  AND location_id = $2
  AND table_id = $3
  AND table_session_id = $4
  AND action_key = $5
  AND source = 'guest_qr'
  AND status IN ('resolved', 'cancelled')
ORDER BY requested_at DESC
LIMIT 1;

-- name: GetGuestAssistByID :one
SELECT *
FROM assist_requests
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND table_id = $4
  AND source = 'guest_qr';

-- name: CancelPendingGuestAssistRequest :one
UPDATE assist_requests
SET status = 'cancelled',
    cancelled_at = now(),
    cancelled_by = $4,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND source = 'guest_qr'
  AND status = 'pending'
RETURNING *;

-- name: LockAssistRequest :one
SELECT *
FROM assist_requests
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
FOR UPDATE;

-- name: AcknowledgeAssistRequest :one
UPDATE assist_requests
SET status = 'acknowledged',
    acknowledged_at = now(),
    acknowledged_by = $4,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND status = 'pending'
  AND version = $5
RETURNING *;

-- name: ResolveAssistRequest :one
UPDATE assist_requests
SET status = 'resolved',
    resolved_at = now(),
    resolved_by = $4,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND status IN ('pending', 'acknowledged')
  AND version = $5
RETURNING *;

-- name: CancelAssistRequest :one
UPDATE assist_requests
SET status = 'cancelled',
    cancelled_at = now(),
    cancelled_by = $4,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND status IN ('pending', 'acknowledged')
  AND version = $5
RETURNING *;

-- name: ListActiveAssistsByTable :many
SELECT *
FROM assist_requests
WHERE organisation_id = $1
  AND location_id = $2
  AND table_id = $3
  AND status IN ('pending', 'acknowledged')
ORDER BY requested_at;

-- name: ListActiveAssistsByLocation :many
SELECT *
FROM assist_requests
WHERE organisation_id = $1
  AND location_id = $2
  AND status IN ('pending', 'acknowledged')
ORDER BY requested_at;
