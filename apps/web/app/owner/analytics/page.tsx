import type {
  AnalyticsComparisonResponse,
  AnalyticsSummary,
  AnalyticsTimeseriesResponse,
} from "@seatd/typescript-seatd-client";
import { getSession } from "../../../lib/session";
import { seatdFetch } from "../../../lib/seatd-api";

export const dynamic = "force-dynamic";

export default async function OwnerAnalyticsPage() {
  const session = await getSession();
  const today = new Date().toISOString().slice(0, 10);
  const from = startDate(6);
  const to = nextDate(1);
  const locationQuery = `locationId=${session.locationId}`;
  const [summary, series, zones, servicePeriods] = await Promise.all([
    seatdFetch<AnalyticsSummary>(
      `/v1/analytics/summary?${locationQuery}&date=${today}`,
    ),
    seatdFetch<AnalyticsTimeseriesResponse>(
      `/v1/analytics/timeseries?${locationQuery}&from=${from}&to=${to}&grain=day`,
    ),
    seatdFetch<AnalyticsComparisonResponse>(
      `/v1/analytics/comparison?${locationQuery}&from=${from}&to=${to}&groupBy=zone`,
    ),
    seatdFetch<AnalyticsComparisonResponse>(
      `/v1/analytics/comparison?${locationQuery}&from=${from}&to=${to}&groupBy=service_period`,
    ),
  ]);

  return (
    <main className="page analytics-page" id="main">
      <section className="page-heading">
        <p>Analytics</p>
        <h1>{summary.location.name}</h1>
      </section>

      <section className="metric-strip">
        <MetricCard
          label="Utilisation"
          value={percent(summary.today.metric.utilisationRate)}
        />
        <MetricCard
          label="Occupied hours"
          value={hours(summary.today.metric.occupancySeconds)}
        />
        <MetricCard
          label="Turnover"
          value={summary.today.metric.turnoverRate.toFixed(1)}
        />
        <MetricCard
          label="Assist requests"
          value={String(summary.today.metric.assistRequestCount)}
        />
      </section>

      <section className="grid two">
        <article className="panel">
          <h2>Utilisation over time</h2>
          <div className="bar-list">
            {series.points.length > 0 ? (
              series.points.map((point) => (
                <div className="bar-row" key={point.bucketStart}>
                  <span>{shortDate(point.bucketStart)}</span>
                  <div className="bar-track">
                    <div
                      className="bar-fill"
                      style={{
                        width: `${Math.min(100, point.metric.utilisationRate * 100)}%`,
                      }}
                    />
                  </div>
                  <strong>{percent(point.metric.utilisationRate)}</strong>
                </div>
              ))
            ) : (
              <p className="empty-state">No projected daily metrics yet.</p>
            )}
          </div>
        </article>

        <article className="panel">
          <h2>Sessions and assists</h2>
          <dl className="compact-list analytics-list">
            <dt>Completed sessions</dt>
            <dd>{summary.today.metric.completedSessionCount}</dd>
            <dt>Average session</dt>
            <dd>{minutes(summary.today.metric.avgSessionSeconds)}</dd>
            <dt>P90 session</dt>
            <dd>{minutes(summary.today.metric.p90SessionSeconds)}</dd>
            <dt>Average response</dt>
            <dd>{minutes(summary.today.metric.avgAssistResponseSeconds)}</dd>
            <dt>Average resolution</dt>
            <dd>{minutes(summary.today.metric.avgAssistResolutionSeconds)}</dd>
          </dl>
        </article>
      </section>

      <section className="grid two">
        <ComparisonPanel title="Zones" points={zones.points} />
        <ComparisonPanel
          title="Service periods"
          points={servicePeriods.points}
        />
      </section>

      <section className="grid two">
        <article className="panel">
          <h2>Current service</h2>
          {summary.currentServicePeriod ? (
            <dl className="compact-list analytics-list">
              <dt>Name</dt>
              <dd>{summary.currentServicePeriod.name}</dd>
              <dt>Utilisation</dt>
              <dd>
                {percent(summary.currentServicePeriod.metric.utilisationRate)}
              </dd>
              <dt>Requests</dt>
              <dd>{summary.currentServicePeriod.metric.assistRequestCount}</dd>
            </dl>
          ) : (
            <p className="empty-state">No active service period right now.</p>
          )}
        </article>

        <article className="panel">
          <h2>Data quality</h2>
          <dl className="compact-list analytics-list">
            <dt>Open sessions</dt>
            <dd>{summary.dataQuality.openSessionCount}</dd>
            <dt>Long open sessions</dt>
            <dd>{summary.dataQuality.longOpenSessionCount}</dd>
            <dt>Missing occupancy</dt>
            <dd>{summary.dataQuality.missingOccupancyCount}</dd>
            <dt>Stale devices</dt>
            <dd>{summary.dataQuality.staleDeviceCount}</dd>
            <dt>Projector lag</dt>
            <dd>{seconds(summary.dataQuality.projectorLagSeconds)}</dd>
          </dl>
        </article>
      </section>
    </main>
  );
}

function MetricCard({
  label,
  value,
}: Readonly<{ label: string; value: string }>) {
  return (
    <article className="metric-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </article>
  );
}

function ComparisonPanel({
  title,
  points,
}: Readonly<{
  title: string;
  points: AnalyticsComparisonResponse["points"];
}>) {
  const totals = new Map<
    string,
    { name: string; utilisation: number; requests: number }
  >();
  for (const point of points) {
    const current = totals.get(point.groupId) ?? {
      name: point.groupName,
      utilisation: 0,
      requests: 0,
    };
    current.utilisation = Math.max(
      current.utilisation,
      point.metric.utilisationRate,
    );
    current.requests += point.metric.assistRequestCount;
    totals.set(point.groupId, current);
  }
  const rows = [...totals.entries()];

  return (
    <article className="panel">
      <h2>{title}</h2>
      <div className="comparison-list">
        {rows.length > 0 ? (
          rows.map(([id, row]) => (
            <div className="comparison-row" key={id}>
              <span>{row.name}</span>
              <strong>{percent(row.utilisation)}</strong>
              <small>{row.requests} assists</small>
            </div>
          ))
        ) : (
          <p className="empty-state">No projected comparison metrics yet.</p>
        )}
      </div>
    </article>
  );
}

function startDate(daysAgo: number) {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() - daysAgo);
  return date.toISOString().slice(0, 10);
}

function nextDate(daysAhead: number) {
  const date = new Date();
  date.setUTCDate(date.getUTCDate() + daysAhead);
  return date.toISOString().slice(0, 10);
}

function shortDate(value: string) {
  return new Intl.DateTimeFormat("en", {
    month: "short",
    day: "numeric",
  }).format(new Date(value));
}

function percent(value: number) {
  return `${Math.round(value * 100)}%`;
}

function hours(value: number) {
  return `${(value / 3600).toFixed(1)}h`;
}

function minutes(value: number) {
  if (value <= 0) {
    return "0m";
  }
  return `${Math.round(value / 60)}m`;
}

function seconds(value: number) {
  return `${Math.round(value)}s`;
}
