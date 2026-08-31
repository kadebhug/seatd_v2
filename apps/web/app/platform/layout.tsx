import { redirect } from "next/navigation";
import type React from "react";
import { AppShell } from "../components/app-shell";
import {
  canAccessOwner,
  canAccessPlatform,
  getSession,
} from "../../lib/session";

export default async function PlatformLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  const session = await getSession();
  if (!canAccessPlatform(session)) {
    redirect("/owner");
  }

  const navItems = [{ href: "/platform", label: "Console" }];
  if (canAccessOwner(session)) {
    navItems.push({ href: "/owner", label: "Owner workspace" });
  }

  return (
    <AppShell
      actorRef={session.actorRef}
      displayName={session.displayName}
      homeHref="/platform"
      navItems={navItems}
      role="Platform"
    >
      {children}
    </AppShell>
  );
}
