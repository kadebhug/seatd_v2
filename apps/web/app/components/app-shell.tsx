import Image from "next/image";
import Link from "next/link";
import type React from "react";
import { NavLink } from "./nav-link";

export type AppNavItem = {
  href: string;
  label: string;
};

type AppShellProps = Readonly<{
  role: "Owner" | "Platform";
  homeHref: string;
  displayName: string;
  actorRef: string;
  locationId?: string;
  showLocationSelect?: boolean;
  navItems: AppNavItem[];
  children: React.ReactNode;
}>;

export function AppShell({
  role,
  homeHref,
  displayName,
  actorRef,
  locationId,
  showLocationSelect = false,
  navItems,
  children,
}: AppShellProps) {
  return (
    <div className="app-shell">
      <aside className="sidebar">
        <Link className="brand" href={homeHref}>
          <Image
            alt=""
            aria-hidden="true"
            className="brand-symbol"
            height={32}
            src="/brand/seatd-symbol-reverse.svg"
            width={32}
          />
          <span className="brand-copy">
            <span className="brand-name" translate="no">
              Seatd
            </span>
            <span className="role-label">{role}</span>
          </span>
        </Link>
        <nav aria-label={`${role} navigation`}>
          <ul className="sidebar-nav">
            {navItems.map((item) => (
              <li key={item.href}>
                <NavLink href={item.href}>{item.label}</NavLink>
              </li>
            ))}
          </ul>
        </nav>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="topbar-identity">
            <strong>{displayName}</strong>
            <span className="font-mono tabular-nums">{actorRef}</span>
          </div>
          {showLocationSelect && locationId ? (
            <label className="location-select">
              <span className="visually-hidden">Location</span>
              <select
                autoComplete="off"
                defaultValue={locationId}
                name="locationId"
              >
                <option value={locationId}>Main Street</option>
              </select>
            </label>
          ) : null}
          <form action="/api/auth/logout" method="post">
            <button className="button secondary" type="submit">
              Sign out
            </button>
          </form>
        </header>
        {children}
      </div>
    </div>
  );
}
