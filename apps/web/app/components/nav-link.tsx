"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

export function NavLink({
  href,
  children,
}: Readonly<{ href: string; children: React.ReactNode }>) {
  const pathname = usePathname();
  // Exact-match roots so sibling sections (e.g. /platform/admins) don't
  // keep the parent Console/Overview item highlighted.
  const isActive =
    href === "/owner"
      ? pathname === "/owner"
      : href === "/platform"
        ? pathname === "/platform" ||
          pathname.startsWith("/platform/tenants")
        : pathname === href || pathname.startsWith(`${href}/`);

  return (
    <Link href={href} aria-current={isActive ? "page" : undefined}>
      {children}
    </Link>
  );
}
