import { CheckIcon } from "@phosphor-icons/react/dist/ssr";
import Image from "next/image";
import { BookDemoButton } from "../components/book-demo-button";
import { Clipboard } from "../components/clipboard";
import { DemoForm } from "../components/demo-form";
import { EntranceBoard } from "../components/entrance-board";
import { FaqList } from "../components/faq-list";
import { GuestAssistStage } from "../components/guest-assist-stage";
import { HeroClipboard } from "../components/hero-clipboard";
import { LiveFloor } from "../components/live-floor";
import { SalesWalkthrough } from "../components/sales-walkthrough";
import { SiteFooter } from "../components/site-footer";
import { SiteHeader } from "../components/site-header";
import { StickyDemoBar } from "../components/sticky-demo-bar";
import { ZoneFloor } from "../components/zone-floor";
import { DEMO_VENUE, MAIN_FIXTURES, MAIN_FLOOR } from "../lib/demo-floor";

const trust = [
  "Built for busy hospitality teams",
  "No guest app required",
  "Works alongside your existing systems",
  "Live floor updates",
  "Simple table QR experience",
];

const problems = [
  {
    title: "Staff are not always sure which tables are free.",
    body: "Availability lives in someone’s head, or in a walk between the patio and the main floor.",
  },
  {
    title: "Guests struggle to get attention when the room is full.",
    body: "Waving across a busy section is slow for the guest and noisy for the floor.",
  },
  {
    title: "Sections drift out of sync.",
    body: "Bar, dining, and outdoor seating become separate pictures of the same service.",
  },
  {
    title: "Managers see the floor, not the pattern.",
    body: "You can feel a tough Saturday. It is harder to see where attention actually went.",
  },
];

const roles = [
  { who: "Walk-in guest", need: "Sees where seating is available" },
  { who: "Seated guest", need: "Asks for help from the table" },
  { who: "Waiter", need: "Works from a live floor" },
  { who: "Owner", need: "Configures the venue and stays visible" },
];

const apps = [
  {
    name: "Owner",
    detail: "Map floors, zones, and table QR actions to match how you run service.",
  },
  {
    name: "Waiter",
    detail: "Handheld live floor with table status and guest requests in one view.",
  },
  {
    name: "Display",
    detail: "Entrance board showing availability across dining, patio, bar, and lounge.",
  },
  {
    name: "Guest",
    detail: "Browser table QR — no download, account, or loyalty signup required.",
  },
];

const steps = [
  {
    title: "Map your restaurant",
    body: "Create floors and tables that match the physical venue.",
  },
  {
    title: "Give your team access",
    body: "Staff use the live floor during service.",
  },
  {
    title: "Add table QR codes",
    body: "Guests request help without installing anything.",
  },
  {
    title: "Run service",
    body: "Table activity and guest requests stay visible in one place.",
  },
];

const outcomes = [
  {
    title: "Better floor visibility",
    body: "Your team knows what is happening without checking every section in person.",
  },
  {
    title: "Faster guest response",
    body: "Requests show staff exactly which table needs attention.",
  },
  {
    title: "Easier coordination",
    body: "Front-of-house shares one operational picture across the shift.",
  },
  {
    title: "Better use of seating",
    body: "See availability across floors, patios, bars, and lounges from one view.",
  },
];

const venues = [
  {
    title: "High-volume restaurants",
    body: "Casual dining rooms where table state changes all night.",
    image: "/images/seatd-dining-room.webp",
    alt: "Dining room with rows of dark timber tables and bone plaster walls",
  },
  {
    title: "Pubs and sports bars",
    body: "Seating and guest attention that move with the game and the bar.",
    image: "/images/seatd-bar.webp",
    alt: "Bar with stools, high-tops, and a long timber counter",
  },
  {
    title: "Multi-area venues",
    body: "Patio, deck, lounge, and upstairs seating that need the same picture.",
    image: "/images/seatd-patio.webp",
    alt: "Covered restaurant patio with timber tables and garden beyond",
  },
  {
    title: "Restaurant groups",
    body: "Operators who want the same floor language across more than one site.",
    image: "/images/seatd-lounge.webp",
    alt: "Upstairs lounge with low seating and warm lamps",
  },
];

const pricingTiers = [
  {
    title: "Small room",
    scope: "Under 25 tables",
    includes: [
      "Live visual floor",
      "Table QR guest assistance",
      "Owner and waiter access",
      "One service area",
    ],
  },
  {
    title: "Full floor",
    scope: "25–50 tables",
    includes: [
      "Everything in Small room",
      "Multi-zone awareness",
      "Entrance availability display",
      "Patio, bar, and lounge views",
    ],
    featured: true,
  },
  {
    title: "Multi-location",
    scope: "Groups and rollout",
    includes: [
      "Everything in Full floor",
      "Same floor language across sites",
      "Rollout and onboarding support",
      "Per-location configuration",
    ],
  },
];

export default function HomePage() {
  return (
    <>
      <SiteHeader />
      <StickyDemoBar />
      <main id="main">
        <Hero />
        <Trust />
        <SalesGuide />
        <Problem />
        <Promise />
        <Product />
        <Scenarios />
        <HowItWorks />
        <GuestAssist />
        <LiveOps />
        <Outcomes />
        <Who />
        <Beside />
        <Pricing />
        <Faq />
        <FinalCta />
      </main>
      <SiteFooter />
    </>
  );
}

function Hero() {
  return (
    <section className="grid min-h-[100dvh] lg:grid-cols-[minmax(0,0.92fr)_minmax(0,1.18fr)]">
      <div className="flex flex-col justify-center px-[max(1.25rem,env(safe-area-inset-left))] py-10 md:px-12 lg:pl-[max(2rem,calc((100vw-1400px)/2+1.25rem))] lg:pr-10">
        <h1 className="display max-w-[14ch] text-4xl md:text-5xl lg:text-[3.4rem]">
          Know what’s happening on your floor
          <span className="period">.</span>
        </h1>
        <p className="mt-5 max-w-[34ch] text-lg text-muted">
          See available, occupied, and attention tables without walking the
          room. Built for owners who run busy floors, not another POS.
        </p>
        <div className="mt-7 flex flex-wrap items-center gap-3">
          <BookDemoButton source="hero" />
          <a className="cta-secondary" data-cta="see-how" href="/#walkthrough">
            Walk the page
          </a>
        </div>
      </div>
      <HeroClipboard />
    </section>
  );
}

function Trust() {
  return (
    <section className="border-y border-line py-6">
      <ul className="shell grid gap-x-8 gap-y-3 sm:grid-cols-2 lg:flex lg:flex-wrap">
        {trust.map((item) => (
          <li className="trust-item max-w-[28ch]" key={item}>
            <CheckIcon aria-hidden="true" size={18} weight="bold" />
            {item}
          </li>
        ))}
      </ul>
    </section>
  );
}

function SalesGuide() {
  return (
    <section className="section-anchor py-16 md:py-24" id="walkthrough">
      <div className="shell grid gap-10 lg:grid-cols-[0.85fr_1.15fr] lg:items-start">
        <div>
          <h2 className="display max-w-[14ch] text-3xl md:text-4xl">
            A page sales can walk on a call
            <span className="period">.</span>
          </h2>
          <p className="mt-5 max-w-[42ch] text-muted">
            Five beats, one story: the problem, the live floor, guest proof,
            positioning beside POS, and the demo close. Scroll or jump — each
            beat links to the section below.
          </p>
        </div>
        <SalesWalkthrough />
      </div>
    </section>
  );
}

function Problem() {
  return (
    <section className="section-anchor py-24 md:py-32" id="problem">
      <div className="shell grid items-center gap-10 lg:grid-cols-[1.15fr_0.85fr]">
        <div className="relative min-h-[320px] overflow-hidden lg:min-h-[560px]">
          <Image
            alt="Service aisle between restaurant tables during a busy period"
            className="object-cover"
            fill
            sizes="(max-width: 1024px) 100vw, 55vw"
            src="/images/seatd-service-aisle.webp"
          />
        </div>
        <div>
          <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
            Busy service shouldn’t depend on everyone remembering everything
            <span className="period">.</span>
          </h2>
          <ul className="mt-10 grid gap-6">
            {problems.map((item) => (
              <li key={item.title}>
                <h3 className="font-sans text-lg font-semibold tracking-normal">
                  {item.title}
                </h3>
                <p className="mt-1 max-w-[48ch] text-muted">{item.body}</p>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  );
}

function Promise() {
  return (
    <section className="border-y border-line py-24 md:py-32">
      <div className="shell">
        <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
          One shared view of your restaurant floor
          <span className="period">.</span>
        </h2>
        <p className="mt-5 max-w-[54ch] text-muted">
          Seatd connects the people who need to know what is happening, without
          asking guests to join another app.
        </p>
        <div className="mt-12 grid gap-px bg-line md:grid-cols-4">
          {roles.map((role) => (
            <article className="bg-canvas py-6 md:px-6 md:py-8" key={role.who}>
              <h3 className="font-sans text-xl font-semibold tracking-normal">
                {role.who}
              </h3>
              <p className="mt-2 text-muted">{role.need}</p>
            </article>
          ))}
        </div>
        <div className="mt-12 grid gap-px bg-line md:grid-cols-2 xl:grid-cols-4">
          {apps.map((app) => (
            <article className="app-tile bg-surface px-6 py-7" key={app.name}>
              <h3 className="font-sans text-lg font-semibold">{app.name}</h3>
              <p className="mt-2 text-sm text-muted">{app.detail}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function Product() {
  return (
    <section className="section-anchor py-24 md:py-32" id="product">
      <div className="shell grid gap-12">
        <div>
          <h2 className="display max-w-[14ch] text-3xl md:text-5xl">
            See the whole floor at a glance
            <span className="period">.</span>
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            Give staff a live visual view of which tables are available,
            occupied, or need attention across every service area. Less walking
            around just to find out what is happening.
          </p>
        </div>
        <Clipboard className="mx-auto w-full max-w-5xl" stamp="Live floor">
          <LiveFloor
            animate
            area="Main floor"
            fixtures={MAIN_FIXTURES}
            tables={MAIN_FLOOR}
            venue={DEMO_VENUE}
          />
        </Clipboard>
        <div>
          <BookDemoButton source="after-product" />
        </div>
      </div>
    </section>
  );
}

function Scenarios() {
  return (
    <section className="section-anchor pb-8" id="scenarios">
      <div className="shell grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <article className="relative min-h-[420px] overflow-hidden">
          <Image
            alt="Restaurant entrance looking through to the dining room"
            className="object-cover"
            fill
            sizes="(max-width: 1024px) 100vw, 60vw"
            src="/images/seatd-entrance.webp"
          />
          <div className="absolute inset-0 bg-[linear-gradient(180deg,transparent_30%,#1c1410_100%)]" />
          <div className="absolute inset-x-0 bottom-0 grid gap-5 p-6 text-paper md:p-8">
            <div>
              <h3 className="display text-2xl tracking-tight">
                A guest walks in
              </h3>
              <p className="mt-2 max-w-[42ch] text-paper/75">
                The entrance display shows where seating is available, including
                patio and bar, without sending the host on a tour.
              </p>
            </div>
            <EntranceBoard />
          </div>
        </article>
        <div className="grid gap-6">
          <article className="border border-line bg-surface p-6 md:p-8">
            <h3 className="display text-2xl tracking-tight">During service</h3>
            <p className="mt-2 max-w-[42ch] text-muted">
              The waiter sees which tables are occupied and where attention is
              needed, from a handheld live floor.
            </p>
          </article>
          <article className="relative min-h-[240px] overflow-hidden">
            <Image
              alt="Covered patio seating at a restaurant"
              className="object-cover"
              fill
              sizes="(max-width: 1024px) 100vw, 40vw"
              src="/images/seatd-patio.webp"
            />
            <div className="absolute inset-0 bg-[linear-gradient(180deg,transparent_20%,#1c1410_100%)]" />
            <div className="absolute inset-x-0 bottom-0 p-6 text-paper">
              <h3 className="display text-2xl tracking-tight">At the table</h3>
              <p className="mt-2 max-w-[36ch] text-paper/75">
                The guest scans the table QR, requests the bill, and staff see
                Table 14 waiting.
              </p>
            </div>
          </article>
        </div>
      </div>
    </section>
  );
}

function HowItWorks() {
  return (
    <section className="section-anchor py-24 md:py-32" id="how-it-works">
      <div className="shell">
        <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
          Seatd works alongside the systems you already use
          <span className="period">.</span>
        </h2>
        <p className="mt-5 max-w-[58ch] text-lg text-muted">
          You do not replace the POS to see the floor. Map the room, give the
          team access, put QR codes on tables, and run service from one view.
        </p>
        <ol className="mt-14 grid gap-10 md:grid-cols-2 xl:grid-cols-4">
          {steps.map((step) => (
            <li key={step.title}>
              <h3 className="font-sans text-xl font-semibold tracking-normal">
                {step.title}
              </h3>
              <p className="mt-2 max-w-[36ch] text-muted">{step.body}</p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  );
}

function GuestAssist() {
  return (
    <section
      className="section-anchor border-t border-line py-24 md:py-32"
      id="guest-assist"
    >
      <div className="shell grid gap-12">
        <div>
          <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
            No app. No account. Just scan and ask
            <span className="period">.</span>
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            Guests request help in seconds from the table QR. Staff see where
            attention is needed. Guests do not download an app, create an
            account, enter an email, or join a loyalty programme just to ask
            for assistance.
          </p>
        </div>
        <GuestAssistStage />
      </div>
    </section>
  );
}

function LiveOps() {
  return (
    <section className="desk-stage timber-field section-anchor py-24 text-paper md:py-32">
      <div className="shell grid gap-12 md:gap-16">
        <div>
          <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
            Your floor should never be a guessing game
            <span className="period">.</span>
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-paper/70">
            See table status across every area of the venue from one live
            operational view. Built for rooms that do not fit on a single sight
            line.
          </p>
        </div>
        <ZoneFloor />
      </div>
    </section>
  );
}

function Outcomes() {
  return (
    <section className="section-anchor py-24 md:py-32">
      <div className="shell grid gap-16 lg:grid-cols-[0.8fr_1.2fr]">
        <div>
          <h2 className="display max-w-[14ch] text-3xl md:text-5xl">
            Why this is worth paying for every month
            <span className="period">.</span>
          </h2>
          <p className="mt-5 max-w-[46ch] text-lg text-muted">
            Less uncertainty, faster decisions, better service flow, a clearer
            guest experience, and a more productive floor.
          </p>
          <div className="mt-8">
            <BookDemoButton source="after-outcomes" />
          </div>
        </div>
        <div className="grid gap-8">
          {outcomes.map((item) => (
            <article className="border-t border-line pt-6" key={item.title}>
              <h3 className="font-sans text-2xl font-semibold tracking-normal">
                {item.title}
              </h3>
              <p className="mt-2 max-w-[54ch] text-muted">{item.body}</p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function Who() {
  return (
    <section className="section-anchor py-24 md:py-32" id="restaurants">
      <div className="shell">
        <h2 className="display max-w-[14ch] text-3xl md:text-5xl">
          Built for floors that get busy
          <span className="period">.</span>
        </h2>
        <p className="mt-5 max-w-[54ch] text-lg text-muted">
          High-volume rooms, multiple service areas, and teams that move
          between sections.
        </p>
        <div className="mt-12 grid gap-4 md:grid-cols-6">
          {venues.map((venue, index) => {
            const span =
              index === 0
                ? "md:col-span-4 md:min-h-[420px]"
                : index === 1
                  ? "md:col-span-2"
                  : "md:col-span-3";
            return (
              <article
                className={`relative min-h-[280px] overflow-hidden ${span}`}
                key={venue.title}
              >
                <Image
                  alt={venue.alt}
                  className="object-cover"
                  fill
                  sizes="(max-width: 768px) 100vw, 50vw"
                  src={venue.image}
                />
                <div className="absolute inset-0 bg-[linear-gradient(180deg,transparent_35%,#1c1410_100%)]" />
                <div className="absolute inset-x-0 bottom-0 p-6 text-paper">
                  <h3 className="display text-2xl tracking-tight">
                    {venue.title}
                  </h3>
                  <p className="mt-2 max-w-[36ch] text-paper/75">{venue.body}</p>
                </div>
              </article>
            );
          })}
        </div>
      </div>
    </section>
  );
}

function Beside() {
  return (
    <section className="section-anchor border-t border-line py-24 md:py-32" id="beside">
      <div className="shell">
        <h2 className="display max-w-[18ch] text-3xl md:text-5xl">
          Your POS runs the transaction. Seatd runs the floor
          <span className="period">.</span>
        </h2>
        <p className="mt-5 max-w-[58ch] text-lg text-muted">
          Seatd is complementary. It does not ask you to throw out reservations
          or payments software.
        </p>
        <div className="mt-12 overflow-x-auto">
          <table className="compare-table min-w-[640px]">
            <thead>
              <tr>
                <th scope="col">Capability</th>
                <th scope="col">Manual floor</th>
                <th scope="col">POS only</th>
                <th className="compare-seatd" scope="col">
                  Seatd
                </th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <th scope="row">Live visual floor status</th>
                <td>Limited</td>
                <td>Sometimes</td>
                <td className="compare-seatd">Yes</td>
              </tr>
              <tr>
                <th scope="row">Guest table assistance</th>
                <td>No</td>
                <td>Usually no</td>
                <td className="compare-seatd">Yes</td>
              </tr>
              <tr>
                <th scope="row">Public availability view</th>
                <td>No</td>
                <td>No</td>
                <td className="compare-seatd">Yes</td>
              </tr>
              <tr>
                <th scope="row">Multi-area awareness</th>
                <td>Manual</td>
                <td>Varies</td>
                <td className="compare-seatd">Yes</td>
              </tr>
              <tr>
                <th scope="row">No guest app required</th>
                <td>n/a</td>
                <td>n/a</td>
                <td className="compare-seatd">Yes</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

function Pricing() {
  return (
    <section className="section-anchor py-24 md:py-32" id="pricing">
      <div className="shell">
        <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
          Priced per location, sized to your floor
          <span className="period">.</span>
        </h2>
        <p className="mt-5 max-w-[58ch] text-lg text-muted">
          Seatd is a monthly product for a venue, not a per-seat gadget.
          Confirm exact pricing on the demo once we have seen how your floor
          is laid out.
        </p>
        <div className="mt-12 grid gap-4 lg:grid-cols-3">
          {pricingTiers.map((tier) => (
            <article
              className={`pricing-tier ${tier.featured ? "is-featured" : ""}`}
              key={tier.title}
            >
              <h3 className="font-sans text-xl font-semibold">{tier.title}</h3>
              <p className="mt-1 text-sm text-muted">{tier.scope}</p>
              <ul className="mt-6 grid gap-3">
                {tier.includes.map((item) => (
                  <li className="pricing-include" key={item}>
                    <CheckIcon aria-hidden="true" size={16} weight="bold" />
                    {item}
                  </li>
                ))}
              </ul>
              <p className="pricing-note mt-8 text-sm font-semibold">
                Confirm on demo
              </p>
            </article>
          ))}
        </div>
      </div>
    </section>
  );
}

function Faq() {
  return (
    <section className="section-anchor border-t border-line py-24 md:py-32" id="faq">
      <div className="shell grid gap-10 lg:grid-cols-[0.8fr_1.2fr]">
        <h2 className="display text-3xl md:text-5xl">
          Questions owners actually ask
          <span className="period">.</span>
        </h2>
        <FaqList />
      </div>
    </section>
  );
}

function FinalCta() {
  return (
    <section className="section-anchor pb-24 md:pb-32" id="demo">
      <div className="shell grid items-start gap-12 lg:grid-cols-[1fr_1fr]">
        <div>
          <h2 className="display max-w-[16ch] text-3xl md:text-5xl">
            See what Seatd would look like in your restaurant
            <span className="period">.</span>
          </h2>
          <p className="mt-5 max-w-[50ch] text-lg text-muted">
            Tell us about your venue. We’ll show how Seatd fits your floor, then
            walk through the owner, waiter, display, and guest experience.
          </p>
          <ol className="mt-8 grid gap-3 text-muted">
            <li>1. Tell us about your venue</li>
            <li>2. We’ll show how Seatd would fit your floor</li>
            <li>3. See owner, waiter, display, and guest views</li>
            <li>4. Ask about setup and rollout</li>
          </ol>
        </div>
        <Clipboard sheetClassName="chart-sheet--form" stamp="Demo">
          <DemoForm />
        </Clipboard>
      </div>
    </section>
  );
}
