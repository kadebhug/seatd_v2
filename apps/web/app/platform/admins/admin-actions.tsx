"use client";

import { useState } from "react";
import type {
  PlatformAdmin,
  PlatformAdminGrant,
  PlatformAdminsResponse,
} from "@seatd/typescript-seatd-client";
import { StatusBadge } from "../../components/status-badge";
import { TypeToConfirmDialog } from "../../components/confirm-dialog";

type Props = {
  initialData: PlatformAdminsResponse;
};

type PendingAction =
  | { type: "revoke-admin"; admin: PlatformAdmin }
  | { type: "reactivate-admin"; admin: PlatformAdmin }
  | { type: "revoke-invitation"; grant: PlatformAdminGrant }
  | null;

export function AdminActions({ initialData }: Readonly<Props>) {
  const [data, setData] = useState(initialData);
  const [email, setEmail] = useState("");
  const [role, setRole] = useState<"platform_admin" | "support">(
    "platform_admin",
  );
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [pending, setPending] = useState<PendingAction>(null);

  async function refresh() {
    const response = await fetch("/api/platform/admins");
    const body = await response.json();
    if (!response.ok) {
      throw new Error(body.error?.message ?? "Request failed.");
    }
    setData(body as PlatformAdminsResponse);
  }

  async function invite() {
    setBusy(true);
    setError("");
    try {
      const response = await fetch("/api/platform/admins/invitations", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, role, reason }),
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error?.message ?? "Invite failed.");
      }
      setEmail("");
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
      const path =
        pending.type === "revoke-invitation"
          ? `/api/platform/admins/invitations/${pending.grant.id}`
          : `/api/platform/admins/${pending.admin.id}/${pending.type === "revoke-admin" ? "revoke" : "reactivate"}`;
      const response = await fetch(path, {
        method: pending.type === "revoke-invitation" ? "DELETE" : "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: actionReason }),
      });
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error?.message ?? "Request failed.");
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
      <section className="panel form">
        <h2>Invite Admin</h2>
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
          Role
          <select
            onChange={(event) =>
              setRole(event.target.value as "platform_admin" | "support")
            }
            value={role}
          >
            <option value="platform_admin">Platform admin</option>
            <option value="support">Support</option>
          </select>
        </label>
        <label>
          Reason
          <textarea
            onChange={(event) => setReason(event.target.value)}
            value={reason}
          />
        </label>
        <button disabled={busy || !email || !reason.trim()} onClick={invite}>
          {busy ? "Inviting..." : "Invite"}
        </button>
        {error ? (
          <p aria-live="assertive" className="toast error" role="alert">
            {error}
          </p>
        ) : null}
      </section>

      <section className="panel">
        <h2>Pending Invitations</h2>
        <div className="platform-table">
          {data.invitations.filter((grant) => !grant.consumedAt && !grant.revokedAt)
            .length > 0 ? (
            data.invitations
              .filter((grant) => !grant.consumedAt && !grant.revokedAt)
              .map((grant) => (
                <div className="platform-table-row" key={grant.id}>
                  <span>
                    <strong>{grant.email}</strong>
                    <small>{humanize(grant.role)}</small>
                  </span>
                  <button
                    className="secondary"
                    onClick={() => setPending({ type: "revoke-invitation", grant })}
                    type="button"
                  >
                    Revoke
                  </button>
                </div>
              ))
          ) : (
            <p className="empty-state">No pending invitations.</p>
          )}
        </div>
      </section>

      <section className="panel grid-span">
        <h2>Admins</h2>
        <div className="platform-table">
          {data.admins.map((admin) => (
            <div className="platform-table-row" key={admin.id}>
              <span>
                <strong>{admin.displayName || admin.userProfileId}</strong>
                <small>{admin.email ?? admin.userProfileId}</small>
              </span>
              <span>{humanize(admin.role)}</span>
              <StatusBadge
                label={admin.disabledAt ? "disabled" : "active"}
                variant={admin.disabledAt ? "disabled" : "active"}
              />
              <button
                className="secondary"
                onClick={() =>
                  setPending({
                    type: admin.disabledAt ? "reactivate-admin" : "revoke-admin",
                    admin,
                  })
                }
                type="button"
              >
                {admin.disabledAt ? "Reactivate" : "Revoke"}
              </button>
            </div>
          ))}
        </div>
      </section>

      <TypeToConfirmDialog
        actionLabel={pending?.type === "reactivate-admin" ? "Reactivate" : "Revoke"}
        busy={busy}
        confirmationLabel={`Type "${confirmationPhrase(pending)}" to confirm`}
        confirmationPhrase={confirmationPhrase(pending)}
        description="This changes platform access immediately."
        destructive={pending?.type !== "reactivate-admin"}
        errorMessage={error}
        onCancel={() => {
          setPending(null);
          setError("");
        }}
        onConfirm={confirm}
        open={pending !== null}
        title={pending?.type === "reactivate-admin" ? "Reactivate admin" : "Revoke access"}
      />
    </section>
  );
}

function confirmationPhrase(action: PendingAction) {
  if (!action) return "";
  if (action.type === "revoke-invitation") return action.grant.email;
  return action.admin.email ?? action.admin.userProfileId;
}

function humanize(value: string) {
  return value
    .split("_")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
