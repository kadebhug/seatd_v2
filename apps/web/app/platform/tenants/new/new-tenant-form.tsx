"use client";

import type React from "react";
import { useState } from "react";

export function NewTenantForm() {
  const [organisationName, setOrganisationName] = useState("");
  const [organisationSlug, setOrganisationSlug] = useState("");
  const [locationName, setLocationName] = useState("");
  const [locationSlug, setLocationSlug] = useState("");
  const [timezone, setTimezone] = useState(
    Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
  );
  const [ownerEmail, setOwnerEmail] = useState("");
  const [ownerDisplayName, setOwnerDisplayName] = useState("");
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [createdID, setCreatedID] = useState("");

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setError("");
    setCreatedID("");
    try {
      const response = await fetch("/api/platform/tenants", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          organisationName,
          organisationSlug,
          locationName,
          locationSlug,
          timezone,
          ownerEmail,
          ownerDisplayName,
          reason,
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error?.message ?? "Tenant creation failed.");
      }
      setCreatedID(body.tenant.tenant.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Tenant creation failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="grid two" onSubmit={submit}>
      <section className="panel form">
        <h2>Organisation</h2>
        <label>
          Name
          <input
            onBlur={() => {
              if (!organisationSlug) setOrganisationSlug(slugify(organisationName));
            }}
            onChange={(event) => setOrganisationName(event.target.value)}
            required
            value={organisationName}
          />
        </label>
        <label>
          Slug
          <input
            onChange={(event) => setOrganisationSlug(event.target.value)}
            required
            value={organisationSlug}
          />
        </label>
        <label>
          Reason
          <textarea
            onChange={(event) => setReason(event.target.value)}
            required
            value={reason}
          />
        </label>
      </section>
      <section className="panel form">
        <h2>Initial Owner</h2>
        <label>
          Email
          <input
            autoComplete="email"
            onChange={(event) => setOwnerEmail(event.target.value)}
            required
            type="email"
            value={ownerEmail}
          />
        </label>
        <label>
          Display name
          <input
            onChange={(event) => setOwnerDisplayName(event.target.value)}
            value={ownerDisplayName}
          />
        </label>
      </section>
      <section className="panel form">
        <h2>Location</h2>
        <label>
          Name
          <input
            onBlur={() => {
              if (!locationSlug) setLocationSlug(slugify(locationName));
            }}
            onChange={(event) => setLocationName(event.target.value)}
            required
            value={locationName}
          />
        </label>
        <label>
          Slug
          <input
            onChange={(event) => setLocationSlug(event.target.value)}
            required
            value={locationSlug}
          />
        </label>
        <label>
          Timezone
          <input
            onChange={(event) => setTimezone(event.target.value)}
            required
            value={timezone}
          />
        </label>
      </section>
      <section className="panel form">
        <h2>Create</h2>
        <button disabled={busy} type="submit">
          {busy ? "Creating..." : "Create Tenant"}
        </button>
        {createdID ? (
          <a className="button-link" href={`/platform?tenantId=${createdID}`}>
            Open Tenant
          </a>
        ) : null}
        {error ? (
          <p aria-live="assertive" className="toast error" role="alert">
            {error}
          </p>
        ) : null}
      </section>
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
