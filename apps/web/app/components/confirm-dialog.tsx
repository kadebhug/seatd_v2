"use client";

import { useEffect, useRef, useState } from "react";

type Props = {
  open: boolean;
  title: string;
  description: string;
  confirmationPhrase: string;
  confirmationLabel: string;
  reasonLabel?: string;
  actionLabel: string;
  destructive?: boolean;
  busy?: boolean;
  errorMessage?: string | null;
  onCancel: () => void;
  onConfirm: (reason: string) => void;
};

export function TypeToConfirmDialog({
  open,
  title,
  description,
  confirmationPhrase,
  confirmationLabel,
  reasonLabel = "Reason",
  actionLabel,
  destructive = true,
  busy = false,
  errorMessage,
  onCancel,
  onConfirm,
}: Readonly<Props>) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const [typedValue, setTypedValue] = useState("");
  const [reason, setReason] = useState("");

  useEffect(() => {
    const node = dialogRef.current;
    if (!node) return;
    if (open && !node.open) node.showModal();
    if (!open && node.open) node.close();
  }, [open]);

  useEffect(() => {
    if (open) {
      setTypedValue("");
      setReason("");
    }
  }, [open]);

  const canConfirm =
    typedValue === confirmationPhrase && reason.trim().length > 0 && !busy;

  return (
    <dialog
      className="confirm-dialog"
      onCancel={(event) => {
        event.preventDefault();
        onCancel();
      }}
      ref={dialogRef}
    >
      <form
        method="dialog"
        onSubmit={(event) => {
          event.preventDefault();
          if (canConfirm) onConfirm(reason.trim());
        }}
      >
        <h2>{title}</h2>
        <p>{description}</p>
        <label>
          {confirmationLabel}
          <input
            autoComplete="off"
            onChange={(event) => setTypedValue(event.target.value)}
            value={typedValue}
          />
        </label>
        <label>
          {reasonLabel}
          <textarea
            onChange={(event) => setReason(event.target.value)}
            required
            value={reason}
          />
        </label>
        {errorMessage ? (
          <p className="toast error" role="alert">
            {errorMessage}
          </p>
        ) : null}
        <div className="confirm-dialog-actions">
          <button
            className="secondary"
            onClick={onCancel}
            type="button"
          >
            Cancel
          </button>
          <button
            className={destructive ? "danger" : undefined}
            disabled={!canConfirm}
            type="submit"
          >
            {busy ? "Working…" : actionLabel}
          </button>
        </div>
      </form>
    </dialog>
  );
}
