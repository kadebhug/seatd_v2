import { redirect } from "next/navigation";
import { canWritePlatform, getSession } from "../../../../lib/session";
import { NewTenantForm } from "./new-tenant-form";

export const dynamic = "force-dynamic";

export default async function NewPlatformTenantPage() {
  const session = await getSession();
  if (!canWritePlatform(session)) {
    redirect("/platform");
  }

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p className="eyebrow">Platform</p>
        <h1>New Tenant</h1>
      </section>
      <NewTenantForm />
    </main>
  );
}
