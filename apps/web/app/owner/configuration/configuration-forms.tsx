"use client";

import type {
  OwnerSnapshotResponse,
  ServicePeriod,
} from "@seatd/typescript-seatd-client";
import { useState } from "react";

export function ConfigurationForms({
  snapshot,
  servicePeriods,
}: Readonly<{
  snapshot: OwnerSnapshotResponse;
  servicePeriods: ServicePeriod[];
}>) {
  const [message, setMessage] = useState("");
  const [saving, setSaving] = useState(false);
  const location = snapshot.locations[0];

  async function submit(path: string, body: unknown, method = "PUT") {
    setSaving(true);
    setMessage("");
    try {
      const response = await fetch(path, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      setMessage(response.ok ? "Saved." : await response.text());
    } finally {
      setSaving(false);
    }
  }

  async function archivePeriod(period: ServicePeriod) {
    if (
      !window.confirm(
        `Archive service period "${period.name}"? This stops it from applying to new service windows.`,
      )
    ) {
      return;
    }
    await submit(
      `/api/owner/service-periods/${period.id}/archive`,
      { expectedVersion: period.version },
      "POST",
    );
  }

  return (
    <section className="grid two configuration-forms">
      <form
        className="panel form"
        onSubmit={(event) => {
          event.preventDefault();
          const data = new FormData(event.currentTarget);
          void submit(`/api/owner/organisations/${snapshot.organisation.id}`, {
            slug: data.get("slug"),
            name: data.get("name"),
            status: data.get("status"),
          });
        }}
      >
        <h2>Organisation</h2>
        <label>
          Name
          <input
            autoComplete="organization"
            defaultValue={snapshot.organisation.name}
            name="name"
            required
          />
        </label>
        <label>
          Slug
          <input
            autoComplete="off"
            defaultValue={snapshot.organisation.slug}
            name="slug"
            required
            spellCheck={false}
          />
        </label>
        <label>
          Status
          <select
            autoComplete="off"
            defaultValue={snapshot.organisation.status}
            name="status"
          >
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
          </select>
        </label>
        <button disabled={saving} type="submit">
          {saving ? "Saving…" : "Save Organisation"}
        </button>
      </form>

      {location ? (
        <form
          className="panel form"
          onSubmit={(event) => {
            event.preventDefault();
            const data = new FormData(event.currentTarget);
            void submit(`/api/owner/locations/${location.id}`, {
              name: data.get("name"),
              timezone: data.get("timezone"),
              status: data.get("status"),
              operatingConfig: location.operatingConfig,
              featureFlags: JSON.parse(
                String(data.get("featureFlags") || "{}"),
              ),
            });
          }}
        >
          <h2>Location</h2>
          <label>
            Name
            <input
              autoComplete="off"
              defaultValue={location.name}
              name="name"
              required
            />
          </label>
          <label>
            Timezone
            <input
              autoComplete="off"
              defaultValue={location.timezone}
              name="timezone"
              placeholder="Europe/London…"
              required
              spellCheck={false}
            />
          </label>
          <label>
            Status
            <select
              autoComplete="off"
              defaultValue={location.status}
              name="status"
            >
              <option value="active">Active</option>
              <option value="disabled">Disabled</option>
            </select>
          </label>
          <label>
            Feature flags
            <textarea
              autoComplete="off"
              defaultValue={JSON.stringify(location.featureFlags, null, 2)}
              name="featureFlags"
              spellCheck={false}
            />
          </label>
          <button disabled={saving} type="submit">
            {saving ? "Saving…" : "Save Location"}
          </button>
        </form>
      ) : null}

      {location ? (
        <form
          className="panel form service-periods-form"
          onSubmit={(event) => {
            event.preventDefault();
            const data = new FormData(event.currentTarget);
            void submit(
              "/api/owner/service-periods",
              {
                name: data.get("name"),
                daysOfWeek: data.getAll("daysOfWeek").map(Number),
                startTime: data.get("startTime"),
                endTime: data.get("endTime"),
              },
              "POST",
            );
          }}
        >
          <h2>Service Periods</h2>
          <div className="service-period-list">
            {servicePeriods.length > 0 ? (
              servicePeriods.map((period) => (
                <div className="service-period-row" key={period.id}>
                  <span>{period.name}</span>
                  <strong className="font-mono tabular-nums">
                    {dayNames(period.daysOfWeek)} {period.startTime}-
                    {period.endTime}
                  </strong>
                  <button
                    className="danger"
                    disabled={!period.isActive || saving}
                    type="button"
                    onClick={() => void archivePeriod(period)}
                  >
                    Archive
                  </button>
                </div>
              ))
            ) : (
              <p className="empty-state">No service periods yet.</p>
            )}
          </div>
          <label>
            Name
            <input name="name" placeholder="Dinner…" required />
          </label>
          <fieldset className="weekday-picker">
            <legend>Days</legend>
            {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map(
              (label, index) => (
                <label key={label}>
                  <input name="daysOfWeek" type="checkbox" value={index} />
                  {label}
                </label>
              ),
            )}
          </fieldset>
          <label>
            Start
            <input name="startTime" required type="time" />
          </label>
          <label>
            End
            <input name="endTime" required type="time" />
          </label>
          <button disabled={saving} type="submit">
            {saving ? "Adding…" : "Add Service Period"}
          </button>
        </form>
      ) : null}
      {message ? (
        <p aria-live="polite" className="toast" role="status">
          {message}
        </p>
      ) : null}
    </section>
  );
}

function dayNames(days: number[]) {
  const names = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  return days.map((day) => names[day] ?? String(day)).join(", ");
}
