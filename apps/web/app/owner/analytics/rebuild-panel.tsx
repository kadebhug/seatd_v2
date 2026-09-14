"use client";

import { useState } from "react";
import type {
  AnalyticsCheckpoint,
  AnalyticsRebuildResponse,
  AnalyticsRebuildRun,
} from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../../components/status-badge";

type Props = {
  initialCheckpoint?: AnalyticsCheckpoint;
  initialLastRun?: AnalyticsRebuildRun;
  locationId: string;
  from: string;
  to: string;
};

const STALE_LAG_SECONDS = 300;

export function RebuildPanel({
  initialCheckpoint,
  initialLastRun,
  locationId,
  from,
  to,
}: Props) {
  const [checkpoint, setCheckpoint] = useState(initialCheckpoint);
  const [lastRun, setLastRun] = useState(initialLastRun);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function rebuild() {
    if (
      !window.confirm(
        `Rebuild analytics for ${from} to ${to}? This recomputes all projected metrics for the range.`,
      )
    ) {
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      const result = await fetchJSON<AnalyticsRebuildResponse>(
        "/api/owner/analytics/rebuild",
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ locationId, from, to }),
        },
      );
      setLastRun(result.rebuildRun);
      setCheckpoint((current) =>
        current
          ? { ...current, rebuildStatus: result.rebuildRun.status }
          : current,
      );
      setMessage(
        result.rebuildRun.status === "completed"
          ? "Rebuild completed."
          : `Rebuild failed: ${result.rebuildRun.lastError ?? "unknown error"}`,
      );
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Rebuild failed.");
    } finally {
      setBusy(false);
    }
  }

  const asOf = checkpoint?.cursorOccurredAt ?? checkpoint?.updatedAt;
  const isStale =
    checkpoint != null && checkpoint.lagSeconds > STALE_LAG_SECONDS;

  return (
    <article className="panel">
      <h2>Data freshness</h2>
      <dl className="compact-list analytics-list">
        <dt>Status</dt>
        <dd>
          <StatusBadge variant={isStale ? "stale" : "available"} />
        </dd>
        <dt>Data as of</dt>
        <dd>{asOf ? formatDate(asOf) : "Never synced"}</dd>
        <dt>Last rebuild</dt>
        <dd>
          {lastRun ? (
            <>
              <StatusBadge variant={lastRun.status} />{" "}
              {formatDate(lastRun.completedAt ?? lastRun.startedAt)}
            </>
          ) : (
            "Never"
          )}
        </dd>
        {lastRun?.status === "failed" && lastRun.lastError ? (
          <>
            <dt>Last error</dt>
            <dd>{lastRun.lastError}</dd>
          </>
        ) : null}
      </dl>
      <button disabled={busy} onClick={() => void rebuild()} type="button">
        {busy ? "Rebuilding…" : "Rebuild analytics"}
      </button>
      {message ? (
        <p
          aria-live="polite"
          className={
            message.startsWith("Rebuild failed") ? "toast error" : "toast"
          }
          role="status"
        >
          {message}
        </p>
      ) : null}
    </article>
  );
}

async function fetchJSON<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, init);
  const body = await response.json();
  if (!response.ok) {
    throw new Error(body.error?.message ?? "Request failed.");
  }
  return body as T;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
