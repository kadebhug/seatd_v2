# Event schema

Operational events use immutable, versioned JSON Schema documents. Additive
changes can extend a version; breaking changes require a new version directory.

- `schemas/v1/operational-event.schema.json` defines the canonical envelope.
- `fixtures/v1/*.json` are representative events validated in CI.
