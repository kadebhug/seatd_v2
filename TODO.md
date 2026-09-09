UI to build:

Owner web - QR capability management (apps/web/app/owner):
- Export/generate a table's QR token (backend exists: POST /v1/tables/{id}/qr-capabilities, no frontend calls it)
- Rotate / revoke controls
- Render an actual scannable QR image (no QR-image library anywhere in the repo yet - only token/URL is generated)
- Print-friendly view for posting codes at tables

Owner web - Analytics freshness/rebuild:
- Data-as-of / staleness indicator
- Rebuild job status/progress (rebuild endpoint returns "accepted" but no visible lifecycle)
- Alerting surface for projector lag

Display app - kiosk hardening:
- Stale snapshot age display
- Real Android app identifiers + release signing config (still default/TODO)
- Launch-on-boot / kiosk deployment instructions

Integrations - vendor mapping workflow:
- Table-mapping UI once a real vendor adapter (beyond reference_pos) is picked

Waiter app - verify in-flight production polish actually covers:
- Reconnect/resync UX
- Conflict-resolution UI
- Offline queue visibility
