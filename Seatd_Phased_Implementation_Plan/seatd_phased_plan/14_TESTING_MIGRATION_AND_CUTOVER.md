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
