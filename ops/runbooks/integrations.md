# Integrations Runbook

Use this when `SeatdIntegrationWebhookFailures`,
`SeatdIntegrationReconciliationDiscrepancies`, or operator reports show POS or
external integration state diverging from Seatd.

## Dashboard

Use `ops/grafana/dashboards/seatd-operations-health.json`.

## Alerts

- `SeatdIntegrationWebhookFailures`: webhook processing failed recently or failed inbox rows exist.
- `SeatdIntegrationReconciliationDiscrepancies`: Seatd and external integration state disagree.

## Webhook Failures

1. Open Owner -> Integrations.
2. Check webhook rate by vendor and outcome.
3. Check the webhook inbox for failed or skipped records.
4. Signature failures mean the vendor secret or `SEATD_REFERENCE_POS_WEBHOOK_SECRET` is wrong.
5. Replay only after the mapping and credential issue is corrected.

## Duplicate Webhooks

Duplicate `vendor + externalEventId` webhooks are idempotent. The original
inbox row is returned and Seatd table state is not mutated a second time.

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

## Mitigation

1. Correct bad secrets, disabled connections, or table mappings before replay.
2. Trigger reconciliation after mapping changes or missed webhook windows.
3. If reconciliation repeatedly produces discrepancies, freeze automatic correction for that connection if supported and escalate to the integration owner.
4. If outbox delivery also lags, follow `outbox-health.md`.

## Escalation

Escalate when a vendor integration repeatedly fails valid signatures, when
reconciliation finds unresolved discrepancies for active service, or when replay
would risk applying stale state.

## Post-Incident

- Record vendor, connection id, webhook outcome, and discrepancy type.
- Confirm whether data was corrected by replay, reconciliation, or manual operator action.
- Add a vendor-specific section if the incident exposed provider-specific behavior.
