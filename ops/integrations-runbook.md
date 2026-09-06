# Integrations Runbook

The production incident runbook lives at
[runbooks/integrations.md](runbooks/integrations.md). This file remains as a
stable pointer for older links and local notes.

## Webhook Failures

1. Open Owner -> Integrations.
2. Check the webhook inbox for failed or skipped records.
3. Signature failures mean the vendor secret or `SEATD_REFERENCE_POS_WEBHOOK_SECRET` is wrong.
4. Replay only after the mapping and credential issue is corrected.

## Duplicate Webhooks

Duplicate `vendor + externalEventId` webhooks are idempotent. The original inbox row is returned and Seatd table state is not mutated a second time.

## Unmapped Tables

1. Open the table mappings section.
2. Assign the external table id to the correct Seatd table.
3. Replay the webhook or trigger reconciliation.

## Reconciliation Mismatches

- `occupancy_mismatch` with `auto_corrected` was corrected by Seatd.
- `ambiguous_conflict` requires operator review because an active assist exists.
- `unmapped_table` requires a table mapping before Seatd can apply external state.

## Reference POS Webhook

Send webhooks to:

```text
POST /v1/integrations/reference_pos/webhooks
X-Seatd-Integration-Connection-ID: 77777777-7777-7777-7777-777777777777
X-Seatd-Integration-Signature: <hex hmac sha256 over raw JSON body>
```

Example body:

```json
{"eventId":"evt-local-1","externalTableId":"ref-t1","status":"occupied","partySize":4}
```
