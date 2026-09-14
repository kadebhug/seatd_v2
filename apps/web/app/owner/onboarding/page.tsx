import { redirect } from "next/navigation";
import { canAccessOwner, getSession } from "../../../lib/session";
import { OwnerOnboardingForm } from "./owner-onboarding-form";

export const dynamic = "force-dynamic";

export default async function OwnerOnboardingPage() {
  const session = await getSession();
  if (!canAccessOwner(session)) {
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
