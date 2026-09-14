# Platform Admin Audit Policy

Platform admin actions require an explicit permission check for `platform.admin`. Mutating actions (tenant suspend/reactivate and future write actions) require the stricter `platform.admin.write` permission instead — `platform.admin` alone grants read access only.

Actions that affect organisations, locations, devices, support access, kill switches, or security settings must record an `audit_events` row with:

- `actor_ref`
- `action`
- `target_type`
- `target_id`
- organisation/location context when applicable
- non-sensitive JSON metadata

Mutating platform actions (e.g. `platform.tenant_suspend`, `platform.tenant_reactivate`) additionally require a non-empty `reason` string from the request body, recorded under a `reason` key in the audit metadata.

Support impersonation is not enabled. If added later, it must be visibly marked in the product and every impersonated action must include both support actor and effective tenant actor in audit metadata.

Device, organisation, and location disable controls must be reversible only through audited platform-admin or owner workflows.
