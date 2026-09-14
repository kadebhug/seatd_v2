import Link from "next/link";
import type {
  PlatformTenantDetail,
  PlatformTenantsResponse,
} from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../components/status-badge";
import { seatdFetch } from "../../lib/seatd-api";
import { TenantDetailPanel } from "./tenant-detail-panel";

export const dynamic = "force-dynamic";

type SearchParams = {
  q?: string;
  tenantId?: string;
};

export default async function PlatformPage({
  searchParams,
}: Readonly<{ searchParams: Promise<SearchParams> }>) {
  const params = await searchParams;
  const query = params.q?.trim() ?? "";
  const tenants = await seatdFetch<PlatformTenantsResponse>(
    `/v1/platform/tenants?${new URLSearchParams({ q: query })}`,
  );
  const selectedID = params.tenantId ?? tenants.tenants[0]?.id;
  const detail = selectedID
    ? await seatdFetch<PlatformTenantDetail>(
        `/v1/platform/tenants/${selectedID}`,
      )
    : null;

  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p className="eyebrow">Platform</p>
        <h1>Tenant Operations</h1>
      </section>

      <section className="platform-console">
        <aside className="panel platform-search">
          <form action="/platform" className="platform-search-form">
            <label>
              Tenant search
              <input
                autoComplete="off"
                defaultValue={query}
                name="q"
                placeholder="Name or slug"
              />
            </label>
            <button type="submit">Search</button>
          </form>
          <div className="platform-tenant-list">
            {tenants.tenants.length > 0 ? (
              tenants.tenants.map((tenant) => (
                <Link
                  aria-current={tenant.id === selectedID ? "page" : undefined}
                  className="platform-tenant-link"
                  href={`/platform?${new URLSearchParams({
                    q: query,
                    tenantId: tenant.id,
                  })}`}
                  key={tenant.id}
                >
                  <span>
                    <strong>{tenant.name}</strong>
                    <small className="font-mono">{tenant.slug}</small>
                  </span>
                  <StatusBadge label={tenant.status} variant={tenant.status} />
                </Link>
              ))
            ) : (
              <p className="empty-state">No tenants match this search.</p>
            )}
          </div>
        </aside>

        {detail ? (
          <TenantDetailPanel initialDetail={detail} />
        ) : (
          <section className="panel">
            <p className="empty-state">No tenants are available.</p>
          </section>
        )}
      </section>
    </main>
  );
}
