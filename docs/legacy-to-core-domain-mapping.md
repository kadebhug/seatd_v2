# Legacy To Core Domain Mapping

The target model preserves stable legacy identifiers until cutover is complete.

## Restaurant

- Legacy `restaurant` becomes one `organisation`.
- Each legacy restaurant gets one initial `location`.
- `legacy_restaurant_mappings` records the legacy restaurant id, target
  organisation id, and target location id.
- Explicit `legacy_restaurant_id` columns on `organisations` and `locations`
  support direct audits while migration tooling is still being built.

## Layout

- Legacy floors map to `floors`.
- If legacy data has no zone concept, create one default zone per floor during
  migration.
- Legacy tables map to `tables`.
- `legacy_table_mappings` records stable legacy table ids for cutover
  comparison and rollback planning.
- Table visual placement is stored in `tables.geometry`.
- QR capability tokens are stored in `table_qr_capabilities`, separate from
  visual placement.

## Operational State

- Legacy `available` maps to `table_occupancy.status = 'available'`.
- Legacy `occupied` maps to `table_occupancy.status = 'occupied'` and opens a
  current `table_sessions` row from the safest available timestamp.
- Legacy `attention` must not map to an occupancy state. Migrate the assist
  independently and choose occupancy only from reliable source data.
- Do not fabricate historical sessions unless legacy timestamps support the
  reconstruction. Label reconstructed session quality in migration output.
