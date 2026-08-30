# QR Capability Threat Model

## Asset

QR capabilities allow a guest at a table to perform narrowly scoped actions such as requesting assistance. A token is not guest identity and must not grant staff or management permissions.

## Main Threats

| Threat | Control |
| --- | --- |
| Token guessing | Generate high-entropy opaque tokens; store lookup prefixes and compare hashes |
| Token leakage through logs | Redact sensitive query/header values at proxies and application boundaries |
| Replay after rotation | Support revocation and expiry through `revoked_at` and `expires_at` |
| Cross-location use | Capability rows carry organisation, location, and table IDs and are protected by RLS |
| Disabled table/location use | Capability lookup joins active locations and tables |
| Request flooding | Add route-level rate limits and per-action cooldowns before guest endpoints are exposed |
| PII collection | Do not collect guest PII by default |

## Accepted Scope

The current model supports prefix lookup, hashed storage, revocation, expiry, and active table/location validation. Public endpoint rate limiting and cooldown enforcement must be implemented with the guest QR API surface.
