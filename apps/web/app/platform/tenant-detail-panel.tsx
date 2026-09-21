"use client";

import { useState } from "react";
import type { PlatformTenantDetail } from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../components/status-badge";
import { TenantActions } from "./tenant-actions";
import { OwnerActions } from "./owner-actions";

type Props = {
  initialDetail: PlatformTenantDetail;
  canWrite?: boolean;
};

export function TenantDetailPanel({
  initialDetail,
  canWrite = false,
}: Readonly<Props>) {
  const [detail, setDetail] = useState(initialDetail);

  return (
    <section className="platform-detail">
      <section className="panel">
        <div className="platform-title-row">
          <div>
            <h2>{detail.tenant.name}</h2>
            <p className="font-mono">{detail.tenant.id}</p>
          </div>
          <div className="platform-title-actions">
            <StatusBadge
              label={detail.tenant.status}
              variant={detail.tenant.status}
            />
            {canWrite ? (
              <TenantActions detail={detail} onUpdated={setDetail} />
            ) : null}
          </div>
        </div>
        <div className="metric-strip">
          <Metric label="Locations" value={detail.tenant.locationCount} />
          <Metric label="Active" value={detail.tenant.activeLocationCount} />
          <Metric label="Devices" value={detail.tenant.deviceCount} />
          <Metric label="Trusted" value={detail.tenant.trustedDeviceCount} />
        </div>
        <dl className="compact-list">
          <dt>Slug</dt>
          <dd className="font-mono">{detail.tenant.slug}</dd>
          <dt>Updated</dt>
          <dd>{formatDate(detail.tenant.updatedAt)}</dd>
          <dt>Last device seen</dt>
          <dd>{formatOptionalDate(detail.tenant.lastDeviceSeenAt)}</dd>
          <dt>Last audit</dt>
          <dd>{formatOptionalDate(detail.tenant.lastAuditAt)}</dd>
        </dl>
      </section>

      <section className="grid two">
        <section className="panel">
          <h2>Diagnostics</h2>
          <dl className="compact-list">
            <dt>Disabled locations</dt>
            <dd>{detail.diagnostics.disabledLocationCount}</dd>
            <dt>Offline devices</dt>
            <dd>
              {detail.diagnostics.neverHeartbeatDeviceCount +
                detail.diagnostics.staleDeviceCount}
            </dd>
            <dt>Pending devices</dt>
            <dd>{detail.diagnostics.pendingDeviceCount}</dd>
            <dt>Revoked devices</dt>
            <dd>{detail.diagnostics.revokedDeviceCount}</dd>
            <dt>Integrations</dt>
            <dd>
              {detail.diagnostics.connectedIntegrationCount} connected,{" "}
              {detail.diagnostics.degradedIntegrationCount} degraded,{" "}
              {detail.diagnostics.disconnectedIntegrationCount} disconnected
            </dd>
            <dt>Analytics lag</dt>
            <dd>{formatSeconds(detail.diagnostics.analyticsLagSeconds)}</dd>
            <dt>Platform audits, 7d</dt>
            <dd>{detail.diagnostics.recentPlatformAuditCount}</dd>
          </dl>
        </section>

        <section className="panel">
          <h2>Role Counts</h2>
          {detail.roleCounts.length > 0 ? (
            <dl className="compact-list">
              {detail.roleCounts.map((item) => (
                <Row
                  key={item.role}
                  label={humanize(item.role)}
                  value={item.memberCount}
                />
              ))}
            </dl>
          ) : (
            <p className="empty-state">No active memberships.</p>
          )}
        </section>
      </section>

      <section className="grid two">
        <section className="panel">
          <h2>Locations</h2>
          <div className="platform-table">
            {detail.locations.map((location) => (
              <div className="platform-table-row" key={location.id}>
                <span>
                  <strong>{location.name}</strong>
                  <small>{location.timezone}</small>
                </span>
                <StatusBadge label={location.status} variant={location.status} />
              </div>
            ))}
          </div>
        </section>

        <section className="panel">
          <h2>Device Counts</h2>
          <div className="platform-table">
            {detail.deviceCounts.length > 0 ? (
              detail.deviceCounts.map((item) => (
                <div
                  className="platform-table-row"
                  key={`${item.trustState}:${item.deviceType}`}
                >
                  <span>
                    <strong>{humanize(item.deviceType)}</strong>
                    <small>{humanize(item.trustState)}</small>
                  </span>
                  <span className="font-mono tabular-nums">
                    {item.deviceCount}
                  </span>
                </div>
              ))
            ) : (
              <p className="empty-state">No devices registered.</p>
            )}
          </div>
        </section>
      </section>

      <OwnerActions canWrite={canWrite} tenantId={detail.tenant.id} />
    </section>
  );
}

function Metric({ label, value }: Readonly<{ label: string; value: number }>) {
  return (
    <div className="metric-card">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function Row({ label, value }: Readonly<{ label: string; value: number }>) {
  return (
    <>
      <dt>{label}</dt>
      <dd>{value}</dd>
    </>
  );
}

function formatOptionalDate(value?: string) {
  return value ? formatDate(value) : "None";
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

function formatSeconds(value?: number) {
  if (value === undefined) {
    return "Unknown";
  }
  if (value < 60) {
    return `${Math.round(value)}s`;
  }
  return `${Math.round(value / 60)}m`;
}

function humanize(value: string) {
  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
