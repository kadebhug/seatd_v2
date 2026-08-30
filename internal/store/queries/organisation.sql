-- name: CreateOrganisation :one
INSERT INTO organisations (slug, name, legacy_restaurant_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateLocation :one
INSERT INTO locations (organisation_id, slug, name, timezone, legacy_restaurant_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetOrganisation :one
SELECT *
FROM organisations
WHERE id = $1;

-- name: GetLocation :one
SELECT *
FROM locations
WHERE id = $1 AND organisation_id = $2;

-- name: ListLocationsByOrganisation :many
SELECT *
FROM locations
WHERE organisation_id = $1
ORDER BY name, slug;

-- name: UpdateOrganisation :one
UPDATE organisations
SET name = $3,
    status = $4,
    updated_at = now()
WHERE id = $1
  AND slug = $2
RETURNING *;

-- name: UpdateLocation :one
UPDATE locations
SET name = $3,
    timezone = $4,
    status = $5,
    operating_config = $6,
    feature_flags = $7,
    updated_at = now()
WHERE id = $1
  AND organisation_id = $2
RETURNING *;

-- name: CreateOrganisationMembership :one
INSERT INTO organisation_memberships (organisation_id, member_ref, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateLocationMembership :one
INSERT INTO location_memberships (organisation_id, location_id, member_ref, role)
VALUES ($1, $2, $3, $4)
RETURNING *;
