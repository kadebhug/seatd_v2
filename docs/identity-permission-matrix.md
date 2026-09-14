# Identity Permission Matrix

| Role | Scope | Permissions |
| --- | --- | --- |
| `platform_admin` | Platform | `platform.admin`, `platform.admin.write`, `audit.read` |
| `support` | Platform | `audit.read` |
| `organisation_owner` | Organisation | `organisation.manage`, `location.manage`, `layout.read`, `layout.write`, `operations.read`, `operations.write`, `device.manage`, `audit.read` |
| `location_manager` | Location | `location.manage`, `layout.read`, `layout.write`, `operations.read`, `operations.write`, `device.manage` |
| `waiter` | Location | `layout.read`, `operations.read`, `operations.write` |
| `read_only` | Organisation | `layout.read`, `operations.read` |

Authentication answers who the caller is. These permissions answer what the caller may do for an organisation or location.
