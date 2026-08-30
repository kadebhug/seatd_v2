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
