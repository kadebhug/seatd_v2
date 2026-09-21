import type { OwnerSnapshotResponse } from "@seatd/typescript-seatd-client";
import { redirect } from "next/navigation";
import { StatusBadge } from "../components/status-badge";
import { seatdFetch } from "../../lib/seatd-api";
import {
  canAccessOwner,
  getSession,
  needsOwnerSetup,
} from "../../lib/session";

export const dynamic = "force-dynamic";

export default async function OwnerPage() {
  const session = await getSession();
  if (!session.organisationId) {
    return (
      <main className="page" id="main">
        <section className="page-heading">
          <p className="eyebrow">Owner workspace</p>
          <h1>No Workspace Yet</h1>
        </section>
        <section className="panel">
          <p className="empty-state">Contact your platform administrator.</p>
        </section>
      </main>
    );
  }
  if (!canAccessOwner(session)) {
    redirect("/platform");
  }

  const snapshot =
    await seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot");
  const showFinishSetup = needsOwnerSetup(snapshot.organisation);

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p className="eyebrow">Owner workspace</p>
        <h1>{snapshot.organisation.name}</h1>
        {showFinishSetup ? (
          <a className="button-link secondary" href="/owner/onboarding">
            Finish Setup
          </a>
        ) : null}
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
