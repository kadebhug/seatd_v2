import { redirect } from "next/navigation";
import { getSession, needsOwnerOnboarding } from "../../../lib/session";
import { OwnerOnboardingForm } from "./owner-onboarding-form";

export const dynamic = "force-dynamic";

export default async function OwnerOnboardingPage() {
  const session = await getSession();
  if (!needsOwnerOnboarding(session)) {
    redirect("/owner");
  }

  return (
    <main className="page onboarding-page" id="main">
      <section className="page-heading">
        <p className="eyebrow">First run</p>
        <h1>Create your venue</h1>
      </section>
      <OwnerOnboardingForm displayName={session.displayName} />
    </main>
  );
}
