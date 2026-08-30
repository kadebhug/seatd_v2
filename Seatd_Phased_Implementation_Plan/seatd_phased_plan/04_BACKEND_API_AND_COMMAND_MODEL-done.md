# 04 — Backend API and Command Model

## Goal

Build the Go modular monolith and expose explicit, versioned domain commands rather than generic state mutations.

## Dependencies

- `02_CORE_DOMAIN_AND_DATABASE.md`
- `03_IDENTITY_TENANCY_AND_SECURITY.md`

## Phase 1 — Modular monolith skeleton

Create domain modules under `internal/`:
- organisation;
- location;
- identity;
- floor;
- zone;
- table;
- session;
- assist;
- device;
- events;
- outbox;
- analytics;
- integrations.

Module rules:
- domain logic is not placed in HTTP handlers;
- modules communicate through explicit services/interfaces;
- transactions are owned by application services;
- no direct cross-module table writes from arbitrary code.

## Phase 2 — Read APIs

Implement resource reads first:
- organisation/location configuration;
- floors/zones/tables;
- live location/floor state;
- table detail;
- active assists;
- device list;
- membership/staff list.

Define response DTOs in OpenAPI.

## Phase 3 — Replace toggles with commands

Implement explicit mutations:

```text
POST /v1/tables/:id/occupy
POST /v1/tables/:id/clear
POST /v1/assists/:id/acknowledge
POST /v1/assists/:id/resolve
POST /v1/assists/:id/cancel
```

Also add configuration commands/resources for floors, zones, tables, staff, devices, QR actions, service periods.

Each operational mutation includes:
- idempotency key/command ID;
- expected entity version where relevant;
- actor identity;
- device identity where relevant;
- location context.

## Phase 4 — Transactional command handlers

Example occupy handler transaction:

```text
BEGIN
  validate tenant + permission
  lock/read current table state
  validate expected version
  update occupancy
  create TableSession
  write operational_event
  write outbox_record
COMMIT
```

Clear table similarly closes the active session.

Assist handlers never mutate occupancy.

## Phase 5 — Error contract

Define stable machine-readable errors:
- validation_failed;
- forbidden;
- tenant_disabled;
- not_found;
- version_conflict;
- already_occupied;
- already_available;
- no_active_session;
- assist_already_resolved;
- idempotency_conflict;
- rate_limited.

Do not make clients infer behaviour from free-text messages.

## Phase 6 — Idempotency

Every mutation used by mobile/offline clients accepts a command ID.

Server behaviour:
- first valid command executes;
- retries with same command/payload return the original result;
- same ID with a different payload is rejected;
- retention duration is documented.

## Phase 7 — OpenAPI and generated clients

For every endpoint:
- update OpenAPI first or in the same change;
- regenerate Dart/TypeScript clients;
- run compatibility checks;
- publish API examples.

## Phase 8 — Compatibility layer during migration

If the legacy client must run during cutover, implement a temporary adapter for old semantics where safe.

Do **not** emulate the legacy assist-resolve → table-available bug.

Prefer feature flags or versioned routes over contaminating new domain logic.

## Deliverables

- Go API;
- explicit command handlers;
- OpenAPI contract;
- generated clients;
- typed error contract;
- idempotency store;
- API integration tests;
- temporary compatibility endpoints only where required.

## Definition of done

The complete service wedge can be driven using HTTP alone with deterministic outcomes under retries and concurrent updates.
