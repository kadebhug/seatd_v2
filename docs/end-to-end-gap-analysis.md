# Seatd V2 End-to-End Gap Analysis

Date: 2026-09-01

## Executive Summary

Seatd v2 has a strong backend foundation, but it is not yet production-ready end
to end. The repository already contains PostgreSQL migrations, tenant isolation
through row-level security, core operations commands, QR guest assistance,
device pairing, analytics projection code, integration reconciliation, generated
clients, and multiple application shells.

The main missing work is the production shell around that foundation:

- CI must be made green and deterministic.
- User authentication must move from development headers and demo sessions to a
  real OIDC/session boundary.
- Authorization must be enforced consistently at the API edge and in domain
  services.
- Frontend and mobile workflows need to be hardened from functional skeletons
  into production workflows.
- Production deployment, observability, incident response, backups, and secret
  management need to be added.

Until those gaps are closed, the system is best treated as an advanced local or
staging prototype, not a system ready for live venue operations.

## Priority Key

- `P0`: Blocks reliable development, CI, or safe production access.
- `P1`: Required before production rollout.
- `P2`: Needed for operational maturity or complete product workflows.
- `P3`: Useful hardening, cleanup, or future improvement.

<!-- ### Owner Web Login UI Is Missing

What is missing:

The web frontend needs a visible sign-in entry point for owner/platform users,
such as a dedicated login page and/or a clear sign-in action from the public
landing page.

Why it matters:

The OIDC routes can authenticate users, but operators still need an obvious UI
path to begin sign-in. Relying only on protected-route redirects makes the
control plane harder to discover, test, and support.

Evidence:

- The auth flow is route-driven through `/api/auth/login` and
  `/api/auth/callback`.
- The app shell includes logout once signed in, but the public frontend does
  not expose a polished login action.

Recommended next action:

Add a small owner/platform login UI:

- visible sign-in link or button on the public web app;
- optional `/login` page that starts OIDC with a safe `returnTo`;
- authenticated-user redirect away from login when a valid session already
  exists;
- error and retry states that link back into the OIDC flow.

Priority: `P1` -->

<!-- ## P1: Make Authorization Consistent -->

<!-- ### Read Endpoints Bypass Actor Permission Checks

What is missing:

Every tenant data read endpoint needs an explicit actor permission check, not
just tenant RLS.

Why it matters:

RLS can prevent cross-tenant leakage while still allowing an actor inside a
tenant to read resources they should not access. The API authorization document
distinguishes `layout.read`, `operations.read`, `audit.read`,
`analytics.read`, and other permissions, so handlers must enforce those
permissions consistently.

Evidence:

- Some handlers call domain services that enforce permissions, such as
  configuration, analytics, and integrations services.
- Other handlers perform direct query reads inside `api.inTenantTx`, including
  location state, floor lists, table lists, active assists, audit timeline, and
  membership listing.
- `docs/api-authz-rules.md` requires explicit authz by surface.

Recommended next action:

Move direct-read handlers behind domain service methods or shared authorization
helpers that require the correct permission before querying:

- layout reads require `layout.read`;
- operations reads require `operations.read`;
- audit timeline requires `audit.read`;
- membership reads require an organisation management or audit-style permission;
- device reads require `device.manage` or an explicitly documented read
  permission.

Priority: `P1` -->

<!-- ### Device Management Lacks Full Actor Enforcement

What is missing:

Device listing, pairing-code creation, and revocation need verified user
authorization, not only tenant context.

Why it matters:

Device credentials grant operational access to displays and realtime surfaces.
Weak device management authorization can let an attacker pair or revoke devices
inside a tenant.

Evidence:

- `docs/api-authz-rules.md` says device registration, trust, and revocation
  require `device.manage`.
- `internal/domain/identity/service.go` has device credential and pairing logic,
  but API handlers rely on request context and call identity methods directly.

Recommended next action:

Require `device.manage` before listing devices, creating pairing codes, and
revoking devices. Add tests for actor without permission, actor with location
permission, actor with organisation permission, and cross-location attempts.

Priority: `P1` -->

<!-- ### Platform Admin Surface Is Not Implemented

What is missing:

The platform admin product surface and server-side platform permissions need to
be implemented.

Why it matters:

Platform administration controls support, tenant-level operations, and
cross-tenant access. Without a concrete implementation, production support
either cannot function or will happen through unsafe ad hoc database access.

Evidence:

- `apps/web/app/platform/page.tsx` says the area is reserved for the next
  platform phase.
- `docs/platform-admin-audit-policy.md` describes audit policy expectations.
- `docs/api-authz-rules.md` requires `platform.admin` and audit events for
  platform controls.

Recommended next action:

Implement a minimal platform admin v1:

- tenant search/read-only overview;
- support-safe tenant diagnostics;
- explicit platform permission enforcement;
- audit events for every platform action;
- no impersonation unless separately designed and visibly audited.

Priority: `P1` -->

## P1: Complete Production Operations

### Production Deployment Is Missing

What is missing:

The repository needs production deployment definitions for API, worker,
realtime, web, guest, display artifact delivery, and database migrations.

Why it matters:

Local compose is not a deployable system. Live venue operations require
repeatable deployments, rollback, health checks, secrets, database migration
ordering, and service startup dependencies.

Evidence:

- `infra/compose.yml` only defines a local Postgres service.
- `ops/README.md` only reserves space for operational tooling.
- There are no environment-specific deployment manifests or release scripts.

Recommended next action:

Add a deployment baseline for the chosen target platform:

- API, worker, realtime service definitions;
- web and guest hosting configuration;
- managed Postgres connection and migration job;
- environment variable contract;
- rollback procedure;
- health/readiness checks;
- release artifact versioning.

Priority: `P1`

### Backup and Restore Plan Is Missing

What is missing:

Database backup, restore, retention, and restore-test procedures need to be
defined.

Why it matters:

Seatd stores venue layouts, table sessions, QR capabilities, device credentials,
audit data, analytics, and integration records. Data loss or failed recovery
would directly affect venue operations and auditability.

Evidence:

- There are migration and seed scripts under `infra/database`, but no backup or
  restore runbook under `ops`.

Recommended next action:

Add an operational runbook covering:

- backup frequency and retention;
- encryption requirements;
- restore drill cadence;
- recovery time and recovery point objectives;
- tenant-scoped export support if needed;
- who can initiate restores.

Priority: `P1`

### Secret Management Is Only Documented, Not Wired

What is missing:

Runtime secret injection and production secret references need to be configured.

Why it matters:

The system handles database credentials, OIDC client secrets, webhook secrets,
device credentials, and potentially signing keys. The repo states secrets must
come from a secret manager, but there is no concrete integration yet.

Evidence:

- `.env.example` contains safe local defaults and says secrets must be injected.
- `docs/development.md` says staging/production should use the platform secret
  manager.
- No deployment-specific secret references are present.

Recommended next action:

Define secret names and wiring for each environment:

- database URL;
- OIDC issuer/client/client secret;
- session signing/encryption keys;
- integration webhook secrets;
- any outbound provider credentials.

Priority: `P1`

## P1: Add Production Observability

### Metrics and Alerts Are Missing

What is missing:

The API, worker, and realtime services need production metrics and alert rules.

Why it matters:

Without metrics and alerts, the team cannot detect slow APIs, realtime delivery
failure, outbox backlog, database saturation, failing integrations, QR abuse, or
display/device fleet degradation before venues notice.

Evidence:

- `internal/httpkit/server.go` logs HTTP requests but does not expose Prometheus
  metrics.
- `internal/realtime/http.go` exposes an internal JSON metrics endpoint for the
  realtime hub, but it is not Prometheus-compatible and is not documented in the
  OpenAPI contract.
- `internal/outbox/outbox.go` has `Stats`, but no exported metrics endpoint or
  alert wiring.

Recommended next action:

Add Prometheus-style metrics for:

- HTTP request count, latency, and errors by route;
- database pool stats;
- outbox pending count and oldest pending age;
- realtime active connections, drops, and backpressure closes;
- guest QR rate-limit/abuse events;
- device heartbeat freshness;
- integration webhook failures and reconciliation discrepancies.

Add alert rules for the operational failure modes above.

Priority: `P1`

### Distributed Tracing Is Missing

What is missing:

OpenTelemetry tracing should be added across API, worker, realtime, database
operations, outbox processing, and integration reconciliation.

Why it matters:

Seatd has multi-step flows: guest request -> operational event -> outbox ->
realtime/display/waiter update -> analytics/audit projection. Logs alone are
not enough to diagnose latency or dropped delivery through that chain.

Evidence:

- No OpenTelemetry setup is present in the Go services.
- Logs are structured, but there are no trace IDs or spans.

Recommended next action:

Add tracing at service boundaries and meaningful internal operations:

- HTTP middleware spans;
- DB query/transaction spans;
- outbox claim/process spans;
- integration webhook/reconcile spans;
- realtime connection lifecycle spans where useful.

Priority: `P1`

### Dashboards and Runbooks Are Missing

What is missing:

Operational dashboards and incident runbooks need to exist before production.

Why it matters:

When a live floor view or guest QR request fails, operators need fast diagnosis:
is it API, DB, realtime, device credentials, display heartbeat, guest network,
or POS integration?

Evidence:

- `ops/` contains branch protection and integrations notes, but not incident
  response or service dashboards.

Recommended next action:

Create dashboards and runbooks for:

- API availability and latency;
- realtime delivery health;
- outbox health;
- Postgres health;
- display fleet health;
- QR assist success/failure;
- integration webhook and reconciliation health.

Priority: `P1`

## P1: Harden Security

### Header Spoofing Tests Are Missing

What is missing:

Tests need to prove that unauthenticated clients cannot spoof tenant, actor, or
device headers.

Why it matters:

The current design relies heavily on headers while identity is still in
development. When auth middleware is added, regression tests must lock the new
boundary in place.

Evidence:

- API request context comes from `X-Seatd-*` headers.
- Realtime also requires those headers and verifies trusted devices in the DB.

Recommended next action:

Add tests for:

- missing auth rejected;
- invalid user token rejected;
- valid user cannot claim another organisation/location;
- valid device cannot claim another location;
- direct browser/client calls cannot set privileged actor headers;
- platform admin headers are ignored unless backed by verified platform
  identity.

Priority: `P1`

### QR Abuse Controls Need Durable Enforcement

What is missing:

Guest QR abuse controls need durable, tenant-aware enforcement beyond in-memory
rate limiting.

Why it matters:

Guest QR URLs are intentionally public. An attacker can repeatedly request help,
enumerate tokens, or generate noise for staff. In-memory rate limiting resets on
restart and does not coordinate across API instances.

Evidence:

- `internal/httpapi/guest.go` has an in-memory guest rate limiter.
- `guest_abuse_events` exists in migrations, indicating durable abuse evidence
  is expected.

Recommended next action:

Add database or shared-cache based rate limits by token, IP, action, and
location. Use `guest_abuse_events` for reviewable audit trails and add alerting
on abuse spikes.

Priority: `P1`

### Webhook Verification Needs Provider-Ready Hardening

What is missing:

Webhook verification needs production provider policies, replay windows, and
secret rotation.

Why it matters:

Integrations can mutate live table state. Signature verification must protect
against forged events, replayed events, stale secrets, and vendor-specific
payload quirks.

Evidence:

- `ops/integrations-runbook.md` documents the reference POS webhook.
- `internal/domain/integrations/service.go` uses a local fallback
  `reference-pos-local-secret`.
- Production provider-specific secret management is not wired.

Recommended next action:

For each real vendor, define:

- signature algorithm and canonical payload;
- timestamp/replay window;
- secret lookup and rotation;
- duplicate event handling;
- failure visibility;
- manual replay permissions.

Priority: `P1`

### Vulnerability Scanning Needs Reachability Coverage

What is missing:

Go vulnerability scanning and dependency audit results should be enforced in CI.

Why it matters:

CodeQL is useful, but Go dependency vulnerabilities are best checked with
`govulncheck` because it reports reachable vulnerabilities in the actual build.
Node and Flutter dependencies also need routine audit coverage.

Evidence:

- `.github/workflows/scan.yml` exists, but the inspected workflows do not show a
  full `govulncheck ./...` gate in the standard test/lint/build paths.

Recommended next action:

Add or verify CI gates for:

- `govulncheck ./...`;
- `npm audit` with an agreed severity threshold;
- Flutter/Dart dependency audit process;
- scheduled scans, not only PR scans.

Priority: `P1`

## P2: Complete Product Workflows

### Owner Onboarding Is Missing

What is missing:

The owner app needs a first-run flow for organisation, location, floors, zones,
tables, service periods, and initial staff/device setup.

Why it matters:

Seeds make local development work, but a production tenant needs a safe path
from empty database to usable venue operations without manual SQL.

Evidence:

- Local seed data exists in `infra/database/seeds/local.sql`.
- Owner pages assume an existing organisation/location from session context.

Recommended next action:

Add onboarding flows for:

- create organisation;
- create first location;
- configure timezone and service periods;
- create/import floor layout;
- invite first members;
- pair first display or waiter device.

Priority: `P2`

### Member and Role Administration Is Incomplete

What is missing:

The system needs UI and API flows to invite, disable, and manage organisation
and location members.

Why it matters:

Permissions cannot be operationally maintained through seed data or direct SQL.
Venue staff changes are frequent, and stale accounts are a security risk.

Evidence:

- Membership tables and queries exist.
- `GET /v1/memberships` exists.
- There are no obvious create/update/disable membership API handlers or owner
  UI flows.

Recommended next action:

Implement member management:

- invite/link user profile;
- assign organisation or location role;
- disable membership;
- audit membership changes;
- show effective permissions.

Priority: `P2`

### Waiter App Needs Production Device/Auth Flow

What is missing:

The waiter app needs a complete authenticated device/user flow, credential
storage, conflict handling UI, retry states, and production realtime behavior.

Why it matters:

The waiter app is the operational interface for table status and assists. It
must be resilient to spotty venue networks and concurrent staff actions.

Evidence:

- `apps/waiter` has local store, gateway, realtime connector, optimistic command
  handling, and floor UI.
- The app still has skeleton metadata and limited tests.

Recommended next action:

Complete waiter v1:

- login or device pairing flow;
- secure credential storage;
- reconnect and resync UX;
- conflict resolution UI;
- offline queue visibility;
- real API and realtime configuration per environment;
- widget/integration tests for main staff flows.

Priority: `P2`

### Display App Needs Kiosk Hardening

What is missing:

The display app needs production kiosk deployment details and end-to-end fleet
management.

Why it matters:

Displays are venue-facing infrastructure. They need predictable startup,
offline behavior, rotation, stale-state display, device revocation, and version
rollout controls.

Evidence:

- `docs/display-device-runbook.md` describes the target behavior.
- `apps/display` has pairing, heartbeat, snapshot, and cache code.
- Tests are currently stale and CI-breaking.
- Flutter Android templates still contain default TODOs for application ID and
  signing config.

Recommended next action:

Complete display hardening:

- fix tests;
- set real app identifiers;
- configure release signing outside source control;
- add launch-on-boot/kiosk deployment instructions;
- add stale snapshot age display;
- add server-side minimum-version policy only when compatibility requires it.

Priority: `P2`

### Guest QR Flow Needs Production UX and Abuse Review

What is missing:

The guest QR flow should be tested end to end with real QR export URLs, expired
tokens, revoked tokens, duplicate requests, cancellation, and staff resolution.

Why it matters:

The guest QR path is intentionally public and is part of the customer
experience. It must fail gracefully and avoid creating operational noise.

Evidence:

- `apps/guest/src/App.tsx` implements QR lookup, request creation,
  cancellation, polling, and network status states.
- Backend QR capability endpoints exist.
- Durable abuse and multi-instance rate limiting remain incomplete.

Recommended next action:

Add Playwright or equivalent browser tests covering:

- valid token happy path;
- missing token;
- revoked/expired token;
- duplicate active request;
- rate-limited request;
- cancellation;
- staff resolution reflected in guest UI.

Priority: `P2`

## P2: Strengthen Contracts and Clients

### Realtime Contract Is Outside OpenAPI

What is missing:

The realtime WebSocket and metrics surfaces need a contract strategy.

Why it matters:

Generated clients and contract validation cover REST, but realtime clients also
depend on stable message shapes, cursor behavior, reconnect semantics, and auth
headers.

Evidence:

- `packages/api-contract/openapi/seatd.v1.json` includes most REST paths.
- The OpenAPI path list does not include `/v1/realtime` or
  `/v1/realtime/metrics`.
- `docs/realtime-protocol.md` documents realtime behavior separately.

Recommended next action:

Choose one source of truth:

- add AsyncAPI or JSON schema for realtime events and cursors; or
- extend OpenAPI documentation for handshake and metrics while keeping event
  schemas in `packages/event-schema`.

Generate typed clients or shared types from that source.

Priority: `P2`

### Generated Clients Need Usage Coverage

What is missing:

The TypeScript and Dart generated clients need integration tests that prove real
apps use the generated contract correctly.

Why it matters:

Generated clients can compile while app-specific assumptions drift. Contract
tests should catch mismatched request bodies, response shapes, and error codes.

Evidence:

- `packages/typescript-seatd-client` and `packages/dart-seatd-client` exist.
- Frontend apps still perform some local fetch/proxy logic manually.

Recommended next action:

Add contract integration tests:

- generated TypeScript client against an API test server;
- Dart client against an API test server;
- app-level smoke tests for owner, guest, waiter, and display API calls.

Priority: `P2`

## P2: Expand End-to-End Testing

### Integration CI Only Checks Migrations

What is missing:

Integration CI should exercise real API, database, worker, realtime, and web
flows.

Why it matters:

The system is event-driven and multi-process. Unit tests cannot prove that a
guest assist request reaches staff UI, that the outbox updates analytics, or
that realtime clients recover correctly.

Evidence:

- `.github/workflows/integration.yml` starts Postgres and runs
  `./infra/database/scripts/check-migrations`.
- It does not start API, worker, realtime, or app clients.

Recommended next action:

Add integration scenarios for:

- migrate and seed database;
- start API, worker, realtime;
- create/occupy/clear table;
- guest creates assist;
- waiter/display receives update;
- analytics projector updates metrics;
- integration webhook mutates mapped table state;
- authorization failure cases.

Priority: `P2`

### Browser-Level Tests Are Missing

What is missing:

Owner and guest web apps need browser tests for their main flows.

Why it matters:

TypeScript checks prove components compile, not that a user can complete floor
editing, QR export, device pairing, analytics viewing, or guest assistance.

Evidence:

- App package scripts use `tsc --noEmit` as `test`.
- No Playwright test suite is visible for `apps/web` or `apps/guest`.

Recommended next action:

Add Playwright tests for:

- owner dashboard loads seeded data;
- floor editor creates/updates/archive/restores floors, zones, and tables;
- device pairing code generation;
- integration mapping update and replay;
- guest request lifecycle;
- error/empty states.

Priority: `P2`

## P2: Improve Analytics and Integrations Maturity

### Analytics Needs Data Freshness and Backfill Controls

What is missing:

Analytics needs operator-visible freshness, rebuild status, and backfill safety.

Why it matters:

Analytics based on projections can lag or fail. Owners need to know whether
numbers are current, stale, incomplete, or being rebuilt.

Evidence:

- Analytics projection code and API endpoints exist.
- Owner analytics UI shows empty-state messaging when no projected metrics
  exist.
- There is a rebuild endpoint returning `accepted`, but no visible job lifecycle
  or alerting.

Recommended next action:

Add:

- projector checkpoint visibility;
- freshness indicators in the owner UI;
- rebuild job status;
- alerts for projector lag;
- tests for late events and backfills.

Priority: `P2`

### Integrations Need Real Vendor Adapters

What is missing:

The integration framework needs production vendor adapters beyond the reference
POS behavior.

Why it matters:

Seatd's integration value depends on reliable mapping between external POS
state and internal table state. A reference adapter proves architecture but not
real-world compatibility.

Evidence:

- `ops/integrations-runbook.md` documents a `reference_pos` webhook.
- `internal/domain/integrations` has reconciliation and webhook handling.
- Real provider configuration, credentials, and adapter-specific behavior are
  not present.

Recommended next action:

Pick the first real integration target and implement:

- credential model;
- signature verification;
- webhook parser;
- reconciliation pull API if supported;
- table mapping workflow;
- sandbox tests using provider fixtures.

Priority: `P2`

## P3: Repository Hygiene

### Generated and Build Artifacts Are Present Locally

What is missing:

The working tree should stay clean of generated build artifacts unless they are
intentionally committed outputs.

Why it matters:

Large generated directories make audits noisy, slow down searches, and increase
the chance of accidentally reviewing or committing irrelevant files.

Evidence:

- Local tree contains `node_modules`, `.next`, Flutter `.dart_tool`, Flutter
  `build`, app `dist`, and generated client `dist` outputs.
- `.gitignore` ignores many of these paths, but they are still present in the
  workspace.

Recommended next action:

Run cleanup only when safe for the developer environment:

- remove ignored local build outputs;
- confirm no generated committed artifacts are required;
- ensure `make clean` covers all expected build outputs.

Priority: `P3`

### CODEOWNERS and Team Handles Need Real Values

What is missing:

Ownership placeholders need to be mapped to real repository teams.

Why it matters:

The docs say ownership review is part of the governance model, but placeholder
teams cannot enforce review quality or accountability.

Evidence:

- `docs/development.md` says placeholder team handles must be replaced before
  ownership checks are enforced.
- `docs/ownership.md` also notes placeholder team handles.

Recommended next action:

Replace placeholder owners with real Git host teams and enable code owner review
in branch protection.

Priority: `P3`

### Next Metadata Warning

What is missing:

The owner web app should configure `metadataBase` for production.

Why it matters:

Without `metadataBase`, generated Open Graph and Twitter metadata falls back to
`http://localhost:3000`, which is wrong for shared links in staging or
production.

Evidence:

- `make build` passes but warns that `metadataBase` is not set.

Recommended next action:

Add environment-based metadata base URL configuration for web and landing
surfaces.

Priority: `P3`

## Recommended Implementation Order

1. Fix CI blockers:
   - repair or remove stale display widget test;
   - resolve `next-env.d.ts` build drift;
   - make CI format checks non-mutating.
2. Implement production identity:
   - OIDC login/callback/session/logout in owner web;
   - verified API auth middleware;
   - remove trust in direct client-supplied actor headers.
3. Normalize authorization:
   - enforce explicit permissions on all tenant reads and writes;
   - add tests for allowed, forbidden, unauthenticated, and cross-tenant cases.
4. Add production operations:
   - deployment manifests;
   - migration release job;
   - secret manager wiring;
   - backup/restore runbooks.
5. Add observability:
   - metrics;
   - traces;
   - dashboards;
   - alerts;
   - incident runbooks.
6. Complete app workflows:
   - owner onboarding and member management;
   - waiter production auth/offline/realtime flows;
   - display kiosk hardening;
   - guest QR end-to-end tests.
7. Mature integrations and analytics:
   - first real vendor adapter;
   - analytics freshness, rebuild lifecycle, and projection lag alerts.

## Definition of Production-Ready V1

Seatd v2 should not be considered production-ready until these conditions are
true:

- `make lint`, `make test`, and `make build` pass from a clean checkout and
  leave the working tree clean.
- Owner users authenticate through real OIDC and cannot forge tenant or actor
  identity.
- Every tenant data endpoint enforces the documented permission for that
  surface.
- Device credentials are required for device-only surfaces and can be revoked.
- Guest QR rate limiting works across API instances.
- Deployment, rollback, migrations, backups, and restore drills are documented
  and tested.
- API, worker, realtime, database, display fleet, outbox, QR abuse, and
  integrations have metrics and alerts.
- Main owner, guest, waiter, display, realtime, analytics, and integration flows
  are covered by integration or browser-level tests.
- Platform admin actions are explicitly permissioned and audited.
