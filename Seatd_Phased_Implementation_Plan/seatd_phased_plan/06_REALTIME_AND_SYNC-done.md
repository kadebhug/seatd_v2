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
