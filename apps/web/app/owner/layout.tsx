import Link from "next/link";
import { redirect } from "next/navigation";
import type React from "react";
import { canAccessOwner, getSession } from "../../lib/session";

export default async function OwnerLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const session = await getSession();
  if (!canAccessOwner(session)) {
    redirect("/");
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link className="brand" href="/owner">
          Seatd
        </Link>
        <nav>
          <Link href="/owner">Overview</Link>
          <Link href="/owner/analytics">Analytics</Link>
          <Link href="/owner/configuration">Configuration</Link>
          <Link href="/owner/layout">Floor editor</Link>
          <Link href="/owner/devices">Devices</Link>
          <Link href="/owner/integrations">Integrations</Link>
          <Link href="/platform">Platform</Link>
        </nav>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div>
            <strong>{session.displayName}</strong>
            <span>{session.actorRef}</span>
          </div>
          <select defaultValue={session.locationId} aria-label="Location">
            <option value={session.locationId}>Main Street</option>
          </select>
        </header>
        {children}
      </div>
    </div>
  );
}
