import type { OwnerSnapshotResponse } from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../components/status-badge";
import { seatdFetch } from "../../lib/seatd-api";

export const dynamic = "force-dynamic";

export default async function OwnerPage() {
  const snapshot =
    await seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot");

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p className="eyebrow">Owner workspace</p>
        <h1>{snapshot.organisation.name}</h1>
      </section>
      <section className="panel location-list">
        {snapshot.locations.length > 0 ? (
          snapshot.locations.map((location) => (
            <article className="location-row" key={location.id}>
              <StatusBadge label={location.status} variant={location.status} />
              <h2>{location.name}</h2>
              <dl className="compact-list">
                <dt>Slug</dt>
                <dd className="font-mono">{location.slug}</dd>
                <dt>Timezone</dt>
                <dd>{location.timezone}</dd>
              </dl>
            </article>
          ))
        ) : (
          <p className="empty-state">No locations configured yet.</p>
        )}
      </section>
    </main>
  );
}
