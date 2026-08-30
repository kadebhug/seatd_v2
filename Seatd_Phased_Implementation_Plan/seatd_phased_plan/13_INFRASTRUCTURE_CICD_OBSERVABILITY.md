# 13 — Infrastructure, CI/CD, Observability, and Operations

## Goal

Run Seatd reliably on portable infrastructure without making AWS or Kubernetes a prerequisite.

## Dependencies

Can start during foundation work and mature alongside every phase.

## Phase 1 — Local/development infrastructure

Use Docker Compose for:
- PostgreSQL;
- API;
- worker;
- realtime;
- web/guest where useful;
- object storage when introduced;
- observability dependencies as practical.

Create repeatable local setup and fixture loading.

## Phase 2 — Initial staging

Suggested topology:

```text
Caddy
  ├── API
  ├── Realtime
  └── Web apps
       ↓
PostgreSQL
Worker
S3-compatible object storage
```

Use one application VPS if appropriate, with a managed or separately backed-up PostgreSQL option.

## Phase 3 — Object storage

Move floor backgrounds, logos, QR exports, reports, and other files to S3-compatible storage.

Options:
- MinIO when self-hosting;
- any compatible managed provider.

Do not keep production assets on an ephemeral application filesystem.

## Phase 4 — CI/CD

GitHub Actions pipeline:

```text
push
 → lint
 → unit tests
 → integration tests
 → build images/apps
 → vulnerability scan
 → push image
 → deploy staging
 → migrations
 → smoke tests
 → controlled production promotion
```

Use immutable image tags tied to commit/version.

## Phase 5 — Reverse proxy/TLS

Use Caddy initially for:
- automatic TLS;
- routing;
- WebSocket proxying;
- compression/static policies where appropriate;
- security headers.

Redact sensitive headers/query parameters from logs.

## Phase 6 — Backups and recovery

Minimum production posture:
- automated PostgreSQL backups;
- off-site copy;
- encrypted backup storage;
- point-in-time recovery where provider/setup supports it;
- object storage backup/versioning as needed;
- restore drills.

Measure recovery objectives:
- RPO;
- RTO.

A backup process is not complete until restore has been tested.

## Phase 7 — OpenTelemetry baseline

Instrument:
- HTTP requests;
- database operations;
- command handling;
- worker jobs;
- outbox lag;
- realtime publish/delivery;
- integration processing;
- key mobile/web errors.

Propagate:
- trace_id;
- request_id;
- command_id;
- event_id;
- organisation/location ids;
- device id where relevant.

## Phase 8 — Grafana stack

Portable target:
- OpenTelemetry Collector;
- Prometheus-compatible metrics;
- Grafana;
- Loki;
- Tempo;
- Sentry-compatible crash/error reporting for web/mobile if desired.

## Phase 9 — Product-specific operational truth monitoring

Monitor more than servers:
- stale floor state;
- offline command backlog;
- oldest pending outbox record;
- realtime delivery latency;
- unusually long sessions;
- impossible transitions;
- inactive/revoked device attempts;
- assist delivery latency;
- integration mismatches.

**Floor truth quality is a production metric.**

## Phase 10 — Production pilot topology

Move to:
- application server(s);
- dedicated/managed PostgreSQL;
- off-site backups;
- central monitoring;
- object storage;
- separate worker deployment if load warrants.

No AWS-specific service is required.

## Deliverables

- Compose environments;
- staging/production deployment manifests;
- GitHub Actions workflows;
- Caddy config;
- backup/restore runbook;
- OTel instrumentation;
- dashboards/alerts;
- secrets handling guide;
- disaster recovery test record.

## Definition of done

A failed deploy can be rolled back, a database can be restored from backup, and on-call/support can distinguish infrastructure health from incorrect digital-floor truth.
