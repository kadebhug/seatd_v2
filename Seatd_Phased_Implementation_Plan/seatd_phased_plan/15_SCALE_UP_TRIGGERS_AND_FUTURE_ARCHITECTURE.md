# 15 — Scale-Up Triggers and Future Architecture

## Goal

Prevent premature infrastructure complexity by defining measurable conditions for introducing additional systems.

## Dependencies

All previous architecture can run without these components.

## Principle

> Add infrastructure because a current measurable problem requires it, not because future scale is imaginable.

## 1. NATS JetStream

### Do not add when
- one worker can dispatch outbox events directly;
- consumers are few and co-deployed;
- replay/fan-out requirements are simple.

### Add when
- multiple independently deployed consumers require durable fan-out;
- analytics, realtime, integrations, alerts, reporting need isolated consumption;
- consumer replay and independent checkpoints are operationally valuable;
- direct outbox routing has become a bottleneck/maintenance problem.

### Migration shape

```text
Postgres transaction
 → outbox
 → publisher
 → NATS JetStream
    ├── realtime consumer
    ├── analytics consumer
    ├── integration consumer
    └── alert/report consumers
```

The transactional outbox remains the bridge from database commit to message publication.

## 2. Valkey/Redis

### Add when
- shared rate limiting cannot remain local;
- multiple realtime nodes need presence/coordination;
- short-lived distributed locks/pairing state become necessary;
- repeated hot cache workloads materially burden PostgreSQL.

### Never use as
- authoritative table occupancy;
- authoritative assist state;
- only copy of operational data.

## 3. ClickHouse

### Add when
- PostgreSQL analytics has measured latency/resource impact despite projections/indexing;
- event history becomes very large;
- cross-location analytical scans materially affect operational DB performance;
- analytical concurrency/retention needs exceed the practical Postgres design.

Before adding ClickHouse, prove:
- slow-query evidence;
- workload separation need;
- expected retention/volume;
- operational capability to maintain another datastore.

## 4. Temporal

### Add when
- integrations contain long-running workflows with waits/retries/compensations;
- vendors can be unavailable for extended periods;
- workflow state is hard to reason about in ordinary jobs;
- operators need durable workflow visibility/retry controls.

Do not use Temporal for simple outbox jobs or short retries.

## 5. Kubernetes

### Add when
- Seatd operates across enough nodes/services that Compose/manual scheduling is a material operations burden;
- automatic rescheduling/autoscaling is needed;
- deployments are frequent across many independent services;
- the team has capacity to operate Kubernetes correctly.

Before Kubernetes, consider whether a simpler managed container platform solves the actual problem.

## 6. Multi-region

### Consider only when
- customer geography and latency require it;
- availability objectives justify complexity;
- data residency requires it;
- disaster recovery targets cannot be met more simply.

Operational consistency for live floor state should be prioritized over theoretical active-active architecture.

## 7. Cloud-provider-specific services

AWS/Azure/GCP-specific services are acceptable only when:
- there is a concrete feature/reliability/operational advantage;
- cost is understood;
- portability impact is documented in an ADR;
- an exit path is known where practical.

## 8. Scale metrics to collect from the start

Collect enough evidence to make these decisions later:
- API request rate/latency;
- active WebSocket connections;
- events/sec;
- outbox depth and age;
- worker throughput;
- DB CPU/IO/connection usage;
- top query latency;
- projection lag;
- analytics query latency;
- storage growth;
- integration webhook rate/failure rate;
- offline command backlog per location;
- number of organisations/locations/devices.

## 9. Architectural review cadence

Review scale decisions at meaningful milestones, for example:
- 10 production locations;
- 50 locations;
- 100+ locations;
- major integration launch;
- sustained reliability SLO breach;
- measured database/worker bottleneck.

Do not tie infrastructure changes to arbitrary calendar dates.

## Definition of done

The team can answer "why are we adding this technology now?" with production measurements, a documented problem, expected benefit, cost, and rollback/exit implications.
