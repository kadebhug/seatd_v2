import type {
  IntegrationDiscrepanciesResponse,
  IntegrationMappingsResponse,
  IntegrationsResponse,
  IntegrationWebhooksResponse,
  LayoutEditorSnapshotResponse,
} from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../../lib/seatd-api";
import { IntegrationManager } from "./integration-manager";

export const dynamic = "force-dynamic";

export default async function IntegrationsPage() {
  const integrations =
    await seatdFetch<IntegrationsResponse>("/v1/integrations");
  const selected = integrations.integrations[0];
  const [layout, mappings, webhooks, discrepancies] = selected
    ? await Promise.all([
        seatdFetch<LayoutEditorSnapshotResponse>("/v1/layout/editor-snapshot"),
        seatdFetch<IntegrationMappingsResponse>(
          `/v1/integrations/${selected.id}/mappings`,
        ),
        seatdFetch<IntegrationWebhooksResponse>(
          `/v1/integrations/${selected.id}/webhooks`,
        ),
        seatdFetch<IntegrationDiscrepanciesResponse>(
          `/v1/integrations/${selected.id}/discrepancies`,
        ),
      ])
    : [
        { floors: [], zones: [], tables: [] },
        { mappings: [] },
        { webhooks: [] },
        { discrepancies: [] },
      ];

  return (
    <main className="page">
      <section className="page-heading">
        <p>Integrations</p>
        <h1>POS reconciliation and webhook health</h1>
      </section>
      <IntegrationManager
        initialIntegrations={integrations.integrations}
        initialMappings={mappings.mappings}
        initialWebhooks={webhooks.webhooks}
        initialDiscrepancies={discrepancies.discrepancies}
        tables={layout.tables.map((state) => state.table)}
      />
    </main>
  );
}
