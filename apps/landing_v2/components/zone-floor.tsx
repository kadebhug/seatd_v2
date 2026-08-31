"use client";

import { useState } from "react";
import { FLOOR_LAYOUTS, ZONES, type ZoneId } from "../lib/demo-floor";
import { LiveFloor } from "./live-floor";

export function ZoneFloor() {
  const [zone, setZone] = useState<ZoneId>("dining");
  const current = ZONES.find((item) => item.id === zone) ?? ZONES[0];
  if (!current) {
    return null;
  }

  const layout = FLOOR_LAYOUTS[zone];

  function moveZone(nextIndex: number) {
    const next = ZONES[nextIndex];
    if (!next) {
      return;
    }
    setZone(next.id);
    const button = document.querySelector<HTMLButtonElement>(
      `[data-zone="${next.id}"]`,
    );
    button?.focus();
  }

  return (
    <div className="grid gap-8">
      <div
        aria-label="Service areas"
        className="flex flex-wrap gap-2"
        onKeyDown={(event) => {
          const index = ZONES.findIndex((item) => item.id === zone);
          if (event.key === "ArrowRight") {
            event.preventDefault();
            moveZone((index + 1) % ZONES.length);
          } else if (event.key === "ArrowLeft") {
            event.preventDefault();
            moveZone((index - 1 + ZONES.length) % ZONES.length);
          } else if (event.key === "Home") {
            event.preventDefault();
            moveZone(0);
          } else if (event.key === "End") {
            event.preventDefault();
            moveZone(ZONES.length - 1);
          }
        }}
        role="tablist"
      >
        {ZONES.map((item) => {
          const selected = item.id === zone;
          return (
            <button
              aria-controls="zone-floor-panel"
              aria-selected={selected}
              className={`focus-ring min-h-12 rounded-full px-4 text-sm font-semibold transition-[background-color,color,transform] duration-200 ease-[var(--ease-seatd)] ${
                selected
                  ? "bg-bone text-ink"
                  : "border border-[color-mix(in_srgb,#f3eee4_18%,transparent)] bg-transparent text-bone"
              }`}
              data-zone={item.id}
              key={item.id}
              onClick={() => setZone(item.id)}
              role="tab"
              tabIndex={selected ? 0 : -1}
              type="button"
            >
              {item.name}
            </button>
          );
        })}
      </div>
      <p className="max-w-[48ch] text-bone/70">{current.hint}</p>
      <div id="zone-floor-panel" role="tabpanel">
        <LiveFloor
          animate={zone === "dining"}
          area={current.name}
          compact
          fixtures={layout.fixtures}
          tables={layout.tables}
          venue="Marlowe's"
        />
      </div>
    </div>
  );
}
