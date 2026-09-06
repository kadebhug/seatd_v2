# API Availability and Latency Runbook

Use this when `SeatdHighHTTPErrorRate`, `SeatdSlowHTTPP95`, or user reports
show failed or slow API requests.

## Dashboard

Use `ops/grafana/dashboards/seatd-api-health.json`.

## Alerts

- `SeatdHighHTTPErrorRate`: more than 5% HTTP 5xx responses for 10 minutes.
- `SeatdSlowHTTPP95`: route p95 latency above 1 second for 15 minutes.

## First Checks

1. Check `/healthz`, `/status`, and `/version` for the affected API instance.
2. In the API dashboard, identify whether the issue is all routes or one route.
3. Check the 5xx ratio by route and p95/p99 latency by route.
4. Check the data plane dashboard for Postgres pool saturation or canceled acquires.
5. Check recent deploys, configuration changes, and dependency incidents.

## Likely Causes

- Postgres connection pool saturation or blocked queries.
- An API handler returning 5xx after a downstream store or integration failure.
- A route-specific regression after deployment.
- Bad configuration, missing secret, or unavailable external provider.

## Mitigation

1. If DB saturation is present, follow `postgres-health.md`.
2. If only one route is failing, route traffic away from the affected release or roll back the release.
3. If all API routes fail and `/healthz` fails, restart the unhealthy API instances.
4. If guest QR or integration routes are the affected paths, follow the subsystem runbook after restoring base API health.

## Escalation

Escalate when 5xx ratio remains above 5% for another 10 minutes after rollback
or restart, or when p95 stays above 1 second and operators cannot complete
critical floor workflows.

## Post-Incident

- Capture the failing route, status codes, and latency percentile.
- Add route-specific dashboard panels if the route was hard to isolate.
- Add or tune alerts only after confirming the incident signal was actionable.
