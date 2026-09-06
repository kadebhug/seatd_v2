# Postgres Health Runbook

Use this when `SeatdDatabasePoolSaturated`, `SeatdDatabaseAcquireCanceled`, or
API/worker latency points to database contention.

## Dashboard

Use `ops/grafana/dashboards/seatd-data-plane-health.json`.

## Alerts

- `SeatdDatabasePoolSaturated`: more than 85% of configured pool connections are acquired.
- `SeatdDatabaseAcquireCanceled`: requests are timing out or canceled while waiting for a connection.

## First Checks

1. Check pool saturation by service and instance.
2. Check acquired, idle, total, and max connection counts.
3. Check canceled acquire count over the last 5 minutes.
4. Check API p95/p99 latency and 5xx ratio.
5. Check outbox backlog, because worker retries can amplify DB pressure.

## Likely Causes

- Query latency or lock contention.
- Connection leaks or long-lived transactions.
- Insufficient pool sizing for the current traffic pattern.
- Worker backlog producing sustained claim/update pressure.

## Mitigation

1. Reduce incoming load or roll back the change that increased query pressure.
2. Restart instances only if there is evidence of leaked or wedged connections.
3. Pause or reduce worker concurrency if outbox processing is saturating the pool.
4. Scale database or service pools only after confirming Postgres can accept the additional connections.

## Escalation

Escalate when canceled acquires continue after load reduction, or when all
services show sustained pool saturation.

## Post-Incident

- Record the saturated service, pool size, acquired connection count, and canceled acquires.
- Identify the route, worker path, or reconciliation job that drove pressure.
- Add query-level tracing or DB instrumentation if the root query was not visible.
