# Core Domain

This phase makes PostgreSQL the authoritative source for Seatd operational
state. The model separates tenant ownership, physical layout, occupancy,
assist lifecycle, and table session history.

## ERD

```text
organisations
  |-- locations
  |     |-- floors
  |     |     |-- zones
  |     |     |     |-- tables
  |     |     |           |-- table_occupancy
  |     |     |           |-- table_qr_capabilities
  |     |     |           |-- table_sessions
  |     |     |           |-- assist_requests
  |     |-- location_memberships
  |-- organisation_memberships
  |-- legacy_restaurant_mappings
  |-- legacy_table_mappings
```

## Operational Rules

- Organisation owns one or more locations.
- Every floor, zone, table, occupancy row, session, and assist request carries
  organisation and location identifiers.
- Every table belongs to exactly one floor and one zone.
- Occupancy is only `available` or `occupied`.
- Attention is derived from active assist requests, not stored on the table.
- Occupying a table opens one active table session.
- Clearing a table closes the active table session.
- Resolving or cancelling assists never changes occupancy.
- Mutable operational rows use monotonic integer versions for optimistic
  concurrency checks.

## Query Layer

SQL lives under `internal/store/queries` and generated Go code lives under
`internal/store/db`. Regenerate it with:

```sh
go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.27.0 generate
```

The domain operation service in `internal/domain/operations` wraps multi-write
commands in transactions and returns typed domain errors for conflicts and
invalid state transitions.
