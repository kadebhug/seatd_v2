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
