# Platform Admin Management — Implementation Plan

Date: 2026-09-09

## Current State

`platform_admin` (see [`identity-permission-matrix.md`](./identity-permission-matrix.md)) can currently only **view** organisations:

- `GET /v1/platform/tenants`
- `GET /v1/platform/tenants/{id}`
- `GET /v1/platform/tenants/{id}/diagnostics`

Implemented in `internal/httpapi/platform.go`, gated by `inAuthorizedPlatformTx` checking the `platform.admin` permission, with reads already recorded to `audit_events` per [`platform-admin-audit-policy.md`](./platform-admin-audit-policy.md).

There is no path today for a platform admin to change an organisation's state, membership, or roles. This document scopes what's needed to close that gap.

**Status:** the permission split (Section 1) and the suspend/reactivate vertical slice (part of Sections 2/3/6/7) have shipped, gated on a new `platform.admin.write` permission. Edit organisation details, membership management, and archive/delete remain unimplemented — see Suggested Sequencing below.

## Priority Key

- `P0`: Blocks the feature from being safe to ship at all.
- `P1`: Required for a usable first version.
- `P2`: Needed for completeness / operational maturity.
- `P3`: Hardening or future improvement.

## 1. Permission Model (P0)

**Shipped** (migration `000014_platform_admin_write_permission`).

Reusing `platform.admin` for both read and write conflates "can see every tenant" with "can change every tenant." Split them:

- Add `platform.admin.write` permission.
- Grant it to `platform_admin` only — **not** to `support`.
- New migration (`0000NN_platform_admin_write_permission.up/down.sql`) following the pattern of `000003`/`000009`/`000010`.
- Update [`identity-permission-matrix.md`](./identity-permission-matrix.md) once merged.

## 2. Domain Mutations (P0/P1)

New methods in the identity/organisation domain service (wherever tenant lifecycle currently lives, alongside the existing read paths used by `platform.go`):

| Capability | Priority | Notes |
| --- | --- | --- |
| Suspend organisation | P1 | **Shipped.** Soft state change, not delete. Blocking operational writes for a suspended tenant on other request paths is **deferred** — there are 6+ independent tenant-tx helpers across domain services plus a separate device-auth entry point, and correctly gating all of them is its own cross-cutting hardening task. `organisations.status` is visible everywhere it's already read (owner UI, platform detail/search DTOs), so there is a visibility signal even without a hard block yet. Follow-up: a shared `RequireOrganisationActive`-style check, or an `enforceActive bool` on the tenant-tx helpers, prioritizing guest QR ordering and device heartbeat/pairing first. |
| Reactivate organisation | P1 | **Shipped.** Reverses suspend. |
| Edit organisation details (name, contact info) | P1 | Low risk, high value for support workflows. |
| Manage membership (add/remove user, change role) | P1 | Reuses existing `organisation_owner`/`location_manager`/etc. role assignment logic if it exists; otherwise build the minimal version here. |
| Archive/hard-delete organisation | P2 | Deliberately last — see Safety Rails below. Likely never fully "hard" delete; archive + data retention policy instead. |

Favor **suspend over delete** as the primary destructive action for v1. A tenant should never be irrecoverably destroyed by a slipped click.

## 3. API Surface (P1)

Extend `internal/httpapi/platform.go` and register in `internal/httpapi/api.go`:

```
POST   /v1/platform/tenants/{id}/suspend
POST   /v1/platform/tenants/{id}/reactivate
PATCH  /v1/platform/tenants/{id}
POST   /v1/platform/tenants/{id}/members
PATCH  /v1/platform/tenants/{id}/members/{userId}
DELETE /v1/platform/tenants/{id}/members/{userId}
```

Each mutating handler:
- Requires `platform.admin.write`, not just `platform.admin`.
- Requires a `reason` field in the request body — free-text justification, stored in the audit metadata. Matches the audit policy's expectation of non-sensitive JSON metadata per action.
- Returns the updated tenant representation so the frontend can update optimistically-safe state.

## 4. Audit Trail (P0 — already partially built)

Extend `recordPlatformAudit` with new action types:

- `platform.tenant_suspend`
- `platform.tenant_reactivate`
- `platform.tenant_update`
- `platform.tenant_member_add`
- `platform.tenant_member_role_change`
- `platform.tenant_member_remove`

Per [`platform-admin-audit-policy.md`](./platform-admin-audit-policy.md), every one of these must record `actor_ref`, `action`, `target_type`, `target_id`, organisation context, and the `reason` from the request body. Disable-type controls must remain reversible only through this same audited path (no direct DB edits).

## 5. Safety Rails (P0)

- **Self-lockout guard**: reject a role/membership change that would remove the last `platform_admin` from the platform, or the last owner-role member from an organisation with active data.
- **Confirmation semantics**: destructive actions (suspend, archive, member removal) require the `reason` field to be non-empty; frontend requires a typed confirmation (e.g. type the org name) before submitting.
- **RLS interaction**: mutations run through the same `SetPlatformAdminContext` session-flag bypass as reads (`seatd_is_platform_admin()`). Decided for suspend/reactivate: no booking-state check on suspend — suspend is fully reversible (reactivate undoes it, nothing is destroyed), so an in-flight booking on a suspended tenant isn't itself a destructive-action concern. What remains open is whether *other* request paths should reject writes while a tenant is suspended — deferred, see the follow-up note under Section 2.
- No support for impersonation in this plan — matches existing audit policy stating impersonation "is not enabled."

## 6. Contract + Generated Client (P1)

- Add new paths/schemas/request-response bodies to `packages/api-contract/openapi/seatd.v1.json`.
- Regenerate `packages/typescript-seatd-client` via its `scripts/generate.mjs`.
- Verify generated types compile against `apps/web`.

## 7. Frontend (P1/P2)

Extend `apps/web/app/platform/` (currently view-only "Tenant Operations" console):

- P1: Suspend/Reactivate buttons with a confirmation dialog requiring a reason.
- P1: Edit organisation details form.
- P2: Membership management table (add/remove/change role) on the tenant detail page.
- P2: Surface audit history for a tenant directly on its detail page (read via existing `audit.read` permission).

## 8. Testing (P0/P1)

- Unit tests for new domain mutation methods, including the self-lockout guard.
- HTTP handler tests asserting `platform.admin.write` is required (403 for `support` role, 403 for `organisation_owner`, 200 for `platform_admin`).
- Audit assertion tests: each mutation produces exactly one `audit_events` row with the expected action/target/reason.
- Playwright coverage for the new frontend flows (suspend, reactivate, edit, member management), including the confirmation-dialog path.

## Suggested Sequencing

1. Permission split + migration (Section 1).
2. Suspend/Reactivate end-to-end (domain → API → contract → frontend) as the first vertical slice — smallest surface, proves out the audit + safety-rail pattern.
3. Edit organisation details.
4. Membership management.
5. Archive/delete policy, if still wanted after suspend has been in use.

## Open Questions

- Does "archive/delete" need to exist at all for v1, or is suspend sufficient indefinitely?
- Should membership role changes for `organisation_owner` require a secondary approval (e.g. two platform admins), given it can transfer control of a tenant?
- Retention policy for audit metadata containing free-text `reason` fields — any PII handling concerns?
