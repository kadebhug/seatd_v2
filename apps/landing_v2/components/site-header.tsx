"use client";

import { ListIcon, XIcon } from "@phosphor-icons/react";
import Link from "next/link";
import { useEffect, useId, useState } from "react";
import { BookDemoButton } from "./book-demo-button";
import { BrandLockup } from "./brand-lockup";

const nav = [
  { href: "/#walkthrough", label: "Walkthrough" },
  { href: "/#product", label: "Product" },
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#pricing", label: "Pricing" },
  { href: "/#faq", label: "FAQ" },
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
    <header className="site-header sticky top-0 z-20">
      <div className="shell grid h-16 grid-cols-[auto_1fr_auto] items-center gap-3">
        <Link className="focus-ring flex min-w-0 items-center" href="/">
          <BrandLockup />
        </Link>
        <nav
          aria-label="Primary"
          className="hidden justify-center gap-8 lg:flex"
        >
          {nav.map((item) => (
            <a
              className="focus-ring text-[0.95rem] text-fg/80 hover:text-fg"
              href={item.href}
              key={item.href}
            >
              {item.label}
            </a>
          ))}
        </nav>
        <div className="flex items-center justify-end gap-2">
          <BookDemoButton className="hidden sm:inline-flex" source="nav" />
          <button
            aria-controls={menuId}
            aria-expanded={open}
            aria-label={open ? "Close menu" : "Open menu"}
            className="focus-ring relative grid h-12 w-12 place-items-center border border-line bg-surface lg:hidden"
            onClick={() => setOpen((value) => !value)}
            type="button"
            style={{ borderRadius: 10 }}
          >
            {open ? (
              <XIcon aria-hidden="true" size={22} weight="bold" />
            ) : (
              <ListIcon aria-hidden="true" size={22} weight="bold" />
            )}
          </button>
        </div>
      </div>
      {open ? (
        <div
          className="fixed inset-x-0 top-16 bottom-0 z-30 overflow-y-auto bg-canvas overscroll-contain"
          id={menuId}
        >
          <nav aria-label="Mobile" className="shell grid gap-2 py-8">
            {nav.map((item) => (
              <a
                className="focus-ring px-1 py-4 text-2xl font-semibold"
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
