# 10 — Display/Kiosk and Device Management

## Goal

Build a controlled Android display experience and make every operational device visible, manageable, and revocable.

## Dependencies

- `03_IDENTITY_TENANCY_AND_SECURITY.md`
- `06_REALTIME_AND_SYNC.md`

## Phase 1 — Device domain/API

Implement first-class Device records and APIs for:
- registration;
- pairing;
- assignment;
- heartbeat/last seen;
- app version;
- revocation;
- device type;
- capabilities/configuration.

## Phase 2 — Display pairing

Replace incomplete/manual pairing flow with a complete UX:

```text
Owner generates short-lived pairing code
  → display enters/scans code
  → backend validates location
  → backend issues revocable device credential
  → display persists credential securely
```

Pairing codes must expire and be single-use unless a deliberate alternative is documented.

## Phase 3 — Flutter Android display baseline

Standardize on one known Android device profile initially.

Implement:
- launch on boot;
- kiosk/lock task where deployment mode permits;
- orientation control;
- screen wake;
- secure device credential storage;
- automatic reconnect;
- app version reporting.

## Phase 4 — Local cache and offline display

Cache:
- restaurant/location info;
- floors/zones/tables;
- last-known occupancy/assist state;
- rotation/display configuration.

On disconnect:
- show a subtle offline/stale indicator;
- keep last-known layout;
- never pretend stale data is current;
- continue retrying safely.

## Phase 5 — Live display views

Implement:
- availability summary;
- floor plan;
- configurable rotation;
- table capacity labels;
- occupancy visual state;
- optional assist indicator only if appropriate for public display policy.

## Phase 6 — Fleet management

Owner/platform web should show:
- online/offline based on heartbeat threshold;
- last seen;
- version;
- location;
- credential state;
- revoke button;
- pairing/re-pairing workflow.

## Phase 7 — Update strategy

Initially support controlled application release process rather than building a custom updater.

Document:
- staged APK/managed distribution method;
- minimum supported app version;
- forced upgrade behaviour only if necessary;
- rollback.

## Deliverables

- device APIs;
- secure pairing;
- Flutter Android display app;
- offline cache;
- heartbeat/device health;
- owner/platform device management UI;
- deployment runbook.

## Definition of done

A new display can be paired by a venue owner without backend intervention, continue showing clearly marked last-known data offline, and be remotely revoked.

## Implementation status

Implemented in this repository. See `docs/display-device-runbook.md` for pairing, health, offline, and Android deployment notes.
