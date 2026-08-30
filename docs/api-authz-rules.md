# API Authentication and Authorization Rules

| Surface | Authentication mode | Authorization rule |
| --- | --- | --- |
| `/healthz` | Public | No tenant data returned |
| `/readyz` | Public | No tenant data returned |
| `/version` | Public | Build metadata only |
| Layout reads | OIDC user or trusted device | `layout.read` for organisation/location |
| Layout writes | OIDC user | `layout.write` for organisation/location |
| Table occupancy reads | OIDC user or trusted device | `operations.read` for organisation/location |
| Table occupancy writes | OIDC user or trusted device | `operations.write` for organisation/location |
| Assist request management | OIDC user or trusted device | `operations.write` for organisation/location |
| Guest QR assist request | Valid QR capability token | Token must be active; location and table must be active |
| Device registration | OIDC user | `device.manage` for organisation/location |
| Device trust/revocation | OIDC user | `device.manage` for organisation/location |
| Platform controls | OIDC user | `platform.admin`; audit event required |
| Audit reads | OIDC user | `audit.read`; platform admins may read across tenants |

All tenant-owned database access must run with an explicit transaction-local tenant context. Platform-admin context is reserved for platform control flows and credential lookup flows that must identify the tenant from the credential itself.
