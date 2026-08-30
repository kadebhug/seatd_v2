# Developer guide

## Repository model

This directory is the one Git repository and the root of every local command.
New code must not rely on nested Git checkouts, Git submodules, or commands run
from a parent repository. Shared code belongs under `packages/`; applications
must consume it through declared workspace/package dependencies.

The top-level areas are:

- `apps/`: independently runnable Seatd applications;
- `packages/`: versioned contracts, schemas, and generated clients;
- `infra/`: local and deployed infrastructure definitions;
- `ops/`: operational runbooks and delivery automation;
- `docs/`: architecture decisions and engineering documentation.

Code review ownership is declared in the root `CODEOWNERS` file. The placeholder
team handles must be mapped to real teams before ownership checks are enforced.

## Bootstrap

From a clean checkout, run:

```sh
make bootstrap
```

The command validates prerequisites and layout, creates an ignored `.env.local`
from `.env.example`, and invokes component bootstrap hooks when they exist.
It is idempotent and safe to run again.

## Standard commands

Run all commands from the repository root:

```sh
make bootstrap  # prepare local dependencies and configuration
make check      # validate repository policies
make generate   # regenerate contract-derived clients
make fmt        # format every component
make lint       # lint every component
make test       # test every component
make build      # build every component
make clean      # remove generated build output
```

Application phases may extend the command runner, but these root command names
are the stable developer interface.

## Local services

Start the local database with:

```sh
docker compose -f infra/compose.yml up -d postgres
```

Run migrations against the local database with:

```sh
./infra/database/scripts/check-migrations
```

Seed fixtures are separate from migrations:

```sh
./infra/database/scripts/load-seeds
```

Applications must not mutate production schema during startup. Schema changes
are reviewed as ordered SQL migrations and applied as deployment steps.

## Contract generation

The HTTP source of truth is `packages/api-contract/openapi/seatd.v1.json`.
Operational event schemas are versioned under `packages/event-schema/schemas/`.

Regenerate clients after contract changes:

```sh
make generate
```

Generated clients must be committed with the contract change. CI checks for
generation drift with `git diff --exit-code`.

## Environments and configuration

`SEATD_ENV` must be exactly one of:

- `local`: developer workstation and local containers;
- `test`: automated or local tests with isolated disposable resources;
- `staging`: production-like shared validation;
- `production`: live customer workloads.

Configuration follows the twelve-factor model: applications read environment
variables at startup, validate required values, and fail clearly when invalid.
Non-secret local defaults are documented in `.env.example`.

Secrets must never be committed. Inject them at runtime through one of:

- the developer's ignored `.env.local` file or exported shell variables;
- CI's protected secret store;
- the staging or production platform's secret manager.

Configuration files must not contain environment-specific credentials. Secret
values must not appear in logs, command output, build arguments, or generated
artifacts. Application-specific variables should use the `SEATD_` prefix and be
added to `.env.example` with a non-secret example or an explanatory blank value.

## Architectural decisions

Architecturally significant choices are recorded in `docs/adr/`. Copy
`docs/adr/0000-template.md`, assign the next four-digit number, and keep accepted
decisions immutable; supersede them with a new ADR when the decision changes.
