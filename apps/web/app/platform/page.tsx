import { redirect } from "next/navigation";
import { canAccessPlatform, getSession } from "../../lib/session";

export default async function PlatformPage() {
  const session = await getSession();
  if (!canAccessPlatform(session)) {
    redirect("/owner");
  }

  return (
    <main className="page">
      <section className="page-heading">
        <p>Platform</p>
        <h1>Organisation support console</h1>
      </section>
      <section className="panel">
        <h2>Status</h2>
        <p>
          Organisation list, support metadata, and subscription controls are
          reserved for the next platform phase.
        </p>
      </section>
    </main>
  );
}
