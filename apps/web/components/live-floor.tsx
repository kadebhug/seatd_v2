"use client";

import { useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import {
  countByStatus,
  occupancyLabel,
  type DemoTable,
  type FloorFixture,
} from "../lib/demo-floor";

type FloorDensity = "hero" | "room" | "compact";

type LiveFloorProps = Readonly<{
  tables: DemoTable[];
  venue: string;
  area: string;
  fixtures?: FloorFixture[];
  animate?: boolean;
  activeTableId?: string;
  compact?: boolean;
  density?: FloorDensity;
  showKey?: boolean;
  chrome?: boolean;
  className?: string;
}>;

const canvasHeight: Record<FloorDensity, string> = {
  hero: "min-h-[300px] md:min-h-[420px] lg:min-h-[520px]",
  room: "min-h-[420px] md:min-h-[560px]",
  compact: "min-h-[320px] md:min-h-[420px]",
};

export function LiveFloor({
  tables,
  venue,
  area,
  fixtures = [],
  animate = false,
  activeTableId,
  compact = false,
  density,
  showKey = true,
  chrome = true,
  className = "",
}: LiveFloorProps) {
  const scale = density ?? (compact ? "compact" : "room");
  const reduce = useReducedMotion();
  const [live, setLive] = useState(tables);
  const counts = useMemo(() => countByStatus(live), [live]);

  useEffect(() => {
    setLive(tables);
  }, [tables]);

  useEffect(() => {
    if (!animate || reduce || activeTableId) {
      return;
    }
    const heroScript = [
      { at: 1800, id: "T20", status: "attention" },
      { at: 5200, id: "T20", status: "occupied" },
      { at: 7600, id: "T04", status: "occupied" },
      { at: 10200, id: "T04", status: "available" },
    ] as const;
    const timers: number[] = [];
    function play() {
      setLive(tables);
      for (const step of heroScript) {
        timers.push(
          window.setTimeout(() => {
            setLive((current) =>
              current.map((table) =>
                table.id === step.id
                  ? { ...table, status: step.status }
                  : table,
              ),
            );
          }, step.at),
        );
      }
    }
    play();
    const loop = window.setInterval(play, 13000);
    return () => {
      window.clearInterval(loop);
      for (const timer of timers) {
        window.clearTimeout(timer);
      }
    };
  }, [animate, reduce, tables, activeTableId]);

  return (
    <figure className={`grid gap-4 ${className}`}>
      {chrome ? (
        <div className="flex flex-wrap items-baseline justify-between gap-x-6 gap-y-2">
          <p className="text-sm font-semibold">
            <span translate="no">{venue}</span>
            <span className="mx-2 opacity-40">/</span>
            {area}
          </p>
          <p className="font-mono text-sm font-medium tabular-nums opacity-70">
            {counts.available} available, {counts.occupied} occupied,{" "}
            {counts.attention} attention
          </p>
        </div>
      ) : null}
      <div
        aria-label={`${venue} ${area} floor plan`}
        className={`floor-canvas ${canvasHeight[scale]}`}
        role="img"
      >
        {fixtures.map((fixture) => (
          <div
            aria-hidden={fixture.kind === "aisle" ? true : undefined}
            aria-label={
              fixture.kind === "aisle" ? undefined : fixture.label
            }
            className={`floor-fixture ${fixture.kind}`}
            key={fixture.id}
            style={{
              left: `${fixture.x}%`,
              top: `${fixture.y}%`,
              width: `${fixture.w}%`,
              height: `${fixture.h}%`,
            }}
          >
            {fixture.kind === "aisle" ? null : <span>{fixture.label}</span>}
          </div>
        ))}
        {live.map((table) => (
          <div
            aria-label={`${table.label}, ${occupancyLabel[table.status]}, seats ${table.capacity}`}
            className={`table-tile ${table.shape} ${table.status} ${table.id === activeTableId ? "is-active" : ""}`}
            key={table.id}
            style={
              table.shape === "circle"
                ? {
                    left: `${table.x}%`,
                    top: `${table.y}%`,
                    height: `${table.h}%`,
                  }
                : {
                    left: `${table.x}%`,
                    top: `${table.y}%`,
                    width: `${table.w}%`,
                    height: `${table.h}%`,
                  }
            }
          >
            <strong>{table.label}</strong>
            {/* Circles carry the number only — a round 2-top has no room for a
                status word, and the fill colour plus the chart legend already
                say it. Full status stays in the aria-label above. */}
            {table.shape === "circle" ? null : (
              <span>{occupancyLabel[table.status]}</span>
            )}
          </div>
        ))}
      </div>
      {showKey ? (
        <figcaption className="flex flex-wrap gap-5 text-sm opacity-80">
          <StatusKey color="var(--color-available)" label="Available" />
          <StatusKey color="var(--color-signal)" label="Occupied" />
          <StatusKey color="var(--color-attention)" label="Attention" />
        </figcaption>
      ) : null}
    </figure>
  );
}

function StatusKey({
  color,
  label,
}: Readonly<{ color: string; label: string }>) {
  return (
    <span className="inline-flex items-center gap-2">
      <span
        aria-hidden="true"
        className="h-2.5 w-2.5 rounded-full"
        style={{ background: color }}
      />
      {label}
    </span>
  );
}
