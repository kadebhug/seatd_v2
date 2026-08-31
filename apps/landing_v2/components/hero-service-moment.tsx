"use client";

import { useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { LiveFloor } from "./live-floor";
import {
  DEMO_VENUE,
  MAIN_FIXTURES,
  MAIN_FLOOR,
  type DemoTable,
  type Occupancy,
} from "../lib/demo-floor";

type ServiceStep = Readonly<{
  at: number;
  activeTableId: string;
  tables: Partial<Record<string, Occupancy>>;
  guest: string;
  waiter: string;
  display: string;
}>;

const serviceSteps: ServiceStep[] = [
  {
    at: 0,
    activeTableId: "T20",
    tables: { T20: "attention" },
    guest: "Request bill",
    waiter: "New request",
    display: "3 tables free",
  },
  {
    at: 2600,
    activeTableId: "T20",
    tables: { T20: "attention" },
    guest: "Bill requested",
    waiter: "Seen 00:08",
    display: "3 tables free",
  },
  {
    at: 5200,
    activeTableId: "T04",
    tables: { T20: "occupied", T04: "occupied" },
    guest: "Table 20 settled",
    waiter: "Host seats T04",
    display: "2 tables free",
  },
  {
    at: 8200,
    activeTableId: "T04",
    tables: { T20: "occupied", T04: "available" },
    guest: "Next party ready",
    waiter: "T04 reset",
    display: "3 tables free",
  },
];

const LOOP_MS = 11200;
const initialStep = serviceSteps[0] as ServiceStep;

export function HeroServiceMoment() {
  const reduce = useReducedMotion();
  const [stepIndex, setStepIndex] = useState(0);
  const step = serviceSteps[stepIndex] ?? initialStep;

  useEffect(() => {
    if (reduce) {
      setStepIndex(0);
      return;
    }

    const timers: number[] = [];
    function play() {
      setStepIndex(0);
      for (let index = 1; index < serviceSteps.length; index += 1) {
        const nextStep = serviceSteps[index];
        if (!nextStep) {
          continue;
        }
        timers.push(window.setTimeout(() => setStepIndex(index), nextStep.at));
      }
    }

    play();
    const loop = window.setInterval(play, LOOP_MS);

    return () => {
      window.clearInterval(loop);
      for (const timer of timers) {
        window.clearTimeout(timer);
      }
    };
  }, [reduce]);

  const tables = useMemo(
    () =>
      MAIN_FLOOR.map((table): DemoTable => {
        const status = step.tables[table.id];
        return status ? { ...table, status } : table;
      }),
    [step],
  );

  return (
    <div className="hero-service-grid grid min-h-0 gap-4 lg:grid-cols-[minmax(0,1fr)_11rem] lg:gap-5">
      <LiveFloor
        activeTableId={step.activeTableId}
        area="Main floor"
        className="min-h-0"
        density="hero"
        fixtures={MAIN_FIXTURES}
        showKey={false}
        tables={tables}
        venue={DEMO_VENUE}
      />
      <aside
        aria-label="Live service sequence"
        className="hero-callouts hidden flex-col justify-center gap-4 lg:flex"
      >
        <ServiceCallout
          active={step.activeTableId === "T20"}
          body={step.guest}
          kicker="Guest"
          title="Table 20"
        />
        <ServiceCallout
          active
          body={step.waiter}
          kicker="Waiter"
          title="Floor"
        />
        <ServiceCallout
          active={step.activeTableId === "T04"}
          body={step.display}
          kicker="Display"
          title="Entrance"
        />
      </aside>
    </div>
  );
}

function ServiceCallout({
  kicker,
  title,
  body,
  active,
}: Readonly<{
  kicker: string;
  title: string;
  body: string;
  active?: boolean;
}>) {
  return (
    <div className={`service-callout ${active ? "is-active" : ""}`}>
      <p className="text-[0.7rem] font-semibold opacity-70">{kicker}</p>
      <p className="mt-1 font-semibold">{title}</p>
      <p className="text-sm opacity-70">{body}</p>
    </div>
  );
}
