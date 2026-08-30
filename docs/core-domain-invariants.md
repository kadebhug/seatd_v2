# Core Domain Invariants

The database enforces these invariants directly:

- Tenant chains are validated with composite foreign keys.
- Location slugs are unique inside an organisation.
- Floor slugs are unique inside a location.
- Zone names are unique inside a floor.
- Table labels are unique inside a floor.
- Live table queries exclude inactive or soft-deleted tables.
- Table shapes are limited to `rectangle`, `circle`, `square`, and `custom`.
- Table geometry and floor canvas metadata must be JSON objects.
- Current occupancy has one row per table.
- Occupancy status is limited to `available` and `occupied`.
- Occupied rows must point to a current table session; available rows must not.
- Only one active table session can exist for a table.
- QR capability tokens are globally unique.
- Assist status is limited to `pending`, `acknowledged`, `resolved`, and
  `cancelled`.
- Assist timestamps must match lifecycle state.
- Versions start at one and must remain positive.

The application service enforces these command-level rules:

- `expectedVersion` must match before changing occupancy or assist state.
- Occupying an occupied table fails.
- Clearing an available table fails.
- Assist acknowledgement only applies to pending assists.
- Assist resolve and cancel only apply to pending or acknowledged assists.
- Assist changes do not update table occupancy.
