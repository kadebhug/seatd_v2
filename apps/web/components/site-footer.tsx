import { BrandLockup } from "./brand-lockup";
import Link from "next/link";

const product = [
  { href: "/#walkthrough", label: "Walkthrough" },
  { href: "/#product", label: "Product" },
  { href: "/#how-it-works", label: "How it works" },
  { href: "/#pricing", label: "Pricing" },
  { href: "/#restaurants", label: "Restaurants" },
];

const company = [
  { href: "mailto:hello@seatd.app", label: "Contact" },
  { href: "/privacy", label: "Privacy" },
  { href: "/terms", label: "Terms" },
];

export function SiteFooter() {
  return (
    <footer className="border-t border-line py-16">
      <div className="shell grid gap-10 md:grid-cols-[1.4fr_1fr_1fr]">
        <div className="grid gap-4 self-start">
          <BrandLockup />
          <p className="max-w-[36ch] text-[0.95rem] text-muted">
            Restaurant floor operations. Live table status and guest assistance,
            without replacing the systems you already run.
          </p>
        </div>
        <div>
          <p className="mb-3 text-sm font-semibold">Product</p>
          <ul className="grid gap-2 text-[0.95rem] text-muted">
            {product.map((item) => (
              <li key={item.href}>
                <a className="focus-ring hover:text-fg" href={item.href}>
                  {item.label}
                </a>
              </li>
            ))}
          </ul>
        </div>
        <div>
          <p className="mb-3 text-sm font-semibold">Company</p>
          <ul className="grid gap-2 text-[0.95rem] text-muted">
            {company.map((item) => (
              <li key={item.href}>
                <Link className="focus-ring hover:text-fg" href={item.href}>
                  {item.label}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>
      <div className="shell mt-12 text-sm text-muted">
        <p>
          <span translate="no">© Seatd</span>
        </p>
      </div>
    </footer>
  );
}
