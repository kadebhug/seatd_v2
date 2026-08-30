# 01 — Foundation and Engineering Baseline

## Goal

Create the repository, contract, development, and delivery foundations required before domain migration starts.

## Dependencies

None.

## Phase 1 — Establish the target monorepo

Create the recommended top-level structure:

```text
seatd/
  apps/
    api/
    worker/
    realtime/
    web/
    guest/
    waiter/
    display/
  packages/
    api-contract/
    event-schema/
    dart-seatd-client/
    typescript-seatd-client/
  infra/
  ops/
  docs/
```

Actions:
- choose one root Git repository;
- remove nested checkout assumptions for new code;
- document code ownership by area;
- standardize local commands;
- define environment naming: `local`, `test`, `staging`, `production`;
- define configuration via environment variables and secret injection;
- establish ADR directory for architectural decisions.

### Exit criteria

A clean checkout can bootstrap all required local dependencies with one documented command.

## Phase 2 — Create application skeletons

Create minimal runnable applications:
- Go API;
- Go worker;
- Go realtime gateway;
- Next.js web application;
- lightweight guest web application;
- Flutter waiter app;
- Flutter display app.

Do not implement business features yet.

Each app must expose:
- version/build metadata;
- health/status where applicable;
- structured logging;
- configuration validation;
- graceful shutdown.

### Exit criteria

CI can build every application and run basic smoke tests.

## Phase 3 — Contract-first tooling

Establish:
- OpenAPI as HTTP contract source;
- JSON Schema for operational events;
- generated Dart API client;
- generated TypeScript API client;
- versioned schemas under `packages/`.

Rules:
- no manually duplicated request/response models across clients;
- breaking API changes require explicit contract versioning or migration;
- event schemas are immutable once published; evolve with new versions.

### Exit criteria

A trivial API endpoint can be added to OpenAPI and consumed through generated Dart and TypeScript clients.

## Phase 4 — Database migration discipline

Adopt a proper migration tool and process.

Requirements:
- ordered SQL migrations;
- no production schema mutation during app startup;
- forward migrations reviewed in CI;
- migration checks on a clean database;
- migration checks from the latest supported production baseline;
- seed fixtures separated from migrations.

### Exit criteria

The complete schema can be reproduced from migrations only.

## Phase 5 — Engineering quality gates

Add:
- Go formatting/linting/tests;
- TypeScript lint/typecheck/tests;
- Flutter analyze/tests;
- SQL/static migration checks;
- dependency vulnerability scanning;
- container build checks;
- secret scanning.

Recommended CI order:

```text
lint → unit tests → integration tests → build → scan → package
```

### Exit criteria

No branch can merge when required checks fail.

## Deliverables

- target repository structure;
- local development Compose stack;
- app skeletons;
- OpenAPI/JSON Schema pipeline;
- generated client workflow;
- migration tooling;
- CI baseline;
- ADR template;
- developer setup guide.

## Do not build yet

- NATS;
- Redis/Valkey;
- ClickHouse;
- Temporal;
- Kubernetes;
- POS adapters.
