"use client";

import { useMemo, useState } from "react";

type TableDraft = {
  label: string;
  capacityLabel: string;
};

type StaffDraft = {
  name: string;
  email: string;
  role: string;
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
  const [organisationName, setOrganisationName] = useState("");
  const [organisationSlug, setOrganisationSlug] = useState("");
  const [locationName, setLocationName] = useState("");
  const [locationSlug, setLocationSlug] = useState("");
  const [timezone, setTimezone] = useState(
    Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
  );
  const [floorName, setFloorName] = useState("Main floor");
  const [floorSlug, setFloorSlug] = useState("main-floor");
  const [zoneName, setZoneName] = useState("Dining room");
  const [tables, setTables] = useState(defaultTables);
  const [serviceName, setServiceName] = useState("Dinner");
  const [startTime, setStartTime] = useState("17:00");
  const [endTime, setEndTime] = useState("22:00");
  const [daysOfWeek, setDaysOfWeek] = useState([1, 2, 3, 4, 5, 6]);
  const [staff, setStaff] = useState<StaffDraft[]>([
    { name: "", email: "", role: "waiter" },
  ]);
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

  function updateStaff(index: number, patch: Partial<StaffDraft>) {
    setStaff((current) =>
      current.map((item, candidate) =>
        candidate === index ? { ...item, ...patch } : item,
      ),
    );
  }

  async function submit() {
    setBusy(true);
    setStatus("");
    setError("");
    setPairingCode("");
    try {
      const response = await fetch("/api/onboarding/owner", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          organisationName,
          organisationSlug,
          locationName,
          locationSlug,
          timezone,
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
          staff: staff.filter(
            (item) => item.name.trim() !== "" || item.email.trim() !== "",
          ),
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        setError(body.error?.message ?? "Onboarding failed.");
        return;
      }
      setCreatedLocationId(body.location.id);
      setStatus("Venue created.");
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
          <h2>{organisationName}</h2>
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
          <h2>Organisation</h2>
          <label>
            Name
            <input
              autoComplete="organization"
              onBlur={() => {
                if (!organisationSlug) {
                  setOrganisationSlug(slugify(organisationName));
                }
              }}
              onChange={(event) => setOrganisationName(event.target.value)}
              placeholder={`${displayName}'s restaurant group`}
              required
              value={organisationName}
            />
          </label>
          <label>
            Slug
            <input
              autoComplete="off"
              onChange={(event) => setOrganisationSlug(event.target.value)}
              required
              spellCheck={false}
              value={organisationSlug}
            />
          </label>
        </article>

        <article className="panel form">
          <h2>Location</h2>
          <label>
            Name
            <input
              autoComplete="off"
              onBlur={() => {
                if (!locationSlug) {
                  setLocationSlug(slugify(locationName));
                }
              }}
              onChange={(event) => setLocationName(event.target.value)}
              placeholder="Main Street"
              required
              value={locationName}
            />
          </label>
          <label>
            Slug
            <input
              autoComplete="off"
              onChange={(event) => setLocationSlug(event.target.value)}
              required
              spellCheck={false}
              value={locationSlug}
            />
          </label>
          <label>
            Timezone
            <input
              autoComplete="off"
              onChange={(event) => setTimezone(event.target.value)}
              required
              spellCheck={false}
              value={timezone}
            />
          </label>
        </article>
      </section>

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

      <section className="panel form">
        <h2>Staff</h2>
        <div className="staff-draft-list">
          {staff.map((item, index) => (
            <div className="staff-draft-row" key={index}>
              <label>
                Name
                <input
                  onChange={(event) =>
                    updateStaff(index, { name: event.target.value })
                  }
                  value={item.name}
                />
              </label>
              <label>
                Email
                <input
                  autoComplete="email"
                  onChange={(event) =>
                    updateStaff(index, { email: event.target.value })
                  }
                  type="email"
                  value={item.email}
                />
              </label>
              <label>
                Role
                <select
                  autoComplete="off"
                  onChange={(event) =>
                    updateStaff(index, { role: event.target.value })
                  }
                  value={item.role}
                >
                  <option value="waiter">Waiter</option>
                  <option value="location_manager">Location manager</option>
                  <option value="read_only">Read only</option>
                </select>
              </label>
            </div>
          ))}
        </div>
        <button
          className="secondary"
          onClick={() =>
            setStaff((current) => [
              ...current,
              { name: "", email: "", role: "waiter" },
            ])
          }
          type="button"
        >
          Add Staff Row
        </button>
      </section>

      <div className="form-actions">
        <button disabled={busy || daysOfWeek.length === 0} type="submit">
          {busy ? "Creating..." : "Create Venue"}
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

function slugify(value: string) {
  return value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}
