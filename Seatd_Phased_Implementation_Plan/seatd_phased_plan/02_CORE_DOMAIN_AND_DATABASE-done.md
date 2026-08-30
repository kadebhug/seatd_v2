# 02 — Core Domain and Database

## Goal

Create a correct PostgreSQL domain model that becomes Seatd's authoritative operational truth.

## Dependencies

- `01_FOUNDATION_AND_ENGINEERING_BASELINE.md`

## Phase 1 — Model tenancy correctly

Create:

```text
Organisation
  └── Location
```

Minimum tables:
- `organisations`;
- `locations`;
- `organisation_memberships`;
- `location_memberships` where location-specific access is required.

Migration rule:
- every current restaurant becomes one Organisation with one Location;
- preserve stable legacy identifiers in migration mapping tables or explicit legacy-id columns until cutover is complete.

### Exit criteria

A standalone venue and a multi-location group can both be represented without schema changes.

## Phase 2 — Build the floor hierarchy

Create:

```text
Location
  └── Floor
       └── Zone
            └── Table
```

Required properties:
- floor canvas/layout metadata;
- optional background asset reference;
- zone name and sort order;
- table geometry, capacity label, shape, active/deleted status;
- stable QR capability token metadata separate from visual placement.

### Exit criteria

Every table belongs to exactly one floor and one zone; zones can be used independently for assignment and analytics.

## Phase 3 — Separate occupancy from attention

Replace the legacy composite state with explicit operational models.

### Occupancy

```text
available | occupied
```

### Assist request

```text
pending | acknowledged | resolved | cancelled
```

Rules:
- an assist request never changes occupancy by itself;
- resolving an assist never clears a table;
- attention is a query/derived condition: one or more active assists exist;
- clearing a table is an explicit command.

### Exit criteria

The historical bug where resolving the last assist frees an occupied table is structurally impossible.

## Phase 4 — Introduce TableSession

Create `table_sessions` with at least:
- id;
- organisation_id;
- location_id;
- table_id;
- started_at;
- ended_at;
- party_size nullable;
- opened_by;
- closed_by;
- source;
- created_at / updated_at as needed.

Rules:
- occupying a table opens one active session;
- clearing closes the active session;
- only one active session per table;
- session history is never overwritten by later occupancy.

Use a partial unique constraint or equivalent mechanism to enforce one active session.

### Exit criteria

Utilisation and turnover can be calculated from sessions without reconstructing them from table snapshots.

## Phase 5 — Add entity versions

Add monotonic integer versions to mutable operational aggregates, especially:
- table occupancy state;
- assist request where concurrent actions matter;
- editable layout/configuration resources as needed.

Rules:
- clients submit `expectedVersion`;
- successful mutation increments version;
- mismatch returns a typed conflict;
- conflicts include enough current state for reconciliation.

### Exit criteria

Two devices cannot silently overwrite each other's operational changes.

## Phase 6 — Constraints and integrity

Enforce invariants in PostgreSQL where practical:
- valid foreign-key tenancy chains;
- one current occupancy record per table;
- one active TableSession per table;
- valid lifecycle states;
- soft-deleted tables excluded from live queries;
- unique QR tokens;
- unique slugs within the chosen scope;
- timestamps and audit fields consistently populated.

Do not rely only on application code for invariants.

## Phase 7 — SQL-first query layer

Use `pgx` plus `sqlc` or equivalent SQL generation.

Organize queries by domain module:
- organisation;
- location;
- floor;
- zone;
- table;
- session;
- assist;
- device.

Add transactional service methods around multi-write operations.

## Deliverables

- domain ERD;
- migration set;
- SQL query package;
- invariants document;
- legacy-to-target mapping plan;
- seed fixtures for a realistic restaurant floor;
- integration tests for every state transition.

## Definition of done

A database-only test suite can prove the valid/invalid transitions for seat, assist, acknowledge, resolve, cancel, and clear workflows.
