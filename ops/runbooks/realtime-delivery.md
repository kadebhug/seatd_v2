# Realtime Delivery Runbook

Use this when `SeatdRealtimeDrops`, `SeatdRealtimeBackpressureCloses`, or live
floor reports show that connected clients are not receiving updates.

## Dashboard

Use `ops/grafana/dashboards/seatd-realtime-health.json`.

## Alerts

- `SeatdRealtimeDrops`: realtime messages were dropped in the last 5 minutes.
- `SeatdRealtimeBackpressureCloses`: clients were closed because send buffers filled.

## First Checks

1. Confirm whether clients can open the floor view and establish WebSocket connections.
2. Check active connections, connection churn, dropped messages, and backpressure closes.
3. Compare published message rate with active connections.
4. Check API health for command submission failures.
5. Check outbox health if updates are created but not delivered through downstream consumers.

## Likely Causes

- Slow or disconnected clients causing backpressure.
- Realtime gateway overload or an instance-specific failure.
- Client network instability at a venue.
- Events are not being produced because API writes or outbox processing are failing.

## Mitigation

1. If one realtime instance is unhealthy, drain or restart that instance.
2. If backpressure closes spike for one venue, ask operators to refresh affected devices and check local network quality.
3. If published message rate drops to zero while API commands continue, follow `outbox-health.md`.
4. If API writes are also failing, follow `api-availability-latency.md`.

## Escalation

Escalate immediately when multiple venues lose live floor updates, or when
message drops continue after instance restart and client refresh.

## Post-Incident

- Record whether the issue was gateway, client network, event production, or outbox delivery.
- Capture active connection counts and drop/backpressure rates.
- Add venue-level labels only if they can be kept low-cardinality and safe.
