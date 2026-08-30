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
