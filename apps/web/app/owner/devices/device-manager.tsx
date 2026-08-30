"use client";

import { useMemo, useState } from "react";
import type {
  Device,
  Location,
  PairingCode,
} from "@seatd/typescript-seatd-client";

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
  const locationById = useMemo(
    () => new Map(locations.map((location) => [location.id, location])),
    [locations],
  );

  async function createPairingCode() {
    setMessage(null);
    setPairingCode(null);
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
      setMessage(body.error?.message ?? "Pairing code failed");
      return;
    }
    setPairingCode(body.pairingCode);
  }

  async function revokeDevice(id: string) {
    setMessage(null);
    const response = await fetch(`/api/owner/devices/${id}/revoke`, {
      method: "POST",
    });
    const body = await response.json();
    if (!response.ok) {
      setMessage(body.error?.message ?? "Revoke failed");
      return;
    }
    setDevices((current) =>
      current.map((device) => (device.id === id ? body : device)),
    );
  }

  return (
    <section className="grid">
      <article className="panel">
        <h2>Pair a display</h2>
        <div className="device-action-row">
          <label>
            Location
            <select
              value={locationId}
              onChange={(event) => setLocationId(event.target.value)}
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
              value={deviceType}
              onChange={(event) => setDeviceType(event.target.value)}
            >
              <option value="display">Display</option>
              <option value="host_device">Host device</option>
              <option value="manager_tablet">Manager tablet</option>
              <option value="waiter_mobile">Waiter mobile</option>
            </select>
          </label>
          <button onClick={createPairingCode} disabled={!locationId}>
            Generate code
          </button>
        </div>
        {pairingCode ? (
          <div className="pairing-code">
            <strong>{pairingCode.code}</strong>
            <span>Expires {formatDate(pairingCode.expiresAt)}</span>
          </div>
        ) : null}
        {message ? <p className="toast">{message}</p> : null}
      </article>

      <section className="device-list">
        {devices.map((device) => {
          const health = deviceHealth(device);
          return (
            <article className="panel device-card" key={device.id}>
              <div>
                <span className={`status-dot ${health}`} />
                <h2>{device.name ?? device.deviceType}</h2>
                <p>
                  {locationById.get(device.locationId ?? "")?.name ??
                    "Unassigned"}
                </p>
              </div>
              <dl className="compact-list">
                <dt>State</dt>
                <dd>{device.trustState}</dd>
                <dt>Health</dt>
                <dd>{health}</dd>
                <dt>Version</dt>
                <dd>{device.appVersion}</dd>
                <dt>Last seen</dt>
                <dd>
                  {device.lastSeenAt ? formatDate(device.lastSeenAt) : "Never"}
                </dd>
              </dl>
              <button
                className="danger"
                disabled={device.trustState === "revoked"}
                onClick={() => revokeDevice(device.id)}
              >
                Revoke
              </button>
            </article>
          );
        })}
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

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
