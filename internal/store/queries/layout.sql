-- name: CreateFloor :one
INSERT INTO floors (
    organisation_id,
    location_id,
    slug,
    name,
    sort_order,
    canvas,
    background_asset_ref
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CreateZone :one
INSERT INTO zones (organisation_id, location_id, floor_id, name, sort_order)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateTable :one
INSERT INTO tables (
    organisation_id,
    location_id,
    floor_id,
    zone_id,
    label,
    capacity_label,
    shape,
    geometry,
    legacy_table_id
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: CreateTableOccupancy :one
INSERT INTO table_occupancy (table_id, organisation_id, location_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateTableQRCapability :one
INSERT INTO table_qr_capabilities (
    organisation_id,
    location_id,
    table_id,
    token,
    token_lookup_prefix,
    token_hash,
    label,
    expires_at
)
VALUES (
    sqlc.arg(organisation_id)::uuid,
    sqlc.arg(location_id)::uuid,
    sqlc.arg(table_id)::uuid,
    left(sqlc.arg(token)::text, 16),
    left(sqlc.arg(token)::text, 16),
    digest(sqlc.arg(token)::text, 'sha256'),
    sqlc.arg(label)::text,
    sqlc.arg(expires_at)::timestamptz
)
RETURNING *;

-- name: ListFloorsByLocation :many
SELECT *
FROM floors
WHERE organisation_id = $1 AND location_id = $2 AND is_active
ORDER BY sort_order, name;

-- name: ListAllFloorsByLocation :many
SELECT *
FROM floors
WHERE organisation_id = $1 AND location_id = $2
ORDER BY is_active DESC, sort_order, name;

-- name: ListZonesByFloor :many
SELECT *
FROM zones
WHERE organisation_id = $1 AND location_id = $2 AND floor_id = $3 AND is_active
ORDER BY sort_order, name;

-- name: ListAllZonesByLocation :many
SELECT *
FROM zones
WHERE organisation_id = $1 AND location_id = $2
ORDER BY is_active DESC, floor_id, sort_order, name;

-- name: ListTablesByFloor :many
SELECT *
FROM tables
WHERE organisation_id = $1
  AND location_id = $2
  AND floor_id = $3
  AND is_active
  AND deleted_at IS NULL
ORDER BY label;

-- name: ListAllTablesByLocation :many
SELECT *
FROM tables
WHERE organisation_id = $1
  AND location_id = $2
ORDER BY is_active DESC, deleted_at NULLS FIRST, floor_id, label;

-- name: ListTableStatesByLocationIncludingArchived :many
SELECT sqlc.embed(t), sqlc.embed(o)
FROM tables t
LEFT JOIN table_occupancy o ON o.table_id = t.id
WHERE t.organisation_id = $1
  AND t.location_id = $2
ORDER BY t.floor_id, t.label;

-- name: UpdateFloor :one
UPDATE floors
SET slug = $4,
    name = $5,
    sort_order = $6,
    canvas = $7,
    background_asset_ref = $8,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $9
RETURNING *;

-- name: ArchiveFloor :one
UPDATE floors
SET is_active = false,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $4
RETURNING *;

-- name: RestoreFloor :one
UPDATE floors
SET is_active = true,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $4
RETURNING *;

-- name: UpdateZone :one
UPDATE zones
SET name = $5,
    sort_order = $6,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND floor_id = $4
  AND version = $7
RETURNING *;

-- name: ArchiveZone :one
UPDATE zones
SET is_active = false,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND floor_id = $4
  AND version = $5
RETURNING *;

-- name: UpdateTable :one
UPDATE tables
SET floor_id = $4,
    zone_id = $5,
    label = $6,
    capacity_label = $7,
    shape = $8,
    geometry = $9,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $10
RETURNING *;

-- name: ArchiveTable :one
UPDATE tables
SET is_active = false,
    deleted_at = now(),
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $4
RETURNING *;

-- name: RestoreTable :one
UPDATE tables
SET is_active = true,
    deleted_at = NULL,
    version = version + 1,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
  AND location_id = $3
  AND version = $4
RETURNING *;

-- name: GetTableDetail :one
SELECT sqlc.embed(t), sqlc.embed(o)
FROM tables t
JOIN table_occupancy o ON o.table_id = t.id
WHERE t.id = $1
  AND t.organisation_id = $2
  AND t.location_id = $3
  AND t.is_active
  AND t.deleted_at IS NULL;

-- name: ListLocationTableStates :many
SELECT sqlc.embed(t), sqlc.embed(o)
FROM tables t
JOIN table_occupancy o ON o.table_id = t.id
WHERE t.organisation_id = $1
  AND t.location_id = $2
  AND t.is_active
  AND t.deleted_at IS NULL
ORDER BY t.label;
