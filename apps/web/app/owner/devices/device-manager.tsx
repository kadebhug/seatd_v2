"use client";

import { useMemo, useState } from "react";
import type {
  Device,
  Location,
  PairingCode,
} from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../../components/status-badge";

type Props = {
  initialDevices: Device[];
  locations: Location[];
};

export function DeviceManager({ initialDevices, locations }: Props) {
  const [devices, setDevices] = useState(initialDevices);
  const [locationId, setLocationId] = useState(locations[0]?.id ?? "");
  const [deviceType, setDeviceType] = useState("display");
  const [pairingCode, setPairingCode] = useState<PairingCode | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const locationById = useMemo(
    () => new Map(locations.map((location) => [location.id, location])),
    [locations],
  );

  async function createPairingCode() {
    setBusy(true);
    setMessage(null);
    setPairingCode(null);
    try {
      const response = await fetch("/api/owner/devices/pairing-codes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          locationId,
          deviceType,
          ttlSeconds: 600,
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        setMessage(body.error?.message ?? "Pairing code failed.");
        return;
      }
      setPairingCode(body.pairingCode);
    } finally {
      setBusy(false);
    }
  }

  async function revokeDevice(id: string, name: string) {
    if (
      !window.confirm(
        `Revoke "${name}"? The device will lose access until paired again.`,
      )
    ) {
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      const response = await fetch(`/api/owner/devices/${id}/revoke`, {
        method: "POST",
      });
      const body = await response.json();
      if (!response.ok) {
        setMessage(body.error?.message ?? "Revoke failed.");
        return;
      }
      setDevices((current) =>
        current.map((device) => (device.id === id ? body : device)),
      );
      setMessage("Device revoked.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="grid">
      <article className="panel">
        <h2>Pair a Display</h2>
        <div className="device-action-row">
          <label>
            Location
            <select
              autoComplete="off"
              name="pairLocationId"
              onChange={(event) => setLocationId(event.target.value)}
              value={locationId}
            >
              {locations.map((location) => (
                <option key={location.id} value={location.id}>
                  {location.name}
                </option>
              ))}
            </select>
          </label>
          <label>
            Type
            <select
              autoComplete="off"
              name="deviceType"
              onChange={(event) => setDeviceType(event.target.value)}
              value={deviceType}
            >
              <option value="display">Display</option>
              <option value="host_device">Host device</option>
              <option value="manager_tablet">Manager tablet</option>
              <option value="waiter_mobile">Waiter mobile</option>
            </select>
          </label>
          <button
            disabled={!locationId || busy}
            onClick={() => void createPairingCode()}
            type="button"
          >
            {busy ? "Generating…" : "Generate Code"}
          </button>
        </div>
        {pairingCode ? (
          <div className="pairing-code">
            <strong>{pairingCode.code}</strong>
            <span>Expires {formatDate(pairingCode.expiresAt)}</span>
          </div>
        ) : null}
        {message ? (
          <p aria-live="polite" className="toast" role="status">
            {message}
          </p>
        ) : null}
      </article>

      <section className="device-list">
        {devices.length > 0 ? (
          devices.map((device) => {
            const health = deviceHealth(device);
            const displayName = device.name ?? device.deviceType;
            return (
              <article className="panel device-card" key={device.id}>
                <div>
                  <StatusBadge
                    label={deviceHealthLabel(health)}
                    variant={health}
                  />
                  <h2>{displayName}</h2>
                  <p>
                    {locationById.get(device.locationId ?? "")?.name ??
                      "Unassigned"}
                  </p>
                </div>
                <dl className="compact-list">
                  <dt>State</dt>
                  <dd>{device.trustState}</dd>
                  <dt>Health</dt>
                  <dd>{deviceHealthLabel(health)}</dd>
                  <dt>Version</dt>
                  <dd className="font-mono tabular-nums">
                    {device.appVersion}
                  </dd>
                  <dt>Last seen</dt>
                  <dd>
                    {device.lastSeenAt
                      ? formatDate(device.lastSeenAt)
                      : "Never"}
                  </dd>
                </dl>
                <button
                  className="danger"
                  disabled={device.trustState === "revoked" || busy}
                  onClick={() => void revokeDevice(device.id, displayName)}
                  type="button"
                >
                  Revoke
                </button>
              </article>
            );
          })
        ) : (
          <article className="panel">
            <p className="empty-state">
              No devices paired yet. Generate a code to connect your first
              display.
            </p>
          </article>
        )}
      </section>
    </section>
  );
}

function deviceHealth(device: Device) {
  if (device.trustState === "revoked") {
    return "revoked";
  }
  if (!device.lastHeartbeatAt) {
    return "offline";
  }
  const last = Date.parse(device.lastHeartbeatAt);
  const threshold = Math.max(device.heartbeatIntervalSeconds * 3, 180) * 1000;
  return Date.now() - last <= threshold ? "online" : "offline";
}

function deviceHealthLabel(health: string) {
  switch (health) {
    case "online":
      return "Online";
    case "offline":
      return "Offline";
    case "revoked":
      return "Revoked";
    default:
      return health;
  }
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
