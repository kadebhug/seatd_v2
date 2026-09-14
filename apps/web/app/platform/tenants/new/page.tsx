import { NewTenantForm } from "./new-tenant-form";

export const dynamic = "force-dynamic";

export default function NewPlatformTenantPage() {
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
