# Seatd — Master Implementation Roadmap

**Status:** Execution plan  
**Basis:** `Seatd_Technology_Architecture_Recommendations(1).md` + current Seatd system functionality reference  
**Objective:** Move Seatd from the current implementation to the recommended portable, event-driven architecture without breaking the operational wedge already proven by the product.

---

## 1. Guiding rule

Build Seatd in dependency order:

> **Operational truth → durable commands/events → reliable clients → management surfaces → intelligence → integrations → scale.**

Do not rebuild every surface at once. The backend domain model, database contracts, tenancy, command semantics, and event history are the foundation all clients depend on.

---

## 2. Ordered document set

| Order | Document | Major system part | Primary outcome |
|---:|---|---|---|
| 01 | `01_FOUNDATION_AND_ENGINEERING_BASELINE.md` | Repository, contracts, environments | Stable development and delivery baseline |
| 02 | `02_CORE_DOMAIN_AND_DATABASE.md` | PostgreSQL + domain model | Correct source of operational truth |
| 03 | `03_IDENTITY_TENANCY_AND_SECURITY.md` | Auth, authorization, RLS, devices | Secure multi-tenant access model |
| 04 | `04_BACKEND_API_AND_COMMAND_MODEL.md` | Go API | Explicit, versioned operational API |
| 05 | `05_EVENTS_OUTBOX_AND_WORKERS.md` | Event log, outbox, workers | Durable history and async processing |
| 06 | `06_REALTIME_AND_SYNC.md` | WebSockets + sync protocol | Reliable live state propagation |
| 07 | `07_WAITER_APP_OFFLINE_FIRST.md` | Flutter waiter app | Offline-capable operational client |
| 08 | `08_OWNER_AND_PLATFORM_WEB.md` | Next.js owner/admin | Complete management surfaces |
| 09 | `09_GUEST_QR_EXPERIENCE.md` | Guest web | Fast, secure no-login guest assist |
| 10 | `10_DISPLAY_KIOSK_AND_DEVICE_MANAGEMENT.md` | Flutter kiosk + devices | Managed restaurant displays and devices |
| 11 | `11_ANALYTICS_AND_REPORTING.md` | Projections and reporting | Actionable floor intelligence |
| 12 | `12_INTEGRATIONS_AND_RECONCILIATION.md` | POS/external adapters | Safe integration layer without domain pollution |
| 13 | `13_INFRASTRUCTURE_CICD_OBSERVABILITY.md` | Deployment and operations | Portable production platform without AWS dependency |
| 14 | `14_TESTING_MIGRATION_AND_CUTOVER.md` | QA + legacy migration | Controlled transition from current Seatd |
| 15 | `15_SCALE_UP_TRIGGERS_AND_FUTURE_ARCHITECTURE.md` | NATS/Valkey/ClickHouse/Temporal/K8s | Explicit rules for adding complexity |

---

## 3. Programme phases

### Programme Phase A — Correct the core

Complete documents **01–04**.

Deliver:
- new monorepo baseline;
- Go modular monolith skeleton;
- PostgreSQL schema with Organisation → Location → Floor → Zone → Table;
- independent occupancy and assist lifecycles;
- TableSession;
- multi-tenant authorization;
- explicit command API;
- optimistic concurrency;
- OpenAPI contracts.

**Exit gate:** the new core can represent every current operational flow without relying on the legacy `attention` table state or generic table toggle semantics.

### Programme Phase B — Make operations durable

Complete documents **05–06**.

Deliver:
- immutable operational events;
- transactional outbox;
- idempotent command processing;
- Go workers;
- audit reconstruction;
- realtime gateway;
- client sync protocol.

**Exit gate:** a committed operational change cannot be lost even if realtime, analytics, or another downstream consumer is temporarily unavailable.

### Programme Phase C — Rebuild the operational edge

Complete documents **07, 09, 10**.

Deliver:
- production waiter app for Android/iOS/tablet;
- SQLite/Drift local state;
- offline command queue;
- guest QR app;
- Android display/kiosk app;
- device registration and revocation.

**Exit gate:** a venue can run a service with intermittent connectivity and recover to server truth without manual database repair.

### Programme Phase D — Build the management platform

Complete document **08**.

Deliver:
- owner portal;
- platform admin;
- floor editor;
- zones;
- staff;
- QR configuration;
- device management;
- service periods;
- integration management shell.

**Exit gate:** restaurant setup and support no longer require direct SQL or provisioning scripts for normal workflows.

### Programme Phase E — Turn history into intelligence

Complete document **11**.

Deliver:
- analytics projections;
- service-period metrics;
- occupancy/utilisation/turnover metrics;
- assist response metrics;
- floor/zone/location comparisons;
- reports.

**Exit gate:** dashboards query stable projections and can explain their numbers back to operational events.

### Programme Phase F — Integrate and scale deliberately

Complete documents **12–15**.

Deliver:
- canonical integration adapters;
- webhook inbox;
- mappings;
- reconciliation;
- production observability;
- automated cutover process;
- evidence-based scale triggers.

**Exit gate:** complexity is introduced only when a measured requirement exists.

---

## 4. Critical path

```text
Foundation
   ↓
Domain + Database
   ↓
Identity/Tenancy
   ↓
Command API
   ↓
Events + Outbox
   ↓
Realtime/Sync
   ↓
Waiter App
   ├── Guest QR
   └── Kiosk/Devices
   ↓
Owner/Admin
   ↓
Analytics
   ↓
Integrations
   ↓
Scale-up only when triggered
```

The **waiter app must not be treated as the first implementation milestone**. Its offline model depends on server command semantics, entity versions, idempotency, and reconciliation.

---

## 5. Legacy behaviour that must not be carried forward

The current implementation is useful as a behavioural baseline, but several mechanics should be replaced during migration:

- `restaurant` as the only tenant level → replace with **Organisation + Location**;
- no first-class zones → add **Zone**;
- table state `available | occupied | attention` → split into **occupancy** and **assist lifecycle**;
- resolving the last assist frees a table → **remove this behaviour**;
- generic table toggle → replace with **occupy / clear** commands;
- shared restaurant PIN as operational identity → use normal identity + trusted-device quick unlock;
- filesystem floor images → move to S3-compatible object storage;
- database `LISTEN/NOTIFY` as the complete event mechanism → retain only if useful internally; durable events/outbox become authoritative for downstream processing;
- Flutter desktop-first operational client → target mobile/tablet for waiter and controlled Android for display;
- hand-maintained route contracts → OpenAPI + generated clients.

---

## 6. Scope discipline

Throughout all phases, Seatd should continue to **exclude**:

- POS functionality;
- payments;
- reservations;
- waitlists;
- delivery;
- loyalty;
- payroll;
- inventory;
- staff scheduling;
- CRM/customer profiles.

Build integration points where necessary. Do not absorb those products into the Seatd core.

---

## 7. Recommended execution model

Run work as vertical milestones rather than technology-only rewrites.

Each milestone should include:
1. database migration;
2. domain/service implementation;
3. API contract;
4. tests;
5. client support if required;
6. telemetry;
7. rollback/cutover notes;
8. documentation.

Avoid long-lived branches that rebuild the whole platform before anything can be exercised end-to-end.

---

## 8. First production milestone

The first meaningful target architecture release should support:

```text
Create organisation/location
Configure floor + zones + tables
Register staff + waiter device
Occupy table
Open TableSession
Guest requests assistance
Waiter acknowledges + resolves request
Table remains occupied
Clear table
Close TableSession
Realtime updates all connected clients
Offline waiter actions reconcile safely
Event history reconstructs the full sequence
```

If this wedge is reliable, the rest of Seatd can safely grow around it.


---

# 01 — Foundation and Engineering Baseline

## Goal

Create the repository, contract, development, and delivery foundations required before domain migration starts.

## Dependencies

None.

## Phase 1 — Establish the target monorepo

Create the recommended top-level structure:

```text
seatd/
  apps/
    api/
    worker/
    realtime/
    web/
    guest/
    waiter/
    display/
  packages/
    api-contract/
    event-schema/
    dart-seatd-client/
    typescript-seatd-client/
  infra/
  ops/
  docs/
```

Actions:
- choose one root Git repository;
- remove nested checkout assumptions for new code;
- document code ownership by area;
- standardize local commands;
- define environment naming: `local`, `test`, `staging`, `production`;
- define configuration via environment variables and secret injection;
- establish ADR directory for architectural decisions.

### Exit criteria

A clean checkout can bootstrap all required local dependencies with one documented command.

## Phase 2 — Create application skeletons

Create minimal runnable applications:
- Go API;
- Go worker;
- Go realtime gateway;
- Next.js web application;
- lightweight guest web application;
- Flutter waiter app;
- Flutter display app.

Do not implement business features yet.

Each app must expose:
- version/build metadata;
- health/status where applicable;
- structured logging;
- configuration validation;
- graceful shutdown.

### Exit criteria

CI can build every application and run basic smoke tests.

## Phase 3 — Contract-first tooling

Establish:
- OpenAPI as HTTP contract source;
- JSON Schema for operational events;
- generated Dart API client;
- generated TypeScript API client;
- versioned schemas under `packages/`.

Rules:
- no manually duplicated request/response models across clients;
- breaking API changes require explicit contract versioning or migration;
- event schemas are immutable once published; evolve with new versions.

### Exit criteria

A trivial API endpoint can be added to OpenAPI and consumed through generated Dart and TypeScript clients.

## Phase 4 — Database migration discipline

Adopt a proper migration tool and process.

Requirements:
- ordered SQL migrations;
- no production schema mutation during app startup;
- forward migrations reviewed in CI;
- migration checks on a clean database;
- migration checks from the latest supported production baseline;
- seed fixtures separated from migrations.

### Exit criteria

The complete schema can be reproduced from migrations only.

## Phase 5 — Engineering quality gates

Add:
- Go formatting/linting/tests;
- TypeScript lint/typecheck/tests;
- Flutter analyze/tests;
- SQL/static migration checks;
- dependency vulnerability scanning;
- container build checks;
- secret scanning.

Recommended CI order:

```text
lint → unit tests → integration tests → build → scan → package
```

### Exit criteria

No branch can merge when required checks fail.

## Deliverables

- target repository structure;
- local development Compose stack;
- app skeletons;
- OpenAPI/JSON Schema pipeline;
- generated client workflow;
- migration tooling;
- CI baseline;
- ADR template;
- developer setup guide.

## Do not build yet

- NATS;
- Redis/Valkey;
- ClickHouse;
- Temporal;
- Kubernetes;
- POS adapters.


---

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


---

# 03 — Identity, Tenancy, and Security

## Goal

Implement secure authentication and Seatd-owned authorization with strong tenant isolation and device trust.

## Dependencies

- `02_CORE_DOMAIN_AND_DATABASE.md`

## Phase 1 — Choose authentication provider model

Choose one:

### Managed OIDC

Use when reduced identity maintenance is worth recurring cost.

### Self-hosted Keycloak

Use when infrastructure ownership and portability are higher priorities.

Seatd must not depend on provider-specific identity concepts in its domain model.

### Exit criteria

Users authenticate through standards-based OIDC/OAuth2 and the backend validates provider-issued identity securely.

## Phase 2 — Seatd authorization model

Create Seatd-owned concepts:
- user/profile link to external identity;
- OrganisationMembership;
- optional LocationMembership;
- Role;
- Permission.

Initial roles can include:
- platform_admin;
- organisation_owner;
- location_manager;
- waiter;
- read_only/support where needed.

Rules:
- authentication answers **who are you?**;
- Seatd authorization answers **what can you do here?**;
- do not embed the entire authorization source of truth permanently inside tokens.

## Phase 3 — Tenant enforcement

Enforce tenancy twice:

1. application authorization and query scoping;
2. PostgreSQL Row Level Security on tenant-owned tables.

Add automated tests that attempt cross-tenant reads and writes.

### Exit criteria

A deliberately flawed application query is still blocked from reading another organisation's protected rows where RLS applies.

## Phase 4 — Device identity

Create first-class `devices` and device credentials.

Device types:
- waiter_mobile;
- manager_tablet;
- display;
- host_device.

Track:
- platform;
- app version;
- registered_at;
- last_seen_at;
- revoked_at;
- assigned organisation/location;
- trusted-device state.

## Phase 5 — Quick unlock

Replace shared restaurant PIN as backend identity.

Flow:

```text
OIDC authentication
  → register trusted device
  → secure refresh/device credential
  → local biometric/PIN unlock
  → authenticated server requests
```

Rules:
- local PIN/biometric never substitutes for backend identity;
- device trust can be revoked;
- secure credentials live in platform secure storage.

## Phase 6 — Guest capability security

For QR flows:
- use high-entropy opaque tokens;
- store safely, hash at rest if implementation allows practical lookup strategy;
- support rotation/revocation;
- rate-limit requests;
- apply per-action cooldowns;
- do not collect guest PII by default;
- validate location/table active state before accepting assist requests.

## Phase 7 — Platform support controls

Platform admin actions require:
- explicit permissions;
- audit events;
- optional support impersonation only if later required and then clearly marked/audited;
- kill-switch/disable controls for organisation/location/device.

## Phase 8 — Secrets and hardening

Implement:
- secrets outside source code;
- TLS everywhere;
- secure refresh token handling;
- CORS allowlists;
- proxy log redaction for sensitive query/header values;
- least-privilege DB roles;
- dependency scanning;
- backup encryption;
- security headers on web surfaces.

## Deliverables

- identity ADR;
- permission matrix;
- RLS policies;
- tenancy regression tests;
- device registration/revocation model;
- QR capability threat model;
- platform admin audit policy.

## Definition of done

Every API route has a documented authentication mode and authorization rule, and automated tests prove cross-tenant access fails.


---

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


---

# 05 — Events, Transactional Outbox, and Workers

## Goal

Create a durable operational history and reliable asynchronous processing path without requiring NATS on day one.

## Dependencies

- `04_BACKEND_API_AND_COMMAND_MODEL.md`

## Phase 1 — Event envelope

Define a canonical event envelope containing:
- event id;
- type;
- schema version;
- occurred_at;
- organisation_id;
- location_id;
- actor/user id when applicable;
- device id when applicable;
- entity type/id;
- entity version;
- command/correlation id;
- event data.

Initial event types:
- table.occupied;
- table.cleared;
- session.opened;
- session.closed;
- assist.requested;
- assist.acknowledged;
- assist.resolved;
- assist.cancelled;
- device.registered/revoked;
- key configuration events where auditability matters.

## Phase 2 — Immutable operational event log

Create `operational_events`.

Rules:
- append-only through application privileges;
- never use event sourcing as current-state storage;
- events reflect committed domain changes;
- event payloads are schema-versioned.

## Phase 3 — Transactional outbox

Create `outbox_records` in the same PostgreSQL database.

Every business transaction that needs downstream delivery writes the event and outbox record before commit.

Fields should support:
- event id;
- destination/topic category;
- created_at;
- available_at;
- attempt count;
- status;
- last_error;
- processed_at.

## Phase 4 — Go outbox worker

Build worker behaviour:
- claim batches safely;
- process with bounded concurrency;
- retry transient failures;
- exponential backoff;
- dead-letter/failed state after policy threshold;
- expose queue depth and oldest-message age;
- safe shutdown.

Initially dispatch directly to in-process/external consumers:
- realtime publisher;
- analytics projector;
- audit/report jobs.

## Phase 5 — Consumer idempotency

Every consumer records or otherwise guarantees idempotent handling by event id.

A worker crash between side effect and acknowledgement must not duplicate business effects.

## Phase 6 — Audit reconstruction

Build API/query support to reconstruct timelines such as:

```text
18:03 table occupied
18:48 bill requested
18:49 assist acknowledged
18:52 assist resolved
19:31 table cleared
```

Use event data plus actor/device metadata.

## Phase 7 — Event schema governance

Rules:
- published schemas are immutable;
- additive compatible change where possible;
- breaking change creates `v2`;
- consumers declare supported versions;
- CI validates representative event fixtures.

## NATS decision point

Do **not** add NATS yet unless:
- multiple independently deployed consumers need durable fan-out;
- direct outbox dispatch becomes operationally awkward;
- replay/consumer isolation is required.

That decision belongs to `15_SCALE_UP_TRIGGERS_AND_FUTURE_ARCHITECTURE.md`.

## Deliverables

- operational event table;
- event JSON schemas;
- transactional outbox;
- worker process;
- retry/dead-letter tooling;
- event/audit query endpoint;
- observability metrics.

## Definition of done

Kill the worker after a command commits, restart it, and prove the event is still delivered exactly once in business effect.


---

# 06 — Realtime and Sync

## Goal

Provide fast live updates while keeping HTTP/PostgreSQL as truth and making reconnect/reconciliation deterministic.

## Dependencies

- `05_EVENTS_OUTBOX_AND_WORKERS.md`

## Phase 1 — Define realtime contract

Realtime messages must describe concrete events, not generic "refresh" instructions.

Example:

```json
{
  "type": "table.occupied",
  "tableId": "...",
  "version": 27,
  "occurredAt": "...",
  "eventId": "..."
}
```

Scopes:
- organisation;
- location;
- floor;
- user;
- device where needed.

## Phase 2 — Go WebSocket gateway

Implement:
- authenticated connection handshake;
- staff/device authorization;
- location-scoped subscriptions;
- ping/pong/heartbeat;
- backpressure policy;
- connection limits;
- graceful restart behaviour;
- structured connection metrics.

Do not put access tokens in URLs if an alternative handshake/header/subprotocol can be safely used by the chosen clients.

## Phase 3 — Durable source to realtime

Realtime delivery consumes committed operational events via outbox dispatch.

Realtime is **not** the source of truth.

If a client misses events, it reconciles through HTTP sync/read APIs.

## Phase 4 — Sync cursor / recovery API

Provide a recovery mechanism such as:
- current location snapshot + versions;
- event cursor/change cursor if needed;
- `updated_since` only if semantics are robust.

On reconnect:
1. authenticate;
2. reconcile local truth with server snapshot/cursor;
3. subscribe to realtime;
4. process only newer versions/events.

## Phase 5 — Conflict model

For offline/client commands:
- submit expectedVersion;
- server returns `version_conflict` with current representation/version;
- client re-evaluates pending commands;
- impossible/stale commands are marked for user resolution only when automatic reconciliation is unsafe.

## Phase 6 — Realtime reliability tests

Test:
- reconnect during service;
- duplicate messages;
- out-of-order messages;
- missed messages;
- two staff devices updating one table;
- one device offline while another acts;
- server restart;
- gateway restart;
- high assist burst.

## Deliverables

- WebSocket protocol document;
- gateway implementation;
- reconnect strategy;
- recovery/snapshot APIs;
- conflict semantics;
- load/reconnect test harness;
- realtime metrics.

## Definition of done

A client can be offline, reconnect after many server changes, and converge to authoritative state without requiring a full app restart or operator intervention.


---

# 07 — Waiter App: Offline-First Flutter Client

## Goal

Build the primary operational client for Android/iOS/tablets so service can continue through intermittent connectivity.

## Dependencies

- `04_BACKEND_API_AND_COMMAND_MODEL.md`
- `06_REALTIME_AND_SYNC.md`

## Phase 1 — Mobile application baseline

Create supported Flutter targets:
- Android first;
- iOS next;
- tablet layouts as first-class;
- desktop only if a later use case justifies it.

Establish layers:

```text
UI
  ↓
Domain/Application
  ↓
Repository
  ├── SQLite/Drift
  └── Sync Engine
       ├── REST
       └── WebSocket
```

## Phase 2 — Local database

Use SQLite + Drift.

Local tables:
- locations/floors/zones/tables;
- table occupancy snapshots;
- active sessions needed by UI;
- assist requests;
- pending commands;
- sync metadata/cursor;
- device configuration.

Rule:

> UI reads local database; network updates the local database.

Do not make screens depend directly on the last HTTP response.

## Phase 3 — Authentication and trusted device

Implement:
- OIDC login;
- secure credential storage;
- device registration;
- biometric/local PIN quick unlock;
- relock on appropriate lifecycle transitions;
- revoked-device handling.

## Phase 4 — Read-only floor operation

Before mutations, implement:
- floor/zone navigation;
- table geometry;
- occupancy indicators;
- assist indicators separate from occupancy;
- offline/stale-state banner;
- last successful sync timestamp.

## Phase 5 — Offline command queue

Represent every mutation as a command:
- command id;
- type;
- entity id;
- payload;
- expected version;
- created_at;
- retry count;
- state.

Initial command types:
- OCCUPY_TABLE;
- CLEAR_TABLE;
- ACKNOWLEDGE_ASSIST;
- RESOLVE_ASSIST;
- CANCEL_ASSIST where role permits.

Flow:

```text
user action
 → optimistic local transaction
 → queue command
 → UI updates
 → sync attempt
 → server result
 → local reconciliation
```

## Phase 6 — Conflict resolution UX

Most conflicts should resolve automatically.

Examples:
- local occupy, server already occupied → treat as converged when same resulting intent is acceptable;
- local clear, newer server session exists → do not silently clear; refresh and surface a targeted conflict;
- assist already resolved elsewhere → mark command complete and update local state.

Never display generic "sync failed" if the client can explain the real conflict.

## Phase 7 — Realtime integration

Process realtime events into SQLite.

Rules:
- ignore older entity versions;
- deduplicate by event id;
- do not overwrite a pending optimistic state blindly;
- reconcile when remote event intersects a local pending command.

## Phase 8 — Service-focused UX

Optimize for:
- few taps;
- clear table state at a glance;
- assist priority/age;
- floor/zone filtering;
- large touch targets;
- tablet landscape/portrait;
- degraded network visibility without panic-inducing noise.

## Phase 9 — Operational hardening

Test on physical devices:
- Wi-Fi loss;
- captive portal;
- background/foreground;
- app kill/restart;
- low battery;
- token refresh;
- long service period;
- clock skew;
- multiple devices;
- app upgrade with queued commands.

## Deliverables

- Flutter waiter app;
- Drift schema/migrations;
- sync engine;
- offline queue;
- conflict UX;
- device quick unlock;
- physical-device QA matrix;
- release build pipeline.

## Definition of done

A waiter can perform core operations while offline, restart the app, regain connectivity, and safely converge with actions taken by other devices.


---

# 08 — Owner and Platform Web

## Goal

Build a responsive Next.js management surface for venue configuration, operational oversight, analytics, and Seatd platform administration.

## Dependencies

- `04_BACKEND_API_AND_COMMAND_MODEL.md`
- `03_IDENTITY_TENANCY_AND_SECURITY.md`
- basic device APIs from `10_DISPLAY_KIOSK_AND_DEVICE_MANAGEMENT.md` can be developed in parallel.

## Phase 1 — Web shell and access control

Create one Next.js codebase with clearly separated route areas:
- owner/manager;
- platform admin;
- shared design system/components.

Implement:
- OIDC session handling;
- role-aware navigation;
- organisation/location switcher;
- protected server/client routes;
- generated TypeScript API client.

## Phase 2 — Organisation and location configuration

Owner workflows:
- organisation profile;
- location profile;
- location operating configuration;
- enable/disable relevant features;
- location switcher for groups.

Platform workflows:
- create/disable organisations;
- create locations;
- support metadata;
- subscription/status placeholders only as needed.

## Phase 3 — Floor, zone, and table management

Build the visual editor:
- create/edit floors;
- upload background asset;
- add/edit zones;
- add tables;
- drag/resize;
- shape/capacity;
- zone assignment;
- soft delete/restore where supported;
- preview live floor.

Persist edits through explicit configuration endpoints.

## Phase 4 — Staff and permissions

Build:
- invite/create staff according to identity provider flow;
- role assignment;
- location assignment;
- deactivate access;
- audit access changes.

Remove routine dependence on provisioning scripts.

## Phase 5 — QR and guest action configuration

Build:
- guest action CRUD;
- ordering/enabled state;
- QR token regeneration;
- printable/exportable QR assets;
- compromised-token revocation workflow;
- preview guest page.

## Phase 6 — Device management

Build:
- registered device list;
- type/platform/app version;
- last seen;
- location assignment;
- revoke;
- display pairing workflow;
- device health indicators.

## Phase 7 — Service periods

Build first-class configuration for:
- Breakfast;
- Lunch;
- Dinner;
- custom recurring periods;
- special named periods where required.

Service periods feed analytics rather than being UI-only labels.

## Phase 8 — Operational and analytics views

Add:
- current live floor read view;
- assist status;
- current session duration;
- analytics dashboards once `11_ANALYTICS_AND_REPORTING.md` is ready.

## Phase 9 — Platform support console

Seatd operator functions:
- organisations/locations;
- status/disable;
- device status;
- feature flags if introduced;
- integration health;
- audit trail;
- system health links.

## Deliverables

- Next.js owner/admin app;
- floor editor;
- staff management;
- QR configuration/export;
- device management;
- service period management;
- platform support console.

## Definition of done

A new venue can be configured from a browser from organisation creation through a usable floor, staff access, QR setup, and display pairing without direct database intervention.


---

# 09 — Guest QR Experience

## Goal

Deliver an extremely fast, no-account guest service-request experience with minimal data collection and strong abuse controls.

## Dependencies

- `04_BACKEND_API_AND_COMMAND_MODEL.md`
- assist domain from `02_CORE_DOMAIN_AND_DATABASE.md`

## Phase 1 — Define guest contract

Guest capabilities:
- identify table through opaque QR capability;
- view enabled guest actions;
- request assistance;
- see request accepted/current state;
- optionally cancel if product rules permit.

No guest account.
No guest profile.
No guest PII unless a future explicit requirement is approved.

## Phase 2 — QR token lifecycle

Implement:
- high-entropy tokens;
- table/location validation;
- token rotation;
- revocation;
- disabled location checks;
- soft-deleted table checks;
- QR export link generation.

Decide whether tokens remain static per physical sticker or support a versioned indirection mechanism for easier rotation.

## Phase 3 — Assist request semantics

Unlike the current implementation, guest assist logic must not alter occupancy.

Rules:
- table must be eligible according to product policy, normally occupied;
- duplicate pending requests for same action can be idempotently reused;
- request transitions pending → acknowledged → resolved/cancelled;
- the table remains occupied until explicitly cleared by staff/integration.

## Phase 4 — Lightweight frontend

Optimize for:
- first meaningful render speed;
- minimal JavaScript;
- mobile browser compatibility;
- clear large action buttons;
- poor network behaviour;
- accessibility;
- no login friction.

Host portably on a static/edge platform or Seatd infrastructure.

## Phase 5 — Status updates

Start simple:
- HTTP polling with sensible interval/backoff is acceptable;
- do not add guest WebSockets unless there is a demonstrated UX need.

Clearly show:
- request sent;
- acknowledged;
- resolved.

## Phase 6 — Abuse prevention

Implement:
- token entropy;
- IP/token/action rate limits;
- request cooldowns;
- idempotency;
- suspicious-volume telemetry;
- revocation;
- no sensitive data in URLs beyond the QR capability path design already intentionally exposed to the scanner/user.

## Phase 7 — Guest action configuration integration

Load enabled, sorted guest actions from location/organisation configuration.

Examples:
- Call waiter;
- Request bill;
- Request water;
- Request service.

Keep the schema generic enough for configurable labels without turning guest actions into arbitrary workflow automation.

## Deliverables

- guest web application;
- public lookup/request/status APIs;
- QR token lifecycle;
- rate limiting/cooldowns;
- accessibility and performance budget;
- abuse telemetry.

## Definition of done

A guest can scan, request service, and see resolution on an unreliable mobile connection without creating an account or affecting table occupancy.


---

# 10 — Display/Kiosk and Device Management

## Goal

Build a controlled Android display experience and make every operational device visible, manageable, and revocable.

## Dependencies

- `03_IDENTITY_TENANCY_AND_SECURITY.md`
- `06_REALTIME_AND_SYNC.md`

## Phase 1 — Device domain/API

Implement first-class Device records and APIs for:
- registration;
- pairing;
- assignment;
- heartbeat/last seen;
- app version;
- revocation;
- device type;
- capabilities/configuration.

## Phase 2 — Display pairing

Replace incomplete/manual pairing flow with a complete UX:

```text
Owner generates short-lived pairing code
  → display enters/scans code
  → backend validates location
  → backend issues revocable device credential
  → display persists credential securely
```

Pairing codes must expire and be single-use unless a deliberate alternative is documented.

## Phase 3 — Flutter Android display baseline

Standardize on one known Android device profile initially.

Implement:
- launch on boot;
- kiosk/lock task where deployment mode permits;
- orientation control;
- screen wake;
- secure device credential storage;
- automatic reconnect;
- app version reporting.

## Phase 4 — Local cache and offline display

Cache:
- restaurant/location info;
- floors/zones/tables;
- last-known occupancy/assist state;
- rotation/display configuration.

On disconnect:
- show a subtle offline/stale indicator;
- keep last-known layout;
- never pretend stale data is current;
- continue retrying safely.

## Phase 5 — Live display views

Implement:
- availability summary;
- floor plan;
- configurable rotation;
- table capacity labels;
- occupancy visual state;
- optional assist indicator only if appropriate for public display policy.

## Phase 6 — Fleet management

Owner/platform web should show:
- online/offline based on heartbeat threshold;
- last seen;
- version;
- location;
- credential state;
- revoke button;
- pairing/re-pairing workflow.

## Phase 7 — Update strategy

Initially support controlled application release process rather than building a custom updater.

Document:
- staged APK/managed distribution method;
- minimum supported app version;
- forced upgrade behaviour only if necessary;
- rollback.

## Deliverables

- device APIs;
- secure pairing;
- Flutter Android display app;
- offline cache;
- heartbeat/device health;
- owner/platform device management UI;
- deployment runbook.

## Definition of done

A new display can be paired by a venue owner without backend intervention, continue showing clearly marked last-known data offline, and be remotely revoked.


---

# 11 — Analytics and Reporting

## Goal

Turn operational events and TableSessions into trustworthy venue intelligence without introducing a warehouse prematurely.

## Dependencies

- `05_EVENTS_OUTBOX_AND_WORKERS.md`
- `02_CORE_DOMAIN_AND_DATABASE.md`
- service periods configured through owner platform.

## Phase 1 — Metric definitions before dashboards

Define exact formulas and exclusions for:
- occupancy duration;
- utilisation;
- table turnover count/rate;
- median/percentile session duration;
- assist response time;
- assist resolution time;
- request volume by action;
- zone/floor utilisation;
- service-period utilisation;
- device/realtime health where product-facing.

Document how reopened/corrected sessions and anomalous data are handled.

## Phase 2 — Service periods

Model service periods as first-class configuration linked to location/timezone.

Support:
- recurring day/time windows;
- named periods;
- overnight edge cases if required;
- timezone/DST correctness.

## Phase 3 — PostgreSQL projections

Create projection tables such as:
- hourly_table_metrics;
- daily_table_metrics;
- service_period_metrics;
- daily_zone_metrics;
- daily_floor_metrics;
- daily_location_metrics.

Project from operational events/session truth with Go workers.

## Phase 4 — Rebuildability

Every projection must be rebuildable from authoritative source records/events.

Provide:
- projector version;
- checkpoint/cursor;
- rebuild command;
- shadow/rebuild table strategy for large recalculations.

## Phase 5 — Owner dashboards

Deliver in value order:
1. today/current service summary;
2. utilisation over time;
3. turnover/session duration;
4. assist response/resolution;
5. zone/floor comparison;
6. service-period comparison;
7. multi-location comparison when groups exist.

## Phase 6 — Reports

Add:
- daily summary;
- weekly summary;
- exportable CSV/PDF only when useful;
- scheduled delivery only after notification/email infrastructure is intentionally added.

## Phase 7 — Data quality monitoring

Track:
- sessions open unusually long;
- impossible state transitions;
- event/projector lag;
- missing projection ranges;
- POS mismatch once integrations exist;
- duplicate/replayed event handling.

Treat analytics correctness as a production concern.

## Phase 8 — Advanced intelligence, later

Only after enough reliable history exists:
- benchmarks;
- recommendations;
- anomaly detection;
- forecasting.

Do not label weak heuristics as AI simply to add a feature category.

## ClickHouse gate

Stay on PostgreSQL until measured query volume/latency/storage patterns justify ClickHouse.

## Deliverables

- metric specification;
- service-period model;
- projection worker;
- projection tables;
- rebuild tooling;
- dashboards;
- data-quality monitors;
- report exports as justified.

## Definition of done

Any dashboard number can be traced to a documented formula and reconstructed from authoritative events/sessions.


---

# 12 — Integrations and Reconciliation

## Goal

Integrate POS and external systems through canonical adapters while keeping vendor-specific logic out of Seatd's core domain.

## Dependencies

- stable domain/API/event model;
- `05_EVENTS_OUTBOX_AND_WORKERS.md`;
- device/observability foundations.

## Phase 1 — Canonical integration contract

Define an adapter interface around capabilities, not vendors.

Responsibilities may include:
- HandleWebhook;
- ResolveTableMapping;
- FetchExternalStatus;
- CheckHealth;
- Reconcile;
- translate vendor records/events into canonical Seatd commands or facts.

Core domain code must never branch on vendor names.

## Phase 2 — Integration configuration model

Create:
- integration account/connection;
- location assignment;
- credential reference;
- status;
- last successful sync;
- last error;
- external mappings.

Mappings include at least:

```text
external table id ↔ Seatd table id
```

## Phase 3 — Webhook inbox

Persist incoming webhook envelopes before processing.

Store:
- vendor;
- external event id;
- received_at;
- signature validation outcome;
- payload or secure reference;
- processing state;
- attempts;
- last error.

Benefits:
- idempotency;
- replay;
- debugging;
- outage recovery.

## Phase 4 — First adapter

Choose one commercially valuable POS/integration target, not many simultaneously.

Implement:
- credential setup;
- table mapping;
- webhook processing;
- outbound fetch if required;
- health endpoint;
- retry policy;
- reconciliation job.

## Phase 5 — Authority rules

For every integrated field/state, document which system is authoritative.

Example questions:
- Does POS seating open a Seatd TableSession?
- Can a waiter manually override a POS-derived occupancy state?
- What happens if POS says closed while Seatd has an unresolved assist?

Do not allow two systems to race without explicit conflict policy.

## Phase 6 — Reconciliation

Build periodic reconciliation:
- fetch vendor truth;
- compare mappings/state;
- produce discrepancy records;
- auto-correct only safe mismatches;
- flag ambiguous mismatches for operator review;
- emit integration health events/metrics.

## Phase 7 — Retry and workflow escalation

Use Go workers first.

Only consider Temporal when workflows become genuinely long-running with waits, retries, compensations, and external outage handling that is difficult to maintain reliably in normal workers.

## Phase 8 — Integration health UI

Owner/platform surfaces should show:
- connected/disconnected;
- last successful event/sync;
- webhook failures;
- unmapped tables;
- reconciliation mismatches;
- retry/replay actions for authorized operators.

## Deliverables

- canonical adapter interfaces;
- integration data model;
- webhook inbox;
- first production adapter;
- mapping UI;
- reconciliation worker;
- integration health UI;
- runbook.

## Definition of done

A vendor outage, duplicate webhook, or delayed event cannot silently corrupt Seatd floor state, and operators can explain/reconcile mismatches.


---

# 13 — Infrastructure, CI/CD, Observability, and Operations

## Goal

Run Seatd reliably on portable infrastructure without making AWS or Kubernetes a prerequisite.

## Dependencies

Can start during foundation work and mature alongside every phase.

## Phase 1 — Local/development infrastructure

Use Docker Compose for:
- PostgreSQL;
- API;
- worker;
- realtime;
- web/guest where useful;
- object storage when introduced;
- observability dependencies as practical.

Create repeatable local setup and fixture loading.

## Phase 2 — Initial staging

Suggested topology:

```text
Caddy
  ├── API
  ├── Realtime
  └── Web apps
       ↓
PostgreSQL
Worker
S3-compatible object storage
```

Use one application VPS if appropriate, with a managed or separately backed-up PostgreSQL option.

## Phase 3 — Object storage

Move floor backgrounds, logos, QR exports, reports, and other files to S3-compatible storage.

Options:
- MinIO when self-hosting;
- any compatible managed provider.

Do not keep production assets on an ephemeral application filesystem.

## Phase 4 — CI/CD

GitHub Actions pipeline:

```text
push
 → lint
 → unit tests
 → integration tests
 → build images/apps
 → vulnerability scan
 → push image
 → deploy staging
 → migrations
 → smoke tests
 → controlled production promotion
```

Use immutable image tags tied to commit/version.

## Phase 5 — Reverse proxy/TLS

Use Caddy initially for:
- automatic TLS;
- routing;
- WebSocket proxying;
- compression/static policies where appropriate;
- security headers.

Redact sensitive headers/query parameters from logs.

## Phase 6 — Backups and recovery

Minimum production posture:
- automated PostgreSQL backups;
- off-site copy;
- encrypted backup storage;
- point-in-time recovery where provider/setup supports it;
- object storage backup/versioning as needed;
- restore drills.

Measure recovery objectives:
- RPO;
- RTO.

A backup process is not complete until restore has been tested.

## Phase 7 — OpenTelemetry baseline

Instrument:
- HTTP requests;
- database operations;
- command handling;
- worker jobs;
- outbox lag;
- realtime publish/delivery;
- integration processing;
- key mobile/web errors.

Propagate:
- trace_id;
- request_id;
- command_id;
- event_id;
- organisation/location ids;
- device id where relevant.

## Phase 8 — Grafana stack

Portable target:
- OpenTelemetry Collector;
- Prometheus-compatible metrics;
- Grafana;
- Loki;
- Tempo;
- Sentry-compatible crash/error reporting for web/mobile if desired.

## Phase 9 — Product-specific operational truth monitoring

Monitor more than servers:
- stale floor state;
- offline command backlog;
- oldest pending outbox record;
- realtime delivery latency;
- unusually long sessions;
- impossible transitions;
- inactive/revoked device attempts;
- assist delivery latency;
- integration mismatches.

**Floor truth quality is a production metric.**

## Phase 10 — Production pilot topology

Move to:
- application server(s);
- dedicated/managed PostgreSQL;
- off-site backups;
- central monitoring;
- object storage;
- separate worker deployment if load warrants.

No AWS-specific service is required.

## Deliverables

- Compose environments;
- staging/production deployment manifests;
- GitHub Actions workflows;
- Caddy config;
- backup/restore runbook;
- OTel instrumentation;
- dashboards/alerts;
- secrets handling guide;
- disaster recovery test record.

## Definition of done

A failed deploy can be rolled back, a database can be restored from backup, and on-call/support can distinguish infrastructure health from incorrect digital-floor truth.


---

# 14 — Testing, Migration, and Cutover

## Goal

Move from the current Bun/Express/PostgreSQL + React/Vite + Flutter desktop implementation to the target architecture without losing venue data or breaking operational service.

## Dependencies

Runs across all build phases; final cutover depends on the target operational wedge being complete.

## Phase 1 — Freeze the behavioural baseline

Document current required behaviours before changing architecture:
- staff login/refresh;
- owner floor/table editing;
- waiter occupancy changes;
- guest QR request/status;
- kiosk pairing/read/realtime;
- platform admin provisioning/disable;
- QR PDF/export behaviour;
- current data shapes.

Classify each as:
- preserve;
- intentionally change;
- remove.

## Phase 2 — Explicit intentional changes

Mark these as migration changes, not regressions:
- `restaurant` becomes Organisation + Location;
- add Zone;
- `attention` removed as occupancy state;
- assist lifecycle gains acknowledged/cancelled states;
- assist resolution does **not** free table;
- table `toggle` replaced by occupy/clear;
- TableSession added;
- device model strengthened;
- auth may move to OIDC;
- assets move off local filesystem;
- generated API contracts replace manual duplication.

## Phase 3 — Automated test pyramid

### Unit tests

Domain invariants and pure logic.

### Database tests

Transactions, constraints, RLS, query behaviour.

### API integration tests

Commands, idempotency, version conflicts, auth.

### Worker tests

Outbox retries, duplicate events, projector idempotency.

### Client tests

Flutter Drift/sync and web critical flows.

### End-to-end tests

Complete service wedge against real Postgres and built clients where feasible.

## Phase 4 — Legacy data inventory

Inventory tables/fields:
- restaurants;
- floors;
- tables;
- table_states;
- assist_requests;
- guest_actions;
- staff_users;
- platform_admins;
- display tokens/pairing;
- file assets;
- audit data.

For each field define:
- target field;
- transformation;
- default;
- unsupported/archived handling.

## Phase 5 — Migration tooling

Build repeatable, idempotent migration jobs/scripts.

Recommended approach:
1. create Organisation per legacy restaurant;
2. create primary Location;
3. migrate floors;
4. create default Zone per floor, then optionally allow later manual refinement;
5. migrate tables/layout;
6. convert current `available/occupied` directly;
7. for current `attention`, set occupancy based on safest source/history and migrate pending assists independently; if source data cannot prove occupancy, flag for review rather than guessing silently;
8. migrate staff identities/issue new identity links;
9. migrate guest actions/QR tokens where secure/compatible;
10. migrate device relationships;
11. move assets to object storage.

## Phase 6 — Historical session strategy

The current system does not have first-class TableSession history.

Do not fabricate historical sessions unless timestamps/events support reconstruction confidently.

Options:
- start TableSession analytics from target cutover date;
- reconstruct only provable sessions;
- label migrated/reconstructed data quality explicitly.

## Phase 7 — Shadow validation

Before venue cutover:
- copy production-like data to staging;
- run migration;
- compare counts and relationships;
- run tenant-isolation checks;
- run live-flow simulations;
- inspect floor visual parity;
- verify QR links;
- verify user access;
- verify device pairing.

## Phase 8 — Controlled pilot cutover

Choose one internal/demo/low-risk pilot venue first.

Cutover checklist:
- backup legacy DB/assets;
- stop or quiesce legacy writes for the migration window where necessary;
- run final migration;
- validate counts;
- switch API/client configuration;
- smoke test occupy/assist/resolve/clear;
- monitor realtime/outbox/device health;
- maintain rollback plan.

## Phase 9 — Parallel/compatibility period

Where practical:
- support old QR URLs with redirect/adapter;
- maintain API compatibility only for the minimum migration window;
- do not run two authoritative writers for the same state without explicit sync design.

## Phase 10 — Rollout and decommission

Roll venues in batches.

After all supported venues move:
- archive legacy database securely;
- disable legacy write paths;
- remove compatibility endpoints;
- remove stale TableReady naming/env variables;
- update runbooks/docs;
- close migration feature flags.

## Acceptance suite: service wedge

Must prove:
1. occupy available table;
2. session opens;
3. guest requests help;
4. waiter receives event;
5. waiter acknowledges;
6. waiter resolves;
7. table remains occupied;
8. waiter clears table;
9. session closes;
10. all clients converge;
11. history is reconstructable;
12. duplicate/retried commands do not duplicate effects.

## Deliverables

- behaviour matrix;
- migration mapping;
- migration scripts;
- automated test suite;
- staging validation report template;
- cutover/rollback runbook;
- legacy decommission checklist.

## Definition of done

A venue can be migrated reproducibly with verified data, exercised through the full service wedge, and rolled back according to a tested procedure if a release-level failure occurs.


---

# 15 — Scale-Up Triggers and Future Architecture

## Goal

Prevent premature infrastructure complexity by defining measurable conditions for introducing additional systems.

## Dependencies

All previous architecture can run without these components.

## Principle

> Add infrastructure because a current measurable problem requires it, not because future scale is imaginable.

## 1. NATS JetStream

### Do not add when
- one worker can dispatch outbox events directly;
- consumers are few and co-deployed;
- replay/fan-out requirements are simple.

### Add when
- multiple independently deployed consumers require durable fan-out;
- analytics, realtime, integrations, alerts, reporting need isolated consumption;
- consumer replay and independent checkpoints are operationally valuable;
- direct outbox routing has become a bottleneck/maintenance problem.

### Migration shape

```text
Postgres transaction
 → outbox
 → publisher
 → NATS JetStream
    ├── realtime consumer
    ├── analytics consumer
    ├── integration consumer
    └── alert/report consumers
```

The transactional outbox remains the bridge from database commit to message publication.

## 2. Valkey/Redis

### Add when
- shared rate limiting cannot remain local;
- multiple realtime nodes need presence/coordination;
- short-lived distributed locks/pairing state become necessary;
- repeated hot cache workloads materially burden PostgreSQL.

### Never use as
- authoritative table occupancy;
- authoritative assist state;
- only copy of operational data.

## 3. ClickHouse

### Add when
- PostgreSQL analytics has measured latency/resource impact despite projections/indexing;
- event history becomes very large;
- cross-location analytical scans materially affect operational DB performance;
- analytical concurrency/retention needs exceed the practical Postgres design.

Before adding ClickHouse, prove:
- slow-query evidence;
- workload separation need;
- expected retention/volume;
- operational capability to maintain another datastore.

## 4. Temporal

### Add when
- integrations contain long-running workflows with waits/retries/compensations;
- vendors can be unavailable for extended periods;
- workflow state is hard to reason about in ordinary jobs;
- operators need durable workflow visibility/retry controls.

Do not use Temporal for simple outbox jobs or short retries.

## 5. Kubernetes

### Add when
- Seatd operates across enough nodes/services that Compose/manual scheduling is a material operations burden;
- automatic rescheduling/autoscaling is needed;
- deployments are frequent across many independent services;
- the team has capacity to operate Kubernetes correctly.

Before Kubernetes, consider whether a simpler managed container platform solves the actual problem.

## 6. Multi-region

### Consider only when
- customer geography and latency require it;
- availability objectives justify complexity;
- data residency requires it;
- disaster recovery targets cannot be met more simply.

Operational consistency for live floor state should be prioritized over theoretical active-active architecture.

## 7. Cloud-provider-specific services

AWS/Azure/GCP-specific services are acceptable only when:
- there is a concrete feature/reliability/operational advantage;
- cost is understood;
- portability impact is documented in an ADR;
- an exit path is known where practical.

## 8. Scale metrics to collect from the start

Collect enough evidence to make these decisions later:
- API request rate/latency;
- active WebSocket connections;
- events/sec;
- outbox depth and age;
- worker throughput;
- DB CPU/IO/connection usage;
- top query latency;
- projection lag;
- analytics query latency;
- storage growth;
- integration webhook rate/failure rate;
- offline command backlog per location;
- number of organisations/locations/devices.

## 9. Architectural review cadence

Review scale decisions at meaningful milestones, for example:
- 10 production locations;
- 50 locations;
- 100+ locations;
- major integration launch;
- sustained reliability SLO breach;
- measured database/worker bottleneck.

Do not tie infrastructure changes to arbitrary calendar dates.

## Definition of done

The team can answer "why are we adding this technology now?" with production measurements, a documented problem, expected benefit, cost, and rollback/exit implications.


---
