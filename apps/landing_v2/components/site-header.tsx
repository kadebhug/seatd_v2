"use client";

import { ListIcon, XIcon } from "@phosphor-icons/react";
import Link from "next/link";
import { useEffect, useId, useState } from "react";
import { BookDemoButton } from "./book-demo-button";
import { BrandLockup } from "./brand-lockup";

const nav = [
  { href: "/#product", label: "Product" },
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#restaurants", label: "Restaurants" },
  { href: "/#pricing", label: "Pricing" },
];

export function SiteHeader() {
  const [open, setOpen] = useState(false);
  const menuId = useId();

  useEffect(() => {
    document.body.style.overflow = open ? "hidden" : "";
    return () => {
      document.body.style.overflow = "";
    };
  }, [open]);

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <header className="pointer-events-none sticky top-0 z-20 pt-3">
      <div className="shell">
        <div className="pointer-events-auto relative z-40 mx-auto flex h-16 max-w-[1400px] items-center justify-between rounded-[20px] border border-line bg-canvas/88 px-4 shadow-[0_10px_40px_color-mix(in_srgb,var(--color-ink)_8%,transparent)] backdrop-blur-md md:px-5">
          <Link className="focus-ring flex min-w-0 items-center" href="/">
            <BrandLockup />
          </Link>
          <nav
            aria-label="Primary"
            className="hidden items-center gap-7 lg:flex"
          >
            {nav.map((item) => (
              <a
                className="focus-ring text-[0.95rem] font-medium text-fg/80 hover:text-fg"
                href={item.href}
                key={item.href}
              >
                {item.label}
              </a>
            ))}
          </nav>
          <div className="flex items-center gap-2">
            <BookDemoButton className="hidden sm:inline-flex" source="nav" />
            <button
              aria-controls={menuId}
              aria-expanded={open}
              aria-label={open ? "Close menu" : "Open menu"}
              className="focus-ring relative grid h-12 w-12 place-items-center rounded-full border border-line bg-surface lg:hidden"
              onClick={() => setOpen((value) => !value)}
              type="button"
            >
              {open ? (
                <XIcon aria-hidden="true" size={22} weight="bold" />
              ) : (
                <ListIcon aria-hidden="true" size={22} weight="bold" />
              )}
            </button>
          </div>
        </div>
      </div>
      {open ? (
        <div
          className="pointer-events-auto fixed inset-x-0 top-[5.25rem] bottom-0 z-30 overflow-y-auto bg-canvas/94 backdrop-blur-xl overscroll-contain"
          id={menuId}
        >
          <nav aria-label="Mobile" className="shell grid gap-2 py-8">
            {nav.map((item) => (
              <a
                className="focus-ring rounded-[16px] px-4 py-4 text-2xl font-semibold"
                href={item.href}
                key={item.href}
                onClick={() => setOpen(false)}
              >
                {item.label}
              </a>
            ))}
            <div className="pt-4">
              <BookDemoButton source="nav-mobile" />
            </div>
          </nav>
        </div>
      ) : null}
    </header>
  );
}
