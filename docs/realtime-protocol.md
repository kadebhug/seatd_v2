# Realtime Protocol

Seatd realtime is a delivery layer for committed operational events. PostgreSQL and the HTTP APIs remain the source of truth.

## Authentication

Clients connect with an HTTP WebSocket upgrade:

`GET /v1/realtime`

Required headers:

- `X-Seatd-Organisation-ID`
- `X-Seatd-Location-ID`
- `X-Seatd-Actor-Ref`
- `X-Seatd-Device-ID`

Access tokens are not accepted in URLs. During the current development identity phase, the gateway authorizes a connection by checking that the device is trusted for the organisation/location and that the actor has `operations.read` through a location or organisation membership.

## Message

Messages describe concrete events, not generic refresh hints:

```json
{
  "type": "table.occupied",
  "eventId": "11111111-1111-1111-1111-111111111111",
  "organisationId": "22222222-2222-2222-2222-222222222222",
  "locationId": "33333333-3333-3333-3333-333333333333",
  "entityType": "table",
  "entityId": "44444444-4444-4444-4444-444444444444",
  "version": 27,
  "occurredAt": "2026-08-29T10:30:00Z",
  "data": {
    "tableId": "44444444-4444-4444-4444-444444444444",
    "status": "occupied"
  }
}
```

Delivery is scoped to a location. Clients must ignore duplicate events by `eventId` and ignore stale entity updates by comparing `version`.

## Recovery

On reconnect:

1. Authenticate using the same headers.
2. Call `GET /v1/sync/location-snapshot` to load authoritative tables, active assists, and a cursor.
3. Subscribe to `GET /v1/realtime`.
4. Process only events newer than the snapshot cursor.

If a client has an older cursor and does not need a full snapshot, it can call:

`GET /v1/sync/events?after=<cursor>&limit=200`

The cursor is opaque to clients. Responses include the next cursor.

## Backpressure

Each connection has a bounded outbound queue. If the queue fills, the gateway closes that connection and increments the backpressure metric. The client must reconnect and reconcile through the snapshot or events API.

## Heartbeat

The gateway sends WebSocket ping frames on a fixed interval. Clients should respond with pong frames and reconnect if the connection becomes unhealthy.

## Conflict Handling

Offline commands must submit `expectedVersion`. The API returns `version_conflict` with the current version details when another device has already changed the entity. Clients then reconcile through snapshot/events and re-evaluate pending commands.

