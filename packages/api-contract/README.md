# API contract

`openapi/seatd.v1.json` is the source of truth for Seatd HTTP APIs.

The contract covers REST endpoints, realtime sync recovery endpoints, the
`/v1/realtime` WebSocket handshake, and the JSON metrics surface at
`/v1/realtime/metrics`.

Operational event compatibility is owned by `packages/event-schema`. Keep
realtime delivery message fields in this package aligned with that event type
enum, but define event envelope evolution in the event schema package.
