import type { OwnerSnapshotResponse } from "@seatd/typescript-seatd-client";
import { redirect } from "next/navigation";
import { seatdFetch } from "../../../lib/seatd-api";
import {
  canAccessOwner,
  getSession,
  needsOwnerSetup,
} from "../../../lib/session";
import { OwnerOnboardingForm } from "./owner-onboarding-form";

export const dynamic = "force-dynamic";

export default async function OwnerOnboardingPage() {
  const session = await getSession();
  if (!session.organisationId || !canAccessOwner(session)) {
    redirect("/owner");
  }

  const snapshot =
    await seatdFetch<OwnerSnapshotResponse>("/v1/owner/snapshot");
  if (!needsOwnerSetup(snapshot.organisation)) {
    redirect("/owner");
  }

  return (
    <main className="page onboarding-page" id="main">
      <section className="page-heading">
        <p className="eyebrow">First run</p>
        <h1>Finish Setup</h1>
      </section>
      <OwnerOnboardingForm displayName={session.displayName} />
    </main>
  );
}
