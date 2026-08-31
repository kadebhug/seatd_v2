import { ArrowRightIcon, CheckIcon } from "@phosphor-icons/react/dist/ssr";
import Image from "next/image";
import { BookDemoButton } from "../components/book-demo-button";
import { DemoForm } from "../components/demo-form";
import { EntranceBoard } from "../components/entrance-board";
import { FaqList } from "../components/faq-list";
import { GuestAssistStage } from "../components/guest-assist-stage";
import { HeroServiceMoment } from "../components/hero-service-moment";
import { LiveFloor } from "../components/live-floor";
import { Reveal } from "../components/reveal";
import { SiteFooter } from "../components/site-footer";
import { SiteHeader } from "../components/site-header";
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

export default function HomePage() {
  return (
    <>
      <div className="grain" aria-hidden="true" />
      <SiteHeader />
      <main id="main">
        <Hero />
        <Trust />
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
    <section className="shell grid min-h-[100dvh] items-end gap-8 pb-10 pt-8 md:pt-10 lg:grid-cols-[minmax(0,0.85fr)_minmax(0,1.25fr)] lg:items-center lg:gap-12">
      <div className="max-w-xl">
        <p className="text-sm font-medium text-muted">
          Restaurant floor operations
        </p>
        <h1 className="mt-3 max-w-[16ch] text-4xl leading-[1.08] font-semibold tracking-tight md:text-5xl lg:text-6xl">
          Know what’s happening on your floor.
        </h1>
        <p className="mt-5 max-w-[36ch] text-lg text-muted md:text-xl">
          Seatd shows table status live, lets guests ask from their table, and
          keeps service coordinated.
        </p>
        <div className="mt-7 flex flex-wrap items-center gap-5">
          <BookDemoButton source="hero" />
          <a
            className="cta-secondary inline-flex"
            data-cta="see-how"
            href="/#how-it-works"
          >
            See how it works
            <ArrowRightIcon aria-hidden="true" size={16} weight="bold" />
          </a>
        </div>
      </div>
      <HeroServiceMoment />
    </section>
  );
}

function Trust() {
  return (
    <section className="border-y border-line py-8">
      <ul className="shell flex flex-wrap gap-x-8 gap-y-3 text-sm font-semibold text-muted">
        {trust.map((item) => (
          <li className="max-w-[28ch]" key={item}>
            {item}
          </li>
        ))}
      </ul>
    </section>
  );
}

function Problem() {
  return (
    <section className="section-anchor py-24 md:py-32">
      <div className="shell grid items-center gap-10 lg:grid-cols-[1.15fr_0.85fr]">
        <div className="relative min-h-[320px] overflow-hidden rounded-[24px] lg:min-h-[560px]">
          <Image
            alt="Service aisle between restaurant tables during a busy period"
            className="object-cover"
            fill
            sizes="(max-width: 1024px) 100vw, 55vw"
            src="/images/seatd-service-aisle.webp"
          />
        </div>
        <Reveal>
          <h2 className="max-w-[16ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Busy service shouldn’t depend on everyone remembering everything.
          </h2>
          <ul className="mt-10 grid gap-6">
            {problems.map((item) => (
              <li key={item.title}>
                <h3 className="text-lg font-semibold">{item.title}</h3>
                <p className="mt-1 max-w-[48ch] text-muted">{item.body}</p>
              </li>
            ))}
          </ul>
        </Reveal>
      </div>
    </section>
  );
}

function Promise() {
  return (
    <section
      className="ink-stage py-24 text-[var(--seatd-ink-stage-fg)] md:py-32"
      style={{ background: "var(--seatd-ink-stage)" }}
    >
      <div className="shell">
        <Reveal>
          <h2 className="max-w-[16ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            One shared view of your restaurant floor.
          </h2>
          <p className="mt-5 max-w-[54ch] text-[var(--seatd-ink-stage-muted)]">
            Seatd connects the people who need to know what is happening, without
            asking guests to join another app.
          </p>
        </Reveal>
        <div className="mt-14 grid gap-px overflow-hidden rounded-[24px] bg-[var(--seatd-ink-stage-line)] md:grid-cols-4">
          {roles.map((role) => (
            <article
              className="bg-[var(--seatd-ink-stage)] p-6 md:p-8"
              key={role.who}
            >
              <h3 className="text-xl font-semibold">{role.who}</h3>
              <p className="mt-2 text-[var(--seatd-ink-stage-muted)]">
                {role.need}
              </p>
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
      <div className="shell grid gap-16 md:gap-24">
        <Reveal>
          <h2 className="max-w-[14ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            See the whole floor at a glance.
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            Give staff a live visual view of which tables are available, occupied,
            or need attention across every service area. Less walking around just
            to find out what is happening.
          </p>
        </Reveal>
        <LiveFloor
          animate
          area="Main floor"
          fixtures={MAIN_FIXTURES}
          tables={MAIN_FLOOR}
          venue={DEMO_VENUE}
        />
        <div className="flex justify-start">
          <BookDemoButton source="after-product" />
        </div>
      </div>
    </section>
  );
}

function Scenarios() {
  return (
    <section className="pb-8">
      <div className="shell grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <article className="relative min-h-[420px] overflow-hidden rounded-[24px]">
          <Image
            alt="Restaurant entrance looking through to the dining room"
            className="object-cover"
            fill
            sizes="(max-width: 1024px) 100vw, 60vw"
            src="/images/seatd-entrance.webp"
          />
          <div className="absolute inset-0 bg-[linear-gradient(180deg,transparent_30%,#141411_100%)]" />
          <div className="absolute inset-x-0 bottom-0 grid gap-5 p-6 text-bone md:p-8">
            <div>
              <h3 className="text-2xl font-semibold">A guest walks in</h3>
              <p className="mt-2 max-w-[42ch] text-bone/75">
                The entrance display shows where seating is available, including
                patio and bar, without sending the host on a tour.
              </p>
            </div>
            <EntranceBoard />
          </div>
        </article>
        <div className="grid gap-6">
          <article className="rounded-[24px] border border-line bg-surface p-6 md:p-8">
            <h3 className="text-2xl font-semibold">During service</h3>
            <p className="mt-2 max-w-[42ch] text-muted">
              The waiter sees which tables are occupied and where attention is
              needed, from a handheld live floor.
            </p>
          </article>
          <article className="relative min-h-[240px] overflow-hidden rounded-[24px]">
            <Image
              alt="Covered patio seating at a restaurant"
              className="object-cover"
              fill
              sizes="(max-width: 1024px) 100vw, 40vw"
              src="/images/seatd-patio.webp"
            />
            <div className="absolute inset-0 bg-[linear-gradient(180deg,transparent_20%,#141411_100%)]" />
            <div className="absolute inset-x-0 bottom-0 p-6 text-bone">
              <h3 className="text-2xl font-semibold">At the table</h3>
              <p className="mt-2 max-w-[36ch] text-bone/75">
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
        <Reveal>
          <h2 className="max-w-[16ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Seatd works alongside the systems you already use.
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            You do not replace the POS to see the floor. Map the room, give the
            team access, put QR codes on tables, and run service from one view.
          </p>
        </Reveal>
        <ol className="mt-14 grid gap-10 md:grid-cols-2 xl:grid-cols-4">
          {steps.map((step) => (
            <li key={step.title}>
              <h3 className="text-xl font-semibold">{step.title}</h3>
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
    <section className="section-anchor border-t border-line py-24 md:py-32">
      <div className="shell grid gap-12">
        <Reveal>
          <h2 className="max-w-[16ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            No app. No account. Just scan and ask.
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            Guests request help in seconds from the table QR. Staff see where
            attention is needed. Guests do not download an app, create an
            account, enter an email, or join a loyalty programme just to ask for
            assistance.
          </p>
        </Reveal>
        <GuestAssistStage />
      </div>
    </section>
  );
}

function LiveOps() {
  return (
    <section
      className="ink-stage section-anchor py-24 text-[var(--seatd-ink-stage-fg)] md:py-32"
      style={{ background: "var(--seatd-ink-stage)" }}
    >
      <div className="shell grid gap-12 md:gap-16">
        <Reveal>
          <h2 className="max-w-[16ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Your floor should never be a guessing game.
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-[var(--seatd-ink-stage-muted)]">
            See table status across every area of the venue from one live
            operational view. Built for rooms that do not fit on a single sight
            line.
          </p>
        </Reveal>
        <ZoneFloor />
      </div>
    </section>
  );
}

function Outcomes() {
  return (
    <section className="section-anchor py-24 md:py-32">
      <div className="shell grid gap-16 lg:grid-cols-[0.8fr_1.2fr]">
        <Reveal>
          <h2 className="max-w-[14ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Why this is worth paying for every month.
          </h2>
          <p className="mt-5 max-w-[46ch] text-lg text-muted">
            Less uncertainty, faster decisions, better service flow, a clearer
            guest experience, and a more productive floor.
          </p>
          <div className="mt-8">
            <BookDemoButton source="after-outcomes" />
          </div>
        </Reveal>
        <div className="grid gap-8">
          {outcomes.map((item) => (
            <article className="border-t border-line pt-6" key={item.title}>
              <h3 className="text-2xl font-semibold">{item.title}</h3>
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
        <Reveal>
          <h2 className="max-w-[14ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Built for floors that get busy.
          </h2>
          <p className="mt-5 max-w-[54ch] text-lg text-muted">
            High-volume rooms, multiple service areas, and teams that move
            between sections.
          </p>
        </Reveal>
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
              className={`relative min-h-[280px] overflow-hidden rounded-[24px] ${span}`}
              key={venue.title}
            >
              <Image
                alt={venue.alt}
                className="object-cover"
                fill
                sizes="(max-width: 768px) 100vw, 50vw"
                src={venue.image}
              />
              <div className="absolute inset-0 bg-[linear-gradient(180deg,transparent_35%,#141411_100%)]" />
              <div className="absolute inset-x-0 bottom-0 p-6 text-bone">
                <h3 className="text-2xl font-semibold">{venue.title}</h3>
                <p className="mt-2 max-w-[36ch] text-bone/75">{venue.body}</p>
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
    <section className="section-anchor border-t border-line py-24 md:py-32">
      <div className="shell">
        <Reveal>
          <h2 className="max-w-[18ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Your POS runs the transaction. Seatd runs the floor.
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            Seatd is complementary. It does not ask you to throw out reservations
            or payments software.
          </p>
        </Reveal>
        <div className="mt-12 grid gap-4 lg:grid-cols-[0.9fr_0.9fr_1.2fr]">
          <CompareCard
            title="POS"
            body="Orders, bills, and payments."
            points={["Transactions", "Kitchen tickets", "End-of-night reporting"]}
          />
          <CompareCard
            title="Reservations"
            body="Booking future tables."
            points={["Tonight’s book", "Arrival lists", "Waitlist for later"]}
          />
          <CompareCard
            featured
            title="Seatd"
            body="Live floor visibility, guest assistance, and floor operations."
            points={[
              "Live visual floor status",
              "Guest table assistance",
              "Public availability view",
              "Multi-area awareness",
              "No guest app required",
            ]}
          />
        </div>
      </div>
    </section>
  );
}

function CompareCard({
  title,
  body,
  points,
  featured = false,
}: Readonly<{
  title: string;
  body: string;
  points: string[];
  featured?: boolean;
}>) {
  return (
    <article
      className={`rounded-[24px] p-7 ${
        featured
          ? "bg-ink text-bone"
          : "border border-line bg-surface"
      }`}
    >
      <h3 className="text-2xl font-semibold" translate={title === "Seatd" ? "no" : undefined}>
        {title}
      </h3>
      <p className={`mt-2 ${featured ? "text-bone/70" : "text-muted"}`}>{body}</p>
      <ul className="mt-6 grid gap-3">
        {points.map((point) => (
          <li className="flex items-start gap-2" key={point}>
            {featured ? (
              <CheckIcon aria-hidden="true" className="mt-0.5" size={18} weight="bold" />
            ) : null}
            <span>{point}</span>
          </li>
        ))}
      </ul>
    </article>
  );
}

function Pricing() {
  return (
    <section className="section-anchor py-24 md:py-32" id="pricing">
      <div className="shell max-w-3xl">
        <Reveal>
          <h2 className="text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            Priced per location, sized to your floor.
          </h2>
          <p className="mt-5 max-w-[58ch] text-lg text-muted">
            Seatd is a monthly product for a venue, not a per-seat gadget. We’ll
            confirm numbers on the demo once we have seen how your floor is laid
            out.
          </p>
        </Reveal>
      </div>
    </section>
  );
}

function Faq() {
  return (
    <section className="section-anchor border-t border-line py-24 md:py-32" id="faq">
      <div className="shell grid gap-10 lg:grid-cols-[0.8fr_1.2fr]">
        <h2 className="text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
          Questions owners actually ask.
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
          <h2 className="max-w-[16ch] text-3xl leading-[1.12] font-semibold tracking-tight md:text-5xl">
            See what Seatd would look like in your restaurant.
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
        <DemoForm />
      </div>
    </section>
  );
}
