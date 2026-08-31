import type {
  OwnerSnapshotResponse,
  ServicePeriodsResponse,
} from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../../lib/seatd-api";
import { ConfigurationForms } from "./configuration-forms";

export const dynamic = "force-dynamic";

export default async function ConfigurationPage() {
  const [snapshot, servicePeriods] = await Promise.all([
    seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot"),
    seatdFetch<ServicePeriodsResponse>("/v1/service-periods"),
  ]);

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p>Configuration</p>
        <h1>Organisation and location settings</h1>
      </section>
      <ConfigurationForms
        snapshot={snapshot}
        servicePeriods={servicePeriods.servicePeriods}
      />
    </main>
  );
}
