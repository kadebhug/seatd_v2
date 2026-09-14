import type { PlatformAdminsResponse } from "@seatd/typescript-seatd-client";
import { seatdFetch } from "../../../lib/seatd-api";
import { AdminActions } from "./admin-actions";

export const dynamic = "force-dynamic";

export default async function PlatformAdminsPage() {
  const data = await seatdFetch<PlatformAdminsResponse>("/v1/platform/admins");

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p className="eyebrow">Platform</p>
        <h1>Admin Access</h1>
      </section>
      <AdminActions initialData={data} />
    </main>
  );
}
