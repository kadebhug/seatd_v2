# QR Assist Runbook

Use this when guests cannot submit QR assist requests, QR denials spike, or
operators report missing guest assistance requests.

## Dashboard

Use `ops/grafana/dashboards/seatd-operations-health.json` and, when API errors
are present, `ops/grafana/dashboards/seatd-api-health.json`.
QR denial and abuse decisions are visible on the operations dashboard. QR
request success and server-side failure are diagnosed through the API dashboard
route request, latency, and 5xx panels for guest routes.

## Alerts

- `SeatdGuestQRAbuseSpike`: guest QR denial, cooldown, or rate-limit decisions exceed 1 per second.

## First Checks

1. Confirm the table QR URL, venue, table, and guest-visible error.
2. Check API availability and latency for guest routes.
3. Check QR abuse and rate-limit decision rate by decision.
4. Confirm the QR capability is active and has not been rotated or revoked.
5. Check realtime delivery if the guest request is created but staff do not see it.

## Likely Causes

- Expired, rotated, revoked, or malformed QR capability.
- Rate limit, cooldown, or abuse control blocking requests.
- API route failure or DB contention.
- Realtime delivery failure after assist creation.

## Mitigation

1. If the QR token is invalid, issue a fresh QR capability and replace the printed or displayed code.
2. If denials are expected abuse, keep controls in place and notify the venue.
3. If denials block legitimate guests, review recent rate-limit decisions and adjust only with service-owner approval.
4. If the assist is created but not visible to staff, follow `realtime-delivery.md`.

## Escalation

Escalate immediately when valid guests at an active venue cannot request help
and the venue cannot switch to a manual workflow.

## Post-Incident

- Record the decision labels and affected QR capability state.
- Confirm whether the failure happened before creation, during API handling, or during staff delivery.
- Add a dashboard panel for guest request success once a success metric exists.
