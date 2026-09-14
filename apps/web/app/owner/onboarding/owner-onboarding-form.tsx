"use client";

import { useMemo, useState } from "react";

type TableDraft = {
  label: string;
  capacityLabel: string;
};

const weekdays = [
  { label: "Sun", value: 0 },
  { label: "Mon", value: 1 },
  { label: "Tue", value: 2 },
  { label: "Wed", value: 3 },
  { label: "Thu", value: 4 },
  { label: "Fri", value: 5 },
  { label: "Sat", value: 6 },
];

const defaultTables: TableDraft[] = Array.from({ length: 6 }, (_, index) => ({
  label: String(index + 1),
  capacityLabel: index < 2 ? "2" : "4",
}));

export function OwnerOnboardingForm({
  displayName,
}: Readonly<{ displayName: string }>) {
  const [floorName, setFloorName] = useState("Main floor");
  const [floorSlug, setFloorSlug] = useState("main-floor");
  const [zoneName, setZoneName] = useState("Dining room");
  const [tables, setTables] = useState(defaultTables);
  const [serviceName, setServiceName] = useState("Dinner");
  const [startTime, setStartTime] = useState("17:00");
  const [endTime, setEndTime] = useState("22:00");
  const [daysOfWeek, setDaysOfWeek] = useState([1, 2, 3, 4, 5, 6]);
  const [status, setStatus] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [createdLocationId, setCreatedLocationId] = useState("");
  const [deviceType, setDeviceType] = useState("display");
  const [pairingCode, setPairingCode] = useState("");

  const activeTables = useMemo(
    () => tables.filter((table) => table.label.trim() !== ""),
    [tables],
  );

  function updateTable(index: number, patch: Partial<TableDraft>) {
    setTables((current) =>
      current.map((table, candidate) =>
        candidate === index ? { ...table, ...patch } : table,
      ),
    );
  }

  async function submit() {
    setBusy(true);
    setStatus("");
    setError("");
    setPairingCode("");
    try {
      const response = await fetch("/api/owner/setup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          floor: {
            name: floorName,
            slug: floorSlug,
            canvas: { width: 1200, height: 800, unit: "px" },
            zones: [{ name: zoneName, sortOrder: 0 }],
            tables: activeTables.map((table, index) => ({
              label: table.label,
              capacityLabel: table.capacityLabel || "2-4",
              shape: "rectangle",
              zoneName,
              geometry: {
                x: 120 + (index % 3) * 220,
                y: 120 + Math.floor(index / 3) * 170,
                width: 120,
                height: 84,
                rotation: 0,
              },
            })),
          },
          servicePeriods: [
            { name: serviceName, daysOfWeek, startTime, endTime },
          ],
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        setError(body.error?.message ?? "Setup failed.");
        return;
      }
      setCreatedLocationId(body.location.id);
      setStatus("Setup complete.");
    } finally {
      setBusy(false);
    }
  }

  async function generatePairingCode() {
    setBusy(true);
    setError("");
    setPairingCode("");
    try {
      const response = await fetch("/api/owner/devices/pairing-codes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          locationId: createdLocationId,
          deviceType,
          ttlSeconds: 600,
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        setError(body.error?.message ?? "Pairing code failed.");
        return;
      }
      setPairingCode(body.pairingCode.code);
    } finally {
      setBusy(false);
    }
  }

  if (createdLocationId) {
    return (
      <section className="grid two onboarding-grid">
        <article className="panel form">
          <h2>Device Pairing</h2>
          <label>
            Type
            <select
              autoComplete="off"
              onChange={(event) => setDeviceType(event.target.value)}
              value={deviceType}
            >
              <option value="display">Display</option>
              <option value="waiter_mobile">Waiter mobile</option>
              <option value="host_device">Host device</option>
              <option value="manager_tablet">Manager tablet</option>
            </select>
          </label>
          <button disabled={busy} onClick={() => void generatePairingCode()}>
            {busy ? "Generating..." : "Generate Code"}
          </button>
          {pairingCode ? (
            <div className="pairing-code">
              <strong>{pairingCode}</strong>
              <span>Expires in 10 minutes</span>
            </div>
          ) : null}
        </article>
        <article className="panel form">
          <h2>{displayName}</h2>
          <p className="empty-state">{status}</p>
          <a className="button-link" href="/owner">
            Open Owner Workspace
          </a>
        </article>
        {error ? (
          <p aria-live="assertive" className="toast error" role="alert">
            {error}
          </p>
        ) : null}
      </section>
    );
  }

  return (
    <form
      className="onboarding-grid"
      onSubmit={(event) => {
        event.preventDefault();
        void submit();
      }}
    >
      <section className="grid two">
        <article className="panel form">
          <h2>Service Period</h2>
          <label>
            Name
            <input
              onChange={(event) => setServiceName(event.target.value)}
              required
              value={serviceName}
            />
          </label>
          <fieldset className="weekday-picker">
            <legend>Days</legend>
            {weekdays.map((day) => (
              <label key={day.value}>
                <input
                  checked={daysOfWeek.includes(day.value)}
                  name="daysOfWeek"
                  onChange={(event) =>
                    setDaysOfWeek((current) =>
                      event.target.checked
                        ? [...current, day.value].sort()
                        : current.filter((value) => value !== day.value),
                    )
                  }
                  type="checkbox"
                  value={day.value}
                />
                {day.label}
              </label>
            ))}
          </fieldset>
          <label>
            Start
            <input
              onChange={(event) => setStartTime(event.target.value)}
              required
              type="time"
              value={startTime}
            />
          </label>
          <label>
            End
            <input
              onChange={(event) => setEndTime(event.target.value)}
              required
              type="time"
              value={endTime}
            />
          </label>
        </article>

        <article className="panel form">
          <h2>Floor Layout</h2>
          <label>
            Floor
            <input
              onChange={(event) => setFloorName(event.target.value)}
              required
              value={floorName}
            />
          </label>
          <label>
            Floor slug
            <input
              autoComplete="off"
              onChange={(event) => setFloorSlug(event.target.value)}
              required
              spellCheck={false}
              value={floorSlug}
            />
          </label>
          <label>
            Zone
            <input
              onChange={(event) => setZoneName(event.target.value)}
              required
              value={zoneName}
            />
          </label>
          <div className="table-draft-grid">
            {tables.map((table, index) => (
              <div className="table-draft-row" key={index}>
                <label>
                  Table
                  <input
                    onChange={(event) =>
                      updateTable(index, { label: event.target.value })
                    }
                    value={table.label}
                  />
                </label>
                <label>
                  Seats
                  <input
                    onChange={(event) =>
                      updateTable(index, {
                        capacityLabel: event.target.value,
                      })
                    }
                    value={table.capacityLabel}
                  />
                </label>
              </div>
            ))}
          </div>
        </article>
      </section>

      <div className="form-actions">
        <button disabled={busy || daysOfWeek.length === 0} type="submit">
          {busy ? "Saving..." : "Finish Setup"}
        </button>
      </div>
      {error ? (
        <p aria-live="assertive" className="toast error" role="alert">
          {error}
        </p>
      ) : null}
    </form>
  );
}
