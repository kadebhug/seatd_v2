"use client";

import type { FormEvent } from "react";
import { useMemo, useState } from "react";
import type {
  Location,
  Membership,
} from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../../components/status-badge";

type Scope = "organisation" | "location";
type Role = "organisation_owner" | "location_manager" | "waiter" | "read_only";

type Props = {
  initialMemberships: Membership[];
  locations: Location[];
};

export function MemberManager({ initialMemberships, locations }: Props) {
  const [memberships, setMemberships] = useState(initialMemberships);
  const [scope, setScope] = useState<Scope>("location");
  const [locationId, setLocationId] = useState(locations[0]?.id ?? "");
  const [role, setRole] = useState<Role>("waiter");
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const locationById = useMemo(
    () => new Map(locations.map((location) => [location.id, location])),
    [locations],
  );
  const availableRoles = scope === "organisation" ? organisationRoles : locationRoles;

  async function inviteMember(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setMessage(null);
    try {
      const response = await fetch("/api/owner/memberships", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          scope,
          locationId: scope === "location" ? locationId : undefined,
          role,
          email,
          displayName,
        }),
      });
      const body = await response.json();
      if (!response.ok) {
        setMessage(body.error?.message ?? "Invite failed.");
        return;
      }
      setMemberships((current) => upsertMembership(current, body));
      setEmail("");
      setDisplayName("");
      setMessage("Member access saved.");
    } finally {
      setBusy(false);
    }
  }

  async function updateRole(membership: Membership, nextRole: Role) {
    setBusy(true);
    setMessage(null);
    try {
      const response = await fetch(
        `/api/owner/memberships/${membership.scope}/${membership.id}`,
        {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ role: nextRole }),
        },
      );
      const body = await response.json();
      if (!response.ok) {
        setMessage(body.error?.message ?? "Role update failed.");
        return;
      }
      setMemberships((current) => upsertMembership(current, body));
      setMessage("Role updated.");
    } finally {
      setBusy(false);
    }
  }

  async function disableMember(membership: Membership) {
    const label = membership.displayName ?? membership.email ?? membership.memberRef;
    if (!window.confirm(`Disable access for "${label}"?`)) {
      return;
    }
    setBusy(true);
    setMessage(null);
    try {
      const response = await fetch(
        `/api/owner/memberships/${membership.scope}/${membership.id}/disable`,
        { method: "POST" },
      );
      const body = await response.json();
      if (!response.ok) {
        setMessage(body.error?.message ?? "Disable failed.");
        return;
      }
      setMemberships((current) => upsertMembership(current, body));
      setMessage("Member disabled.");
    } finally {
      setBusy(false);
    }
  }

  function changeScope(nextScope: Scope) {
    setScope(nextScope);
    setRole(nextScope === "organisation" ? "read_only" : "waiter");
  }

  return (
    <section className="grid members-grid">
      <form className="panel form member-invite-form" onSubmit={inviteMember}>
        <h2>Invite member</h2>
        <label>
          Email
          <input
            autoComplete="email"
            name="email"
            onChange={(event) => setEmail(event.target.value)}
            required
            type="email"
            value={email}
          />
        </label>
        <label>
          Display name
          <input
            autoComplete="name"
            name="displayName"
            onChange={(event) => setDisplayName(event.target.value)}
            type="text"
            value={displayName}
          />
        </label>
        <label>
          Scope
          <select
            name="scope"
            onChange={(event) => changeScope(event.target.value as Scope)}
            value={scope}
          >
            <option value="location">Location</option>
            <option value="organisation">Organisation</option>
          </select>
        </label>
        {scope === "location" ? (
          <label>
            Location
            <select
              name="locationId"
              onChange={(event) => setLocationId(event.target.value)}
              required
              value={locationId}
            >
              {locations.map((location) => (
                <option key={location.id} value={location.id}>
                  {location.name}
                </option>
              ))}
            </select>
          </label>
        ) : null}
        <label>
          Role
          <select
            name="role"
            onChange={(event) => setRole(event.target.value as Role)}
            value={role}
          >
            {availableRoles.map((item) => (
              <option key={item.value} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
        <button disabled={busy || !email || (scope === "location" && !locationId)} type="submit">
          {busy ? "Saving..." : "Save Access"}
        </button>
        {message ? (
          <p aria-live="polite" className="toast" role="status">
            {message}
          </p>
        ) : null}
      </form>

      <section className="member-list">
        {memberships.length > 0 ? (
          memberships.map((membership) => (
            <article className="panel member-card" key={`${membership.scope}:${membership.id}`}>
              <div className="member-title-row">
                <StatusBadge
                  label={membership.disabledAt ? "disabled" : "active"}
                  variant={membership.disabledAt ? "disabled" : "active"}
                />
                <div>
                  <h2>{membership.displayName ?? membership.email ?? membership.memberRef}</h2>
                  <p>{membership.email ?? membership.memberRef}</p>
                </div>
              </div>
              <dl className="compact-list">
                <dt>Scope</dt>
                <dd>{membership.scope}</dd>
                <dt>Location</dt>
                <dd>
                  {membership.locationId
                    ? locationById.get(membership.locationId)?.name ?? membership.locationId
                    : "All locations"}
                </dd>
                <dt>Permissions</dt>
                <dd>{membership.permissions?.map(humanize).join(", ") || "None"}</dd>
                {membership.disabledAt ? (
                  <>
                    <dt>Disabled</dt>
                    <dd>{formatDate(membership.disabledAt)}</dd>
                  </>
                ) : null}
              </dl>
              <div className="member-actions">
                <select
                  aria-label="Role"
                  disabled={busy || Boolean(membership.disabledAt)}
                  onChange={(event) => void updateRole(membership, event.target.value as Role)}
                  value={membership.role}
                >
                  {(membership.scope === "organisation" ? organisationRoles : locationRoles).map(
                    (item) => (
                      <option key={item.value} value={item.value}>
                        {item.label}
                      </option>
                    ),
                  )}
                </select>
                <button
                  className="danger"
                  disabled={busy || Boolean(membership.disabledAt)}
                  onClick={() => void disableMember(membership)}
                  type="button"
                >
                  Disable
                </button>
              </div>
            </article>
          ))
        ) : (
          <article className="panel">
            <p className="empty-state">No members have been provisioned yet.</p>
          </article>
        )}
      </section>
    </section>
  );
}

const organisationRoles = [
  { value: "read_only", label: "Read only" },
  { value: "organisation_owner", label: "Organisation owner" },
] satisfies { value: Role; label: string }[];

const locationRoles = [
  { value: "waiter", label: "Waiter" },
  { value: "location_manager", label: "Location manager" },
] satisfies { value: Role; label: string }[];

function upsertMembership(items: Membership[], next: Membership) {
  const index = items.findIndex(
    (item) => item.scope === next.scope && item.id === next.id,
  );
  if (index === -1) {
    return [next, ...items];
  }
  return items.map((item, itemIndex) => (itemIndex === index ? next : item));
}

function humanize(value: string) {
  return value.replaceAll(".", " ").replaceAll("_", " ");
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
