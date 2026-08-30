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
