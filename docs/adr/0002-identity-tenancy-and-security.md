# ADR 0002: Identity, Tenancy, and Security Model

## Status

Accepted

## Context

Seatd needs standards-based authentication without making provider-specific identity concepts part of the domain model. Seatd also needs its own authorization source of truth, strong tenant isolation, trusted device support, and guest QR capabilities that do not collect guest PII by default.

## Decision

Seatd will use OIDC/OAuth2 for authentication and store external identity links separately from Seatd user profiles. Provider `issuer` and `subject` identify an external identity; Seatd roles and permissions determine what the user can do.

Tenant-owned database tables use application query scoping and PostgreSQL RLS. Application code sets `seatd.current_organisation_id` and, where needed, `seatd.current_location_id` locally inside transactions. Platform-admin paths set `seatd.platform_admin` locally and must write audit events for administrative actions.

Device trust is modeled with first-class `devices` and `device_credentials`. Device and QR secrets are high-entropy opaque values; database lookups use a short prefix plus a SHA-256 hash so raw credentials do not need to be queried.

## Consequences

Every operational write must execute with an explicit tenant context. Deliberately under-scoped application queries remain constrained by RLS for protected tables.

Authentication tokens are not the permanent authorization source of truth. They can carry identity and session claims, but role and permission checks resolve against Seatd-owned tables.
