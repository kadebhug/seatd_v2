# Event schema

Operational events use immutable, versioned JSON Schema documents. Additive
changes can extend a version; breaking changes require a new version directory.

- `schemas/v1/operational-event.schema.json` defines the canonical envelope.
- `fixtures/v1/*.json` are representative events validated in CI.

OpenAPI owns HTTP endpoint contracts, including realtime handshake, sync
recovery, and realtime metrics. This package owns committed operational event
envelope compatibility and event type names consumed by realtime delivery.
