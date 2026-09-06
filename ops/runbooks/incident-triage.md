# Incident Triage Runbook

Use this runbook when Seatd is failing in production and the first report does
not clearly identify the subsystem.

## Dashboards

- Seatd API Availability and Latency: `ops/grafana/dashboards/seatd-api-health.json`
- Seatd Realtime Delivery Health: `ops/grafana/dashboards/seatd-realtime-health.json`
- Seatd Data Plane Health: `ops/grafana/dashboards/seatd-data-plane-health.json`
- Seatd Operations Health: `ops/grafana/dashboards/seatd-operations-health.json`

## First Five Minutes

1. Confirm the affected environment, venue, location, and user-visible symptom.
2. Check whether the API service is reachable: `/healthz`, `/status`, and the API dashboard.
3. Check the data plane dashboard for Postgres saturation and outbox backlog.
4. Check the realtime dashboard for message drops, backpressure closes, or connection churn.
5. Check the operations dashboard for display heartbeat, QR denial, webhook, or reconciliation alerts.

## Symptom Router

| Symptom | Start with | Then check |
| --- | --- | --- |
| Owner or waiter app cannot load data | API availability and latency | Postgres health |
| Live floor view loads but stops updating | Realtime delivery | Outbox health |
| Guest QR request fails or is denied | QR assist | API availability and latency |
| Display screen is stale or offline | Display fleet | Realtime delivery |
| POS changes do not appear in Seatd | Integrations | Outbox health |
| Alerts mention connection pool or acquire cancellation | Postgres health | API availability and latency |

## Escalation

Escalate to the service owner when any page-severity alert persists beyond its
`for` duration after the first mitigation attempt, or immediately when a
production venue cannot operate around the issue.

## Post-Incident Follow-Up

- Link the incident to the firing alert and dashboard panel.
- Record the root cause category: API, DB, realtime, outbox, display, QR, network, or integration.
- Add missing dashboard panels or runbook steps discovered during the incident.
- Confirm whether alert thresholds produced useful signal or need tuning.
