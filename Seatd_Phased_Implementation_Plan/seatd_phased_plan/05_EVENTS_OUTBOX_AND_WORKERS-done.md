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
