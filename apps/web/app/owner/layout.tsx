import { redirect } from "next/navigation";
import type React from "react";
import { AppShell } from "../components/app-shell";
import {
  canAccessOwner,
  canAccessPlatform,
  getSession,
  needsOwnerOnboarding,
} from "../../lib/session";

const ownerNav = [
  { href: "/owner", label: "Overview" },
  { href: "/owner/analytics", label: "Analytics" },
  { href: "/owner/configuration", label: "Configuration" },
  { href: "/owner/layout", label: "Floor editor" },
  { href: "/owner/devices", label: "Devices" },
  { href: "/owner/integrations", label: "Integrations" },
];

export default async function OwnerLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const session = await getSession();
  if (!canAccessOwner(session) && !needsOwnerOnboarding(session)) {
    redirect("/");
  }

  const navItems = needsOwnerOnboarding(session)
    ? [{ href: "/owner/onboarding", label: "Onboarding" }]
    : [...ownerNav];
  if (canAccessPlatform(session)) {
    navItems.push({ href: "/platform", label: "Platform" });
  }

  return (
    <AppShell
      actorRef={session.actorRef}
      displayName={session.displayName}
      homeHref="/owner"
      locationId={session.locationId}
      navItems={navItems}
      role="Owner"
      showLocationSelect
    >
      {children}
    </AppShell>
  );
}
