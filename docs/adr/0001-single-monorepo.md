# ADR-0001: Use a single Seatd monorepo

- Status: Accepted
- Date: 2026-08-29
- Owners: `@seatd/engineering`

## Context

Seatd will contain several Go, web, and Flutter applications plus shared API and
event contracts. Coordinating compatible contract and client changes across
nested or separately assumed checkouts would make atomic changes and consistent
local tooling harder.

## Decision

Use one Git repository rooted at `seatd_v2`. Store runnable products under
`apps/`, shared artifacts under `packages/`, and platform material under
`infra/`, `ops/`, and `docs/`. Do not introduce nested Git repositories or
submodule-based assumptions for new code. Root Make targets are the stable local
workflow interface.

## Consequences

Cross-component changes can be reviewed and merged atomically. CI may later use
path filtering, but repository-wide commands remain available. Teams share root
tooling conventions and ownership boundaries must remain explicit.

## Alternatives considered

Separate repositories and nested checkouts were rejected because they introduce
version coordination and bootstrap complexity before the application boundaries
are stable.
