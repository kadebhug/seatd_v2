import { SiteFooter } from "../../components/site-footer";
import { SiteHeader } from "../../components/site-header";
import type { Metadata } from "next";
import Link from "next/link";

export const metadata: Metadata = {
  title: "Terms",
  description: "Seatd marketing site terms.",
};

export default function TermsPage() {
  return (
    <>
      <SiteHeader />
      <main className="shell section-anchor max-w-3xl py-24" id="main">
        <p className="text-sm text-muted">
          <Link className="focus-ring underline" href="/">
            Home
          </Link>
        </p>
        <h1 className="mt-6 text-4xl font-semibold tracking-tight">Terms</h1>
        <div className="mt-8 grid gap-5 text-muted">
          <p>
            This marketing site describes Seatd’s floor operations product. A
            demo request is a conversation, not a purchase. Product access,
            pricing, and onboarding are confirmed during that walkthrough.
          </p>
          <p>
            The live floor, guest QR, and waiter views shown here use a
            demonstration restaurant (Marlowe’s) so you can see how the product
            behaves during service.
          </p>
        </div>
      </main>
      <SiteFooter />
    </>
  );
}
