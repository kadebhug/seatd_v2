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
