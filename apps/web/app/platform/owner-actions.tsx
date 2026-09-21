"use client";

import { useEffect, useState } from "react";
import type {
  Membership,
  PlatformAuditEvent,
  PlatformOwnersResponse,
  PlatformTenantAuditResponse,
} from "@seatd/typescript-seatd-client";
import { TypeToConfirmDialog } from "../components/confirm-dialog";
import { StatusBadge } from "../components/status-badge";

type Props = {
  tenantId: string;
  canWrite?: boolean;
};

type PendingAction =
  | { type: "suspend"; owner: Membership }
  | { type: "reactivate"; owner: Membership }
  | { type: "reassign"; owner: Membership; toEmail: string }
  | null;

export function OwnerActions({
  tenantId,
  canWrite = false,
}: Readonly<Props>) {
  const [owners, setOwners] = useState<Membership[]>([]);
  const [events, setEvents] = useState<PlatformAuditEvent[]>([]);
  const [email, setEmail] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [reason, setReason] = useState("");
  const [reassignEmail, setReassignEmail] = useState<Record<string, string>>({});
  const [pending, setPending] = useState<PendingAction>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    void refresh();
  }, [tenantId]);

  async function refresh() {
    const [ownersResponse, auditResponse] = await Promise.all([
      fetch(`/api/platform/tenants/${tenantId}/owners`),
      fetch(`/api/platform/tenants/${tenantId}/audit?limit=8`),
    ]);
    const ownersBody = await ownersResponse.json();
    const auditBody = await auditResponse.json();
    if (ownersResponse.ok) {
      setOwners((ownersBody as PlatformOwnersResponse).owners);
    }
    if (auditResponse.ok) {
      setEvents((auditBody as PlatformTenantAuditResponse).events);
    }
  }

  async function invite() {
    setBusy(true);
    setError("");
    try {
      const response = await fetch(`/api/platform/tenants/${tenantId}/owners`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, displayName, reason }),
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error?.message ?? "Invite failed.");
      }
      setEmail("");
      setDisplayName("");
      setReason("");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Invite failed.");
    } finally {
      setBusy(false);
    }
  }

  async function confirm(actionReason: string) {
    if (!pending) return;
    setBusy(true);
    setError("");
    try {
      const suffix =
        pending.type === "reassign" ? "reassign" : pending.type;
      const body =
        pending.type === "reassign"
          ? { toEmail: pending.toEmail, reason: actionReason }
          : { reason: actionReason };
      const response = await fetch(
        `/api/platform/tenants/${tenantId}/owners/${pending.owner.id}/${suffix}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      );
      const responseBody = await response.json();
      if (!response.ok) {
        throw new Error(responseBody.error?.message ?? "Request failed.");
      }
      setPending(null);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="grid two">
      {canWrite ? (
        <section className="panel form">
          <h2>Owners</h2>
          <label>
            Email
            <input
              autoComplete="email"
              onChange={(event) => setEmail(event.target.value)}
              type="email"
              value={email}
            />
          </label>
          <label>
            Display name
            <input
              onChange={(event) => setDisplayName(event.target.value)}
              value={displayName}
            />
          </label>
          <label>
            Reason
            <textarea
              onChange={(event) => setReason(event.target.value)}
              value={reason}
            />
          </label>
          <button disabled={busy || !email || !reason.trim()} onClick={invite}>
            Invite Owner
          </button>
          {error ? (
            <p aria-live="assertive" className="toast error" role="alert">
              {error}
            </p>
          ) : null}
        </section>
      ) : null}

      <section className={canWrite ? "panel" : "panel grid-span"}>
        <h2>Owner Roster</h2>
        <div className="platform-table">
          {owners.length > 0 ? (
            owners.map((owner) => (
              <div className="platform-owner-row" key={owner.id}>
                <div className="platform-owner-meta">
                  <span>
                    <strong>{owner.displayName ?? owner.memberRef}</strong>
                    <small>{owner.email ?? owner.memberRef}</small>
                  </span>
                  <StatusBadge
                    label={owner.disabledAt ? "Disabled" : "Active"}
                    variant={owner.disabledAt ? "disabled" : "active"}
                  />
                </div>
                {canWrite ? (
                  <div className="platform-owner-actions">
                    <input
                      aria-label="New owner email"
                      onChange={(event) =>
                        setReassignEmail((current) => ({
                          ...current,
                          [owner.id]: event.target.value,
                        }))
                      }
                      placeholder="Reassign to email"
                      type="email"
                      value={reassignEmail[owner.id] ?? ""}
                    />
                    <button
                      className="secondary"
                      onClick={() =>
                        setPending({
                          type: owner.disabledAt ? "reactivate" : "suspend",
                          owner,
                        })
                      }
                      type="button"
                    >
                      {owner.disabledAt ? "Reactivate" : "Suspend"}
                    </button>
                    <button
                      className="secondary"
                      disabled={
                        !reassignEmail[owner.id]?.trim() ||
                        Boolean(owner.disabledAt)
                      }
                      onClick={() =>
                        setPending({
                          type: "reassign",
                          owner,
                          toEmail: reassignEmail[owner.id] ?? "",
                        })
                      }
                      type="button"
                    >
                      Reassign
                    </button>
                  </div>
                ) : null}
              </div>
            ))
          ) : (
            <p className="empty-state">No owners found.</p>
          )}
        </div>
      </section>

      <section className="panel grid-span">
        <h2>Recent Activity</h2>
        <div className="platform-table">
          {events.length > 0 ? (
            events.map((event) => (
              <div className="platform-table-row" key={event.id}>
                <span>
                  <strong>{event.action}</strong>
                  <small>{event.actorRef}</small>
                </span>
                <span>{event.targetType}</span>
                <span>{formatDate(event.createdAt)}</span>
              </div>
            ))
          ) : (
            <p className="empty-state">No recent activity.</p>
          )}
        </div>
      </section>

      <TypeToConfirmDialog
        actionLabel={pending?.type === "reactivate" ? "Reactivate" : pending?.type === "reassign" ? "Reassign" : "Suspend"}
        busy={busy}
        confirmationLabel={`Type "${confirmationPhrase(pending)}" to confirm`}
        confirmationPhrase={confirmationPhrase(pending)}
        description="This changes owner access for this tenant."
        destructive={pending?.type !== "reactivate"}
        errorMessage={error}
        onCancel={() => {
          setPending(null);
          setError("");
        }}
        onConfirm={confirm}
        open={pending !== null}
        title={pending?.type === "reassign" ? "Reassign owner" : pending?.type === "reactivate" ? "Reactivate owner" : "Suspend owner"}
      />
    </section>
  );
}

function confirmationPhrase(action: PendingAction) {
  if (!action) return "";
  return action.owner.email ?? action.owner.memberRef;
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}
