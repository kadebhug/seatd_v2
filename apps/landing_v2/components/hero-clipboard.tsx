"use client";

import { useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { Clipboard } from "./clipboard";
import { LiveFloor } from "./live-floor";
import {
  DEMO_VENUE,
  MAIN_FIXTURES,
  MAIN_FLOOR,
  type DemoTable,
  type Occupancy,
} from "../lib/demo-floor";

/**
 * `noteX`/`noteY` are percentages of the floor canvas. The note is ink on the
 * laminate and sits above every tile, so each step parks it in clear canvas
 * beside the table it describes rather than on top of one.
 */
type ServiceStep = Readonly<{
  at: number;
  activeTableId: string;
  tables: Partial<Record<string, Occupancy>>;
  note: string;
  noteX: number;
  noteY: number;
}>;

// Clear gaps in MAIN_FLOOR: (32, 56) sits between T06 (x8-30, y50-70) and
// T12 (y76); (44, 42) is the service-aisle column, below T04 (y24-42) and above
// T08 (y50-68). The vertical margins are the tight ones — the canvas is only
// 18rem tall on mobile, so a note line eats ~5% of the height.
const serviceSteps: ServiceStep[] = [
  {
    at: 0,
    activeTableId: "T20",
    tables: { T20: "attention" },
    note: "Bill · 00:36",
    noteX: 32,
    noteY: 56,
  },
  {
    at: 2600,
    activeTableId: "T20",
    tables: { T20: "attention" },
    note: "Check on",
    noteX: 32,
    noteY: 56,
  },
  {
    at: 5200,
    activeTableId: "T04",
    tables: { T20: "occupied", T04: "occupied" },
    note: "Seat T04",
    noteX: 44,
    noteY: 42,
  },
  {
    at: 8200,
    activeTableId: "T04",
    tables: { T20: "occupied", T04: "available" },
    note: "T04 reset",
    noteX: 44,
    noteY: 42,
  },
];

const LOOP_MS = 11200;
const initialStep = serviceSteps[0] as ServiceStep;

export function HeroClipboard() {
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
    <div className="timber-field relative flex min-h-[28rem] items-center justify-center overflow-hidden px-5 py-10 md:min-h-[36rem] md:px-10 lg:min-h-[100dvh]">
      <Clipboard className="w-full max-w-[34rem]">
        <div className="relative">
          <div className="mb-3 flex items-start justify-between gap-3">
            <p className="display text-[1.05rem] tracking-[0.08em] text-[#141411] uppercase">
              Marlowe’s floor chart
            </p>
            <ul className="flex flex-wrap justify-end gap-x-3 gap-y-1 text-[0.62rem] font-semibold tracking-wide text-[#141411]/70 uppercase">
              <li className="inline-flex items-center gap-1.5">
                <span className="legend-dot bg-available" />
                Available
              </li>
              <li className="inline-flex items-center gap-1.5">
                <span className="legend-dot bg-signal" />
                Occupied
              </li>
              <li className="inline-flex items-center gap-1.5">
                <span className="legend-dot bg-attention" />
                Attention
              </li>
            </ul>
          </div>
          {/* The note anchors to the floor canvas, not the whole sheet, so its
              coordinates share the tables' percentage space. With chrome and
              key off, LiveFloor's figure box is exactly the canvas box. */}
          <div className="relative">
            <LiveFloor
              activeTableId={step.activeTableId}
              area="Main floor"
              chrome={false}
              density="compact"
              fixtures={MAIN_FIXTURES}
              showKey={false}
              tables={tables}
              venue={DEMO_VENUE}
            />
            <p
              className="chart-note"
              style={{ left: `${step.noteX}%`, top: `${step.noteY}%` }}
            >
              {step.note}
            </p>
          </div>
          <p className="mt-3 text-[0.68rem] text-[#141411]/55">
            Synthetic demo venue. Table states are live on this chart.
          </p>
        </div>
      </Clipboard>
    </div>
  );
}
