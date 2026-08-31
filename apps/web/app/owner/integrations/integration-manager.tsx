"use client";

import { useMemo, useState } from "react";
import type {
  IntegrationConnection,
  IntegrationDiscrepancy,
  IntegrationDiscrepanciesResponse,
  IntegrationMappingsResponse,
  IntegrationTableMapping,
  IntegrationWebhook,
  IntegrationWebhookResult,
  IntegrationWebhooksResponse,
  Table,
} from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../../components/status-badge";

type Props = {
  initialIntegrations: IntegrationConnection[];
  initialMappings: IntegrationTableMapping[];
  initialWebhooks: IntegrationWebhook[];
  initialDiscrepancies: IntegrationDiscrepancy[];
  tables: Table[];
};

export function IntegrationManager({
  initialIntegrations,
  initialMappings,
  initialWebhooks,
  initialDiscrepancies,
  tables,
}: Props) {
  const [integrations] = useState(initialIntegrations);
  const [selectedId, setSelectedId] = useState(
    initialIntegrations[0]?.id ?? "",
  );
  const [mappings, setMappings] = useState(initialMappings);
  const [webhooks, setWebhooks] = useState(initialWebhooks);
  const [discrepancies, setDiscrepancies] = useState(initialDiscrepancies);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const selected = integrations.find((item) => item.id === selectedId);
  const tableById = useMemo(
    () => new Map(tables.map((table) => [table.id, table])),
    [tables],
  );

  async function refresh(connectionId = selectedId) {
    if (!connectionId) {
      return;
    }
    const [nextMappings, nextWebhooks, nextDiscrepancies] = await Promise.all([
      fetchJSON<IntegrationMappingsResponse>(
        `/api/owner/integrations/${connectionId}/mappings`,
      ),
      fetchJSON<IntegrationWebhooksResponse>(
        `/api/owner/integrations/${connectionId}/webhooks`,
      ),
      fetchJSON<IntegrationDiscrepanciesResponse>(
        `/api/owner/integrations/${connectionId}/discrepancies`,
      ),
    ]);
    setMappings(nextMappings.mappings);
    setWebhooks(nextWebhooks.webhooks);
    setDiscrepancies(nextDiscrepancies.discrepancies);
  }

  async function selectConnection(connectionId: string) {
    setSelectedId(connectionId);
    setMessage(null);
    setBusy(true);
    try {
      await refresh(connectionId);
    } finally {
      setBusy(false);
    }
  }

  async function updateMapping(
    mapping: IntegrationTableMapping,
    tableId: string,
  ) {
    setMessage(null);
    const status = tableId ? "mapped" : "unmapped";
    const updated = await fetchJSON<IntegrationTableMapping>(
      `/api/owner/integrations/${selectedId}/mappings/${mapping.id}`,
      {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tableId: tableId || undefined, status }),
      },
    );
    setMappings((current) =>
      current.map((item) => (item.id === updated.id ? updated : item)),
    );
  }

  async function replay(webhook: IntegrationWebhook) {
    if (
      !window.confirm(
        `Replay webhook "${webhook.externalEventId}"? This may re-apply POS state.`,
      )
    ) {
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      const result = await fetchJSON<IntegrationWebhookResult>(
        `/api/owner/integrations/${selectedId}/webhooks/${webhook.id}/replay`,
        { method: "POST" },
      );
      setMessage(
        result.applied ? "Webhook replayed." : "Webhook replay recorded.",
      );
      await refresh();
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Replay failed.");
    } finally {
      setBusy(false);
    }
  }

  async function reconcile() {
    if (
      !window.confirm(
        "Run reconciliation now? Seatd will compare POS state against table occupancy.",
      )
    ) {
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      await fetchJSON(`/api/owner/integrations/${selectedId}/reconcile`, {
        method: "POST",
      });
      setMessage("Reconciliation completed.");
      await refresh();
    } catch (error) {
      setMessage(
        error instanceof Error ? error.message : "Reconciliation failed.",
      );
    } finally {
      setBusy(false);
    }
  }

  if (integrations.length === 0) {
    return (
      <section className="panel">
        <h2>No Integrations Connected</h2>
        <p className="empty-state">
          Connect the reference POS seed or create an integration connection.
        </p>
      </section>
    );
  }

  return (
    <section className="integrations-layout">
      <aside className="panel integration-sidebar">
        <h2>Connections</h2>
        <div className="list-buttons">
          {integrations.map((integration) => (
            <button
              aria-pressed={integration.id === selectedId}
              className={integration.id === selectedId ? "selected" : ""}
              key={integration.id}
              onClick={() => void selectConnection(integration.id)}
              type="button"
            >
              <span>{integration.displayName}</span>
              <small>{integration.vendor}</small>
            </button>
          ))}
        </div>
      </aside>

      <section className="grid">
        {selected ? (
          <article className="panel integration-summary">
            <div>
              <StatusBadge label={selected.status} variant={selected.status} />
              <h2>{selected.displayName}</h2>
              <p>{selected.credentialRef}</p>
            </div>
            <dl className="compact-list">
              <dt>Status</dt>
              <dd>{selected.status}</dd>
              <dt>Last sync</dt>
              <dd>
                {selected.lastSuccessfulSyncAt
                  ? formatDate(selected.lastSuccessfulSyncAt)
                  : "Never"}
              </dd>
              <dt>Last error</dt>
              <dd>{selected.lastError ?? "None"}</dd>
            </dl>
            <button
              disabled={busy}
              onClick={() => void reconcile()}
              type="button"
            >
              {busy ? "Running…" : "Reconcile"}
            </button>
          </article>
        ) : null}

        {message ? (
          <p aria-live="polite" className="toast" role="status">
            {message}
          </p>
        ) : null}

        <article className="panel">
          <h2>Table Mappings</h2>
          <div className="table-list">
            {mappings.length > 0 ? (
              mappings.map((mapping) => (
                <label className="mapping-row" key={mapping.id}>
                  <span>
                    <strong>
                      {mapping.externalLabel ?? mapping.externalTableId}
                    </strong>
                    <small>{mapping.externalTableId}</small>
                  </span>
                  <select
                    aria-label={`Map ${mapping.externalLabel ?? mapping.externalTableId}`}
                    autoComplete="off"
                    name={`mapping-${mapping.id}`}
                    onChange={(event) =>
                      void updateMapping(mapping, event.target.value)
                    }
                    value={mapping.tableId ?? ""}
                  >
                    <option value="">Unmapped</option>
                    {tables.map((table) => (
                      <option key={table.id} value={table.id}>
                        {table.label}
                      </option>
                    ))}
                  </select>
                </label>
              ))
            ) : (
              <p className="empty-state">No table mappings yet.</p>
            )}
          </div>
        </article>

        <article className="panel">
          <h2>Webhook Inbox</h2>
          <div className="table-list">
            {webhooks.length > 0 ? (
              webhooks.map((webhook) => (
                <div className="webhook-row" key={webhook.id}>
                  <span>
                    <strong>{webhook.externalEventId}</strong>
                    <small>{formatDate(webhook.receivedAt)}</small>
                  </span>
                  <StatusBadge
                    label={webhook.processingState}
                    variant={webhook.processingState}
                  />
                  <span>
                    {webhook.signatureValid ? "Signed" : "Signature failed"}
                  </span>
                  <button
                    disabled={webhook.processingState === "processed" || busy}
                    onClick={() => void replay(webhook)}
                    type="button"
                  >
                    Replay
                  </button>
                </div>
              ))
            ) : (
              <p className="empty-state">No webhooks received yet.</p>
            )}
          </div>
        </article>

        <article className="panel">
          <h2>Reconciliation Mismatches</h2>
          <div className="table-list">
            {discrepancies.length > 0 ? (
              discrepancies.map((item) => (
                <div className="discrepancy-row" key={item.id}>
                  <span>
                    <strong>
                      {item.tableId
                        ? tableById.get(item.tableId)?.label
                        : item.externalTableId}
                    </strong>
                    <small>{item.type}</small>
                  </span>
                  <span>
                    {item.seatdState ?? "none"} {"->"}{" "}
                    {item.externalState ?? "none"}
                  </span>
                  <span>{item.resolutionState}</span>
                </div>
              ))
            ) : (
              <p className="empty-state">No mismatches recorded.</p>
            )}
          </div>
        </article>
      </section>
    </section>
  );
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, init);
  const body = await response.json();
  if (!response.ok) {
    throw new Error(body.error?.message ?? "Request failed.");
  }
  return body as T;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
