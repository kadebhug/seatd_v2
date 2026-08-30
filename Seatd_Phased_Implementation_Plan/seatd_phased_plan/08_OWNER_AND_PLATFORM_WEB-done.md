# 08 — Owner and Platform Web

## Goal

Build a responsive Next.js management surface for venue configuration, operational oversight, analytics, and Seatd platform administration.

## Dependencies

- `04_BACKEND_API_AND_COMMAND_MODEL.md`
- `03_IDENTITY_TENANCY_AND_SECURITY.md`
- basic device APIs from `10_DISPLAY_KIOSK_AND_DEVICE_MANAGEMENT.md` can be developed in parallel.

## Phase 1 — Web shell and access control

Create one Next.js codebase with clearly separated route areas:
- owner/manager;
- platform admin;
- shared design system/components.

Implement:
- OIDC session handling;
- role-aware navigation;
- organisation/location switcher;
- protected server/client routes;
- generated TypeScript API client.

## Phase 2 — Organisation and location configuration

Owner workflows:
- organisation profile;
- location profile;
- location operating configuration;
- enable/disable relevant features;
- location switcher for groups.

Platform workflows:
- create/disable organisations;
- create locations;
- support metadata;
- subscription/status placeholders only as needed.

## Phase 3 — Floor, zone, and table management

Build the visual editor:
- create/edit floors;
- upload background asset;
- add/edit zones;
- add tables;
- drag/resize;
- shape/capacity;
- zone assignment;
- soft delete/restore where supported;
- preview live floor.

Persist edits through explicit configuration endpoints.

## Phase 4 — Staff and permissions

Build:
- invite/create staff according to identity provider flow;
- role assignment;
- location assignment;
- deactivate access;
- audit access changes.

Remove routine dependence on provisioning scripts.

## Phase 5 — QR and guest action configuration

Build:
- guest action CRUD;
- ordering/enabled state;
- QR token regeneration;
- printable/exportable QR assets;
- compromised-token revocation workflow;
- preview guest page.

## Phase 6 — Device management

Build:
- registered device list;
- type/platform/app version;
- last seen;
- location assignment;
- revoke;
- display pairing workflow;
- device health indicators.

## Phase 7 — Service periods

Build first-class configuration for:
- Breakfast;
- Lunch;
- Dinner;
- custom recurring periods;
- special named periods where required.

Service periods feed analytics rather than being UI-only labels.

## Phase 8 — Operational and analytics views

Add:
- current live floor read view;
- assist status;
- current session duration;
- analytics dashboards once `11_ANALYTICS_AND_REPORTING.md` is ready.

## Phase 9 — Platform support console

Seatd operator functions:
- organisations/locations;
- status/disable;
- device status;
- feature flags if introduced;
- integration health;
- audit trail;
- system health links.

## Deliverables

- Next.js owner/admin app;
- floor editor;
- staff management;
- QR configuration/export;
- device management;
- service period management;
- platform support console.

## Definition of done

A new venue can be configured from a browser from organisation creation through a usable floor, staff access, QR setup, and display pairing without direct database intervention.
