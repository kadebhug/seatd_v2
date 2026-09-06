# Display Fleet Runbook

Use this when `SeatdDeviceFleetStale`, `SeatdDeviceFleetNeverHeartbeat`, or
operators report that venue display screens are stale or offline.

## Dashboard

Use `ops/grafana/dashboards/seatd-operations-health.json`.

## Alerts

- `SeatdDeviceFleetStale`: at least one trusted device missed the heartbeat freshness window.
- `SeatdDeviceFleetNeverHeartbeat`: at least one trusted device has never heartbeated.

## First Checks

1. Check stale and never-heartbeated trusted device counts.
2. Check heartbeat outcomes by result, device type, and platform.
3. Confirm whether the issue affects one display, one venue network, or all displays.
4. Check API availability and latency for heartbeat routes.
5. Check realtime delivery if the display is online but not updating.

## Likely Causes

- Display device lost network access or browser/app process is paused.
- Device credentials are missing, expired, revoked, or misconfigured.
- API heartbeat handling is failing.
- Realtime delivery is unhealthy after the display connects.

## Mitigation

1. Ask the venue to refresh or restart the affected display.
2. Confirm the display can reach Seatd from the venue network.
3. Re-pair or rotate device credentials if heartbeat failures indicate authorization issues.
4. Follow `api-availability-latency.md` or `realtime-delivery.md` if the display symptoms are downstream of those services.

## Escalation

Escalate when multiple trusted displays across different networks become stale,
or when re-pairing does not restore heartbeats.

## Post-Incident

- Record affected device type, platform, and heartbeat result.
- Confirm whether device credentials or network path caused the issue.
- Add clearer operator-facing device status if diagnosis required database inspection.
