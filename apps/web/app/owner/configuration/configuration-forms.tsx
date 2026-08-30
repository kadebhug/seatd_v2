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
  const location = snapshot.locations[0];

  async function submit(path: string, body: unknown) {
    setMessage("");
    const response = await fetch(path, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    setMessage(response.ok ? "Saved" : await response.text());
  }

  return (
    <section className="grid two">
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
          <input name="name" defaultValue={snapshot.organisation.name} />
        </label>
        <label>
          Slug
          <input name="slug" defaultValue={snapshot.organisation.slug} />
        </label>
        <label>
          Status
          <select name="status" defaultValue={snapshot.organisation.status}>
            <option value="active">Active</option>
            <option value="disabled">Disabled</option>
          </select>
        </label>
        <button type="submit">Save organisation</button>
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
            <input name="name" defaultValue={location.name} />
          </label>
          <label>
            Timezone
            <input name="timezone" defaultValue={location.timezone} />
          </label>
          <label>
            Status
            <select name="status" defaultValue={location.status}>
              <option value="active">Active</option>
              <option value="disabled">Disabled</option>
            </select>
          </label>
          <label>
            Feature flags
            <textarea
              name="featureFlags"
              defaultValue={JSON.stringify(location.featureFlags, null, 2)}
            />
          </label>
          <button type="submit">Save location</button>
        </form>
      ) : null}

      {location ? (
        <form
          className="panel form"
          onSubmit={(event) => {
            event.preventDefault();
            const data = new FormData(event.currentTarget);
            void submit("/api/owner/service-periods", {
              name: data.get("name"),
              daysOfWeek: data.getAll("daysOfWeek").map(Number),
              startTime: data.get("startTime"),
              endTime: data.get("endTime"),
            });
          }}
        >
          <h2>Service periods</h2>
          <div className="service-period-list">
            {servicePeriods.map((period) => (
              <div className="service-period-row" key={period.id}>
                <span>{period.name}</span>
                <strong>
                  {dayNames(period.daysOfWeek)} {period.startTime}-
                  {period.endTime}
                </strong>
                <button
                  className="danger"
                  disabled={!period.isActive}
                  type="button"
                  onClick={() => {
                    void submit(
                      `/api/owner/service-periods/${period.id}/archive`,
                      { expectedVersion: period.version },
                    );
                  }}
                >
                  Archive
                </button>
              </div>
            ))}
          </div>
          <label>
            Name
            <input name="name" placeholder="Dinner" />
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
            <input name="startTime" type="time" />
          </label>
          <label>
            End
            <input name="endTime" type="time" />
          </label>
          <button type="submit">Add service period</button>
        </form>
      ) : null}
      {message ? <p className="toast">{message}</p> : null}
    </section>
  );
}

function dayNames(days: number[]) {
  const names = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  return days.map((day) => names[day] ?? String(day)).join(", ");
}
