import type { OwnerSnapshotResponse } from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../lib/seatd-api";

export const dynamic = "force-dynamic";

export default async function OwnerPage() {
  const snapshot =
    await seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot");

  return (
    <main className="page">
      <section className="page-heading">
        <p>Owner workspace</p>
        <h1>{snapshot.organisation.name}</h1>
      </section>
      <section className="grid two">
        {snapshot.locations.map((location) => (
          <article className="panel" key={location.id}>
            <span className="eyebrow">{location.status}</span>
            <h2>{location.name}</h2>
            <dl className="compact-list">
              <dt>Slug</dt>
              <dd>{location.slug}</dd>
              <dt>Timezone</dt>
              <dd>{location.timezone}</dd>
            </dl>
          </article>
        ))}
      </section>
    </main>
  );
}
