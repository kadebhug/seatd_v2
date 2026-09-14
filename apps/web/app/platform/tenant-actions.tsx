"use client";

import { useState } from "react";
import type { PlatformTenantDetail } from "@seatd/typescript-seatd-client";
import { TypeToConfirmDialog } from "../components/confirm-dialog";

type Props = {
  detail: PlatformTenantDetail;
  onUpdated: (detail: PlatformTenantDetail) => void;
};

type Action = "suspend" | "reactivate" | null;

export function TenantActions({ detail, onUpdated }: Readonly<Props>) {
  const [action, setAction] = useState<Action>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isActive = detail.tenant.status === "active";

  async function submit(reason: string) {
    if (!action) return;
    setBusy(true);
    setError(null);
    try {
      const response = await fetch(
        `/api/platform/tenants/${detail.tenant.id}/${action}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ reason }),
        },
      );
      const body = await response.json();
      if (!response.ok) {
        throw new Error(body.error?.message ?? "Request failed.");
      }
      onUpdated(body as PlatformTenantDetail);
      setAction(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <div className="platform-tenant-actions">
        <button
          className="secondary"
          disabled={!isActive}
          onClick={() => setAction("suspend")}
          type="button"
        >
          Suspend
        </button>
        <button
          className="secondary"
          disabled={isActive}
          onClick={() => setAction("reactivate")}
          type="button"
        >
          Reactivate
        </button>
      </div>
      <TypeToConfirmDialog
        actionLabel={
          action === "suspend" ? "Suspend organisation" : "Reactivate organisation"
        }
        busy={busy}
        confirmationLabel={`Type "${detail.tenant.name}" to confirm`}
        confirmationPhrase={detail.tenant.name}
        description={
          action === "suspend"
            ? "This immediately marks the organisation as disabled. It can be reactivated at any time."
            : "This marks the organisation as active again."
        }
        destructive={action === "suspend"}
        errorMessage={error}
        onCancel={() => {
          setAction(null);
          setError(null);
        }}
        onConfirm={submit}
        open={action !== null}
        title={action === "suspend" ? "Suspend tenant" : "Reactivate tenant"}
      />
    </>
  );
}
