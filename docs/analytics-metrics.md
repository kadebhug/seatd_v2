# Analytics Metrics

Seatd analytics are PostgreSQL projections derived from operational tables. Every
metric is scoped by `organisation_id`, `location_id`, and a bounded time window.
All timestamps are stored in UTC and projected against the location timezone.

## Occupancy And Utilisation

Occupancy duration is the number of seconds where a table session overlaps the
window:

`max(0, min(session_end, window_end) - max(session_start, window_start))`

Open sessions use query time for current/today views and rebuild end time for
closed historical projections. Sessions with negative or impossible overlap are
excluded from totals and counted as anomalies.

Utilisation basis is:

`active_table_count * window_duration_seconds`

Utilisation rate is:

`occupancy_seconds / utilisation_basis_seconds`

When the active table count or window duration is zero, utilisation is reported
as `0`.

## Turnover And Session Duration

Turnover counts sessions completed inside the window.

Turnover rate is:

`completed_session_count / active_table_count`

Session duration percentiles are calculated from completed sessions where
`ended_at >= started_at`. The projected percentiles are continuous percentile
values for p50 and p90.

## Assist Metrics

Request volume is the number of assist requests with `requested_at` inside the
window.

Assist response seconds are measured from `requested_at` to `acknowledged_at`
for acknowledged or resolved requests with non-negative durations.

Assist resolution seconds are measured from `requested_at` to `resolved_at` for
resolved requests with non-negative durations.

Cancelled requests contribute to request volume but not response or resolution
duration averages unless they were acknowledged before cancellation.

## Grouped Metrics

Zone, floor, location, and service-period metrics use the same formulas as table
metrics. Zone and floor metrics are grouped by the table's current layout
assignment at projection time. Service-period metrics use the service period's
recurring local day and local start/end time definition.

## Service Periods

A service period belongs to a location and defines:

- a name;
- recurring local days of week, where Sunday is `0`;
- local `start_time` and `end_time`;
- an active/archive state.

If `end_time <= start_time`, the window is overnight and ends on the next local
calendar day. Projection boundaries are created with the IANA timezone on the
location record, so daylight-saving transitions are evaluated by the database
and Go time libraries instead of fixed UTC offsets.

## Data Quality

The analytics API reports anomalous data separately instead of silently mixing it
into business metrics. Current checks include:

- sessions with impossible negative durations;
- sessions open for more than 6 hours;
- active tables missing occupancy rows;
- trusted devices with stale or missing heartbeats;
- analytics projector lag from the latest checkpoint.
