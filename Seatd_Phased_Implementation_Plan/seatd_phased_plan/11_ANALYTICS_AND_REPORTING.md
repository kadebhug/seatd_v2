# 11 — Analytics and Reporting

## Goal

Turn operational events and TableSessions into trustworthy venue intelligence without introducing a warehouse prematurely.

## Dependencies

- `05_EVENTS_OUTBOX_AND_WORKERS.md`
- `02_CORE_DOMAIN_AND_DATABASE.md`
- service periods configured through owner platform.

## Phase 1 — Metric definitions before dashboards

Define exact formulas and exclusions for:
- occupancy duration;
- utilisation;
- table turnover count/rate;
- median/percentile session duration;
- assist response time;
- assist resolution time;
- request volume by action;
- zone/floor utilisation;
- service-period utilisation;
- device/realtime health where product-facing.

Document how reopened/corrected sessions and anomalous data are handled.

## Phase 2 — Service periods

Model service periods as first-class configuration linked to location/timezone.

Support:
- recurring day/time windows;
- named periods;
- overnight edge cases if required;
- timezone/DST correctness.

## Phase 3 — PostgreSQL projections

Create projection tables such as:
- hourly_table_metrics;
- daily_table_metrics;
- service_period_metrics;
- daily_zone_metrics;
- daily_floor_metrics;
- daily_location_metrics.

Project from operational events/session truth with Go workers.

## Phase 4 — Rebuildability

Every projection must be rebuildable from authoritative source records/events.

Provide:
- projector version;
- checkpoint/cursor;
- rebuild command;
- shadow/rebuild table strategy for large recalculations.

## Phase 5 — Owner dashboards

Deliver in value order:
1. today/current service summary;
2. utilisation over time;
3. turnover/session duration;
4. assist response/resolution;
5. zone/floor comparison;
6. service-period comparison;
7. multi-location comparison when groups exist.

## Phase 6 — Reports

Add:
- daily summary;
- weekly summary;
- exportable CSV/PDF only when useful;
- scheduled delivery only after notification/email infrastructure is intentionally added.

## Phase 7 — Data quality monitoring

Track:
- sessions open unusually long;
- impossible state transitions;
- event/projector lag;
- missing projection ranges;
- POS mismatch once integrations exist;
- duplicate/replayed event handling.

Treat analytics correctness as a production concern.

## Phase 8 — Advanced intelligence, later

Only after enough reliable history exists:
- benchmarks;
- recommendations;
- anomaly detection;
- forecasting.

Do not label weak heuristics as AI simply to add a feature category.

## ClickHouse gate

Stay on PostgreSQL until measured query volume/latency/storage patterns justify ClickHouse.

## Deliverables

- metric specification;
- service-period model;
- projection worker;
- projection tables;
- rebuild tooling;
- dashboards;
- data-quality monitors;
- report exports as justified.

## Definition of done

Any dashboard number can be traced to a documented formula and reconstructed from authoritative events/sessions.
