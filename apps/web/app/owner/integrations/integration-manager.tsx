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
    await refresh(connectionId);
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
    setMessage(null);
    const result = await fetchJSON<IntegrationWebhookResult>(
      `/api/owner/integrations/${selectedId}/webhooks/${webhook.id}/replay`,
      { method: "POST" },
    );
    setMessage(result.applied ? "Webhook replayed" : "Webhook replay recorded");
    await refresh();
  }

  async function reconcile() {
    setMessage(null);
    await fetchJSON(`/api/owner/integrations/${selectedId}/reconcile`, {
      method: "POST",
    });
    setMessage("Reconciliation completed");
    await refresh();
  }

  if (integrations.length === 0) {
    return (
      <section className="panel">
        <h2>No integrations connected</h2>
        <p className="eyebrow">
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
              className={integration.id === selectedId ? "selected" : ""}
              key={integration.id}
              onClick={() => void selectConnection(integration.id)}
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
              <span className={`status-dot ${selected.status}`} />
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
            <button onClick={() => void reconcile()}>Reconcile</button>
          </article>
        ) : null}

        {message ? <p className="toast">{message}</p> : null}

        <article className="panel">
          <h2>Table mappings</h2>
          <div className="table-list">
            {mappings.map((mapping) => (
              <label className="mapping-row" key={mapping.id}>
                <span>
                  <strong>
                    {mapping.externalLabel ?? mapping.externalTableId}
                  </strong>
                  <small>{mapping.externalTableId}</small>
                </span>
                <select
                  value={mapping.tableId ?? ""}
                  onChange={(event) =>
                    void updateMapping(mapping, event.target.value)
                  }
                >
                  <option value="">Unmapped</option>
                  {tables.map((table) => (
                    <option key={table.id} value={table.id}>
                      {table.label}
                    </option>
                  ))}
                </select>
              </label>
            ))}
          </div>
        </article>

        <article className="panel">
          <h2>Webhook inbox</h2>
          <div className="table-list">
            {webhooks.map((webhook) => (
              <div className="webhook-row" key={webhook.id}>
                <span>
                  <strong>{webhook.externalEventId}</strong>
                  <small>{formatDate(webhook.receivedAt)}</small>
                </span>
                <span>{webhook.processingState}</span>
                <span>
                  {webhook.signatureValid ? "signed" : "signature failed"}
                </span>
                <button
                  disabled={webhook.processingState === "processed"}
                  onClick={() => void replay(webhook)}
                >
                  Replay
                </button>
              </div>
            ))}
          </div>
        </article>

        <article className="panel">
          <h2>Reconciliation mismatches</h2>
          <div className="table-list">
            {discrepancies.map((item) => (
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
            ))}
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
    throw new Error(body.error?.message ?? "Request failed");
  }
  return body as T;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
