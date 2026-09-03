import Image from "next/image";
import {
  ArrowRight,
  Buildings,
  Monitor,
  SignIn,
} from "@phosphor-icons/react/dist/ssr";
import {
  DemoForm,
  LandingInteractionLayer,
  MarketingReveal,
} from "./components/landing-interactions";

export default function Page() {
  return (
    <main className="marketing-page" id="main">
      <LandingInteractionLayer />
      <MarketingHeader />
      <MarketingReveal>
        <HeroSection />
      </MarketingReveal>
      <MarketingReveal>
        <TrustSection />
      </MarketingReveal>
      <MarketingReveal>
        <ProblemSection />
      </MarketingReveal>
      <MarketingReveal>
        <PromiseSection />
      </MarketingReveal>
      <MarketingReveal>
        <PillarsSection />
      </MarketingReveal>
      <MarketingReveal>
        <ContextSection />
      </MarketingReveal>
      <MarketingReveal>
        <FitSection />
      </MarketingReveal>
      <MarketingReveal>
        <FinalCtaSection />
      </MarketingReveal>
    </main>
  );
}

function MarketingHeader() {
  return (
    <header className="marketing-header">
      <a aria-label="Seatd home" className="marketing-logo" href="/">
        <Image
          alt=""
          height={34}
          priority
          src="/brand/seatd-lockup-color.svg"
          width={125}
        />
      </a>
      <nav aria-label="Marketing navigation" className="marketing-nav">
        <a href="#product">Product</a>
        <a href="#how-it-works">How It Works</a>
        <a href="#for-restaurants">For Restaurants</a>
        <a href="#pricing">Pricing</a>
      </nav>
      <div className="marketing-header-actions">
        <a className="marketing-login-link" href="/login?returnTo=/owner">
          <SignIn size={17} weight="regular" />
          <span>Sign in</span>
        </a>
        <a className="marketing-button marketing-button-small" href="#demo">
          <span>Book a Demo</span>
          <span aria-hidden="true" className="marketing-button-icon">
            <ArrowRight size={15} weight="bold" />
          </span>
        </a>
      </div>
    </header>
  );
}

function HeroSection() {
  return (
    <section className="marketing-hero">
      <div className="marketing-hero-copy">
        <p className="marketing-eyebrow">Restaurant floor operations</p>
        <h1>See your floor live.</h1>
        <p>
          Seatd gives your team a live floor view, table QR assistance, and
          calmer service coordination.
        </p>
        <div className="marketing-actions">
          <a className="marketing-button" href="#demo">
            <span>Book a Demo</span>
            <span aria-hidden="true" className="marketing-button-icon">
              <ArrowRight size={17} weight="bold" />
            </span>
          </a>
          <a className="marketing-link" href="#product">
            See How It Works
          </a>
        </div>
      </div>
      <div className="product-stage" aria-label="Seatd live floor example">
        <FloorMapVisual />
        <aside className="request-panel">
          <span className="request-kicker">Guest Assist</span>
          <strong>Table 14</strong>
          <p>Bill please</p>
          <small>Waiting 36s</small>
        </aside>
        <aside className="display-panel">
          <Monitor size={18} weight="regular" />
          <span>Entrance display</span>
          <strong>Patio has seats</strong>
        </aside>
      </div>
    </section>
  );
}

function TrustSection() {
  const trustItems = [
    "No guest app required",
    "Works alongside existing systems",
    "Live floor updates",
    "Table-scoped guest requests",
  ];

  return (
    <section aria-label="Seatd trust signals" className="trust-strip">
      <div className="trust-marks" aria-hidden="true">
        <span>HM</span>
        <span>PB</span>
        <span>OR</span>
        <span>LF</span>
      </div>
      <div className="trust-copy">
        {trustItems.map((item) => (
          <span key={item}>{item}</span>
        ))}
      </div>
    </section>
  );
}

function ProblemSection() {
  const problems = [
    {
      title: "Availability changes",
      body: "A table opens, a group lingers, and the host is already moving.",
    },
    {
      title: "Guests wait to be noticed",
      body: "Busy staff cannot always see the table that needs attention now.",
    },
    {
      title: "Sections lose sync",
      body: "Patio, bar, and upstairs teams work from different mental maps.",
    },
    {
      title: "Managers miss the full picture",
      body: "The floor is visible in person, but harder to read over a shift.",
    },
  ];

  return (
    <section className="problem-section">
      <div className="section-copy section-copy-narrow">
        <h2>Busy service should not run on memory.</h2>
        <p>
          Restaurant teams are already moving fast. Seatd gives the floor a
          shared operating picture before small misses turn into service drag.
        </p>
      </div>
      <div className="problem-grid">
        {problems.map((problem) => (
          <article className="problem-card" key={problem.title}>
            <h3>{problem.title}</h3>
            <p>{problem.body}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

function PromiseSection() {
  const roles = [
    ["Guest", "Scan a table QR and ask for help."],
    ["Staff", "See requests and live table state."],
    ["Owner", "Configure the venue and read service patterns."],
    ["Display", "Show availability where guests enter."],
  ];

  return (
    <section className="promise-section" id="product">
      <div className="section-copy">
        <h2>One shared view of the floor.</h2>
        <p>
          Seatd connects the moments that usually happen apart: walk-ins, seated
          guests, waiters, displays, and owner visibility.
        </p>
        <a className="marketing-link" href="#how-it-works">
          See How It Works
        </a>
      </div>
      <div className="system-board" aria-label="Seatd connected roles">
        {roles.map(([role, description]) => (
          <article className="system-card" key={role}>
            <span>{role}</span>
            <strong>{description}</strong>
          </article>
        ))}
      </div>
    </section>
  );
}

function PillarsSection() {
  const pillars = [
    {
      title: "See",
      heading: "See the whole floor at a glance.",
      body: "Give staff a live visual view of tables, service areas, and attention states.",
      className: "pillar-see",
    },
    {
      title: "Respond",
      heading: "Let guests ask. Let staff respond.",
      body: "Guests request help from the table. Staff see where attention is needed.",
      className: "pillar-respond",
    },
    {
      title: "Improve",
      heading: "Learn how service moves.",
      body: "Owner reporting helps teams understand floor patterns without turning Seatd into a generic BI tool.",
      className: "pillar-improve",
    },
  ];

  return (
    <section className="pillars-section">
      <div className="section-copy section-copy-center">
        <h2>See. Respond. Improve.</h2>
        <p>
          Seatd keeps the product story simple because restaurant service is
          already complex enough.
        </p>
      </div>
      <div className="pillar-grid">
        {pillars.map((pillar) => (
          <article
            className={`pillar-card ${pillar.className}`}
            key={pillar.title}
          >
            <span>{pillar.title}</span>
            <h3>{pillar.heading}</h3>
            <p>{pillar.body}</p>
          </article>
        ))}
      </div>
      <a className="marketing-button marketing-button-center" href="#demo">
        <span>Book a Demo</span>
        <span aria-hidden="true" className="marketing-button-icon">
          <ArrowRight size={17} weight="bold" />
        </span>
      </a>
    </section>
  );
}

function ContextSection() {
  return (
    <section className="context-section" id="how-it-works">
      <div className="section-copy">
        <h2>Fits the way service already moves.</h2>
        <p>
          Seatd does not ask guests to download an app or ask restaurants to
          replace the systems they already rely on.
        </p>
      </div>
      <div className="context-grid">
        <SurfaceCard
          body="Guests can see availability as they enter."
          image="/brand/seatd-symbol-reverse.svg"
          title="Entrance"
        />
        <SurfaceCard
          body="Staff keep the live floor in reach during service."
          image="/brand/seatd-symbol-color.svg"
          title="During service"
        />
        <SurfaceCard
          body="A table QR opens simple assistance in the browser."
          image="/brand/seatd-symbol-color.svg"
          title="At the table"
        />
      </div>
    </section>
  );
}

function FitSection() {
  const venues = [
    ["Restaurants", "High-volume casual dining with active table turns."],
    [
      "Pubs and sports bars",
      "Rooms where guests, screens, and sections shift quickly.",
    ],
    ["Multi-area venues", "Patios, decks, lounges, bars, and upstairs floors."],
    ["Restaurant groups", "Operators who need a repeatable floor model."],
  ];

  const comparisons = [
    ["POS", "Orders, bills, and payments."],
    ["Reservations", "Bookings and future table planning."],
    ["Seatd", "Live floor visibility, guest assistance, and floor operations."],
  ];

  return (
    <section className="fit-section" id="for-restaurants">
      <div className="fit-board">
        <h2>Your POS runs transactions. Seatd runs the floor.</h2>
        <div className="comparison-grid" id="pricing">
          {comparisons.map(([system, purpose]) => (
            <article className="comparison-card" key={system}>
              <span>{system}</span>
              <p>{purpose}</p>
            </article>
          ))}
        </div>
      </div>
      <div className="venue-list">
        {venues.map(([venue, description]) => (
          <article key={venue}>
            <Buildings size={22} weight="regular" />
            <h3>{venue}</h3>
            <p>{description}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

function FinalCtaSection() {
  return (
    <section className="final-section" id="demo">
      <div className="section-copy">
        <h2>See what Seatd would look like in your restaurant.</h2>
        <p>
          We&apos;ll walk through your floor, staff flow, guest assist, and
          rollout questions.
        </p>
        <div className="demo-steps" aria-label="Demo walkthrough topics">
          <span>Map your floor</span>
          <span>Show each role</span>
          <span>Plan rollout</span>
        </div>
      </div>
      <DemoForm />
    </section>
  );
}

function SurfaceCard({
  body,
  image,
  title,
}: Readonly<{ body: string; image: string; title: string }>) {
  return (
    <article className="surface-card">
      <div className="surface-media">
        <Image alt="" height={96} src={image} width={96} />
      </div>
      <h3>{title}</h3>
      <p>{body}</p>
    </article>
  );
}

function FloorMapVisual() {
  const tables = [
    ["T01", "Available", "available", "9%", "18%"],
    ["T02", "Occupied", "occupied", "26%", "14%"],
    ["T03", "Available", "available", "45%", "22%"],
    ["T04", "Attention", "attention", "64%", "17%"],
    ["T08", "Occupied", "occupied", "17%", "47%"],
    ["T11", "Available", "available", "42%", "52%"],
    ["T14", "Attention", "attention", "66%", "55%"],
    ["B02", "Occupied", "occupied", "76%", "39%"],
  ];

  return (
    <div className="floor-visual">
      <div className="floor-visual-header">
        <div>
          <strong>Demo restaurant</strong>
          <span>Main Floor, Patio, Bar</span>
        </div>
        <span className="live-chip">Live floor</span>
      </div>
      <div className="floor-plan">
        {tables.map(([label, state, className, left, top]) => (
          <div
            className={`floor-table ${className}`}
            key={label}
            style={{ left, top }}
          >
            <strong>{label}</strong>
            <span>{state}</span>
          </div>
        ))}
        <div className="zone-label zone-main">Main Floor</div>
        <div className="zone-label zone-patio">Patio</div>
        <div className="zone-label zone-bar">Bar</div>
      </div>
      <div className="floor-legend" aria-label="Table states">
        <span>Available</span>
        <span>Occupied</span>
        <span>Attention</span>
      </div>
    </div>
  );
}
