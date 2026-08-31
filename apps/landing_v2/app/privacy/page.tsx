import { SiteFooter } from "../../components/site-footer";
import { SiteHeader } from "../../components/site-header";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Privacy",
  description: "How Seatd treats guest and restaurant information.",
};

export default function PrivacyPage() {
  return (
    <>
      <SiteHeader />
      <main className="shell section-anchor max-w-3xl py-24" id="main">
      <p className="text-sm text-muted">
        <Link className="focus-ring underline" href="/">
          Home
        </Link>
      </p>
      <h1 className="mt-6 text-4xl font-semibold tracking-tight">Privacy</h1>
      <div className="mt-8 grid gap-5 text-muted">
        <p>
          Guests do not need an account to request help from a table. Guest
          assistance is scoped to the table session. The basic assist flow does
          not require guest name, email, or payment details.
        </p>
        <p>
          Restaurant operations data is tenant-scoped to the organisation that
          owns the venue. Display devices use revocable tokens.
        </p>
        <p>
          Demo requests collect only the details you submit on the form so we
          can prepare a walkthrough.
        </p>
      </div>
    </main>
      <SiteFooter />
    </>
  );
}
