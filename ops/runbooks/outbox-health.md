# Outbox Health Runbook

Use this when `SeatdOutboxBacklog`, `SeatdOutboxOldestPendingStale`,
`SeatdOutboxPermanentFailures`, or delayed downstream updates indicate outbox
processing is unhealthy.

## Dashboard

Use `ops/grafana/dashboards/seatd-data-plane-health.json`.

## Alerts

- `SeatdOutboxBacklog`: pending outbox records exceed 100 for 15 minutes.
- `SeatdOutboxOldestPendingStale`: oldest pending record is older than 5 minutes.
- `SeatdOutboxPermanentFailures`: at least one record exhausted retries in 10 minutes.

## First Checks

1. Check pending count and oldest pending age.
2. Compare claimed, processed, retried, and failed record rates.
3. Check `seatd_outbox_consumer_failures_total` by consumer if failures are rising.
4. Check worker `/healthz`, `/status`, and logs.
5. Check Postgres pool saturation and canceled acquires.

## Likely Causes

- Worker is stopped, under-provisioned, or repeatedly restarting.
- A consumer is failing and records are retrying until permanent failure.
- Postgres cannot serve claims or updates fast enough.
- A destination-specific payload or schema regression is blocking processing.

## Mitigation

1. Restart unhealthy worker instances.
2. If backlog is growing and DB is healthy, scale worker capacity within safe limits.
3. If a specific consumer is failing, disable or isolate that consumer if supported, then replay after correction.
4. If DB saturation is present, follow `postgres-health.md`.

## Escalation

Escalate when permanent failures occur or when oldest pending age continues to
grow after worker restart and DB health checks pass.

## Post-Incident

- Record affected destination and topic.
- Capture the consumer failure reason and retry count.
- Decide whether the consumer needs a dead-letter or replay procedure.
