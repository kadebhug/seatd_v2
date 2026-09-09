import type {
  MembershipsResponse,
  OwnerSnapshotResponse,
} from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../../lib/seatd-api";
import { MemberManager } from "./member-manager";

export const dynamic = "force-dynamic";

export default async function MembersPage() {
  const [memberships, snapshot] = await Promise.all([
    seatdFetch<MembershipsResponse>("/v1/memberships?includeDisabled=true"),
    seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot"),
  ]);

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p>Members</p>
        <h1>Organisation and location access</h1>
      </section>
      <MemberManager
        initialMemberships={memberships.memberships}
        locations={snapshot.locations}
      />
    </main>
  );
}
