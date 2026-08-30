# Integration Authority Rules

Seatd integrations translate vendor records into canonical Seatd operations. Core domain code must not branch on vendor names.

## Table Occupancy

- POS-originated table occupied and available states may open or clear Seatd table sessions only through the operations command service.
- Seatd remains the source of record for command idempotency, session versions, operational events, and outbox fanout.
- Manual Seatd actions are allowed while an integration is connected. Later reconciliation records conflicts instead of blindly overwriting ambiguous state.

## Assists

- Seatd is authoritative for assist requests and assist lifecycle state.
- If a POS state conflicts with a table that has an unresolved assist, reconciliation records an `ambiguous_conflict` discrepancy for operator review.

## Auto-Correction

- Safe auto-correction is limited to mapped table occupancy mismatches with no active pending or acknowledged assist.
- Unmapped tables, invalid webhooks, vendor outages, and ambiguous conflicts stay visible in integration health surfaces.

## Reference POS

- `reference_pos` uses HMAC-SHA256 signatures from `SEATD_REFERENCE_POS_WEBHOOK_SECRET`.
- Local development defaults to `reference-pos-local-secret` when the env var is not set.
- Reconciliation fetches deterministic external table states from the connection `config.referenceTables` payload.
