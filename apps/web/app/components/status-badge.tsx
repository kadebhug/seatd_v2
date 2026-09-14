"use client";

import {
  CheckCircleIcon,
  ClockIcon,
  ProhibitIcon,
  WarningCircleIcon,
  WifiHighIcon,
  WifiSlashIcon,
} from "@phosphor-icons/react";
import { deviceHealthLabel, occupancyLabel } from "../../lib/occupancy";

type StatusVariant =
  | "available"
  | "occupied"
  | "attention"
  | "online"
  | "offline"
  | "revoked"
  | "active"
  | "disabled"
  | "processed"
  | "pending"
  | "running"
  | "completed"
  | "failed"
  | "stale";

const variantConfig: Record<
  StatusVariant,
  { label: string; className: string; Icon: typeof CheckCircleIcon }
> = {
  available: {
    label: "Available",
    className: "status-available",
    Icon: CheckCircleIcon,
  },
  occupied: {
    label: "Occupied",
    className: "status-occupied",
    Icon: ClockIcon,
  },
  attention: {
    label: "Attention",
    className: "status-attention",
    Icon: WarningCircleIcon,
  },
  online: {
    label: "Online",
    className: "status-available",
    Icon: WifiHighIcon,
  },
  offline: {
    label: "Offline",
    className: "status-attention",
    Icon: WifiSlashIcon,
  },
  revoked: {
    label: "Revoked",
    className: "status-revoked",
    Icon: ProhibitIcon,
  },
  active: {
    label: "Active",
    className: "status-available",
    Icon: CheckCircleIcon,
  },
  disabled: {
    label: "Disabled",
    className: "status-revoked",
    Icon: ProhibitIcon,
  },
  processed: {
    label: "Processed",
    className: "status-available",
    Icon: CheckCircleIcon,
  },
  pending: {
    label: "Pending",
    className: "status-attention",
    Icon: ClockIcon,
  },
  running: {
    label: "Running",
    className: "status-attention",
    Icon: ClockIcon,
  },
  completed: {
    label: "Completed",
    className: "status-available",
    Icon: CheckCircleIcon,
  },
  failed: {
    label: "Failed",
    className: "status-failed",
    Icon: WarningCircleIcon,
  },
  stale: {
    label: "Stale",
    className: "status-attention",
    Icon: WarningCircleIcon,
  },
};

export function StatusBadge({
  variant,
  label,
}: Readonly<{ variant: StatusVariant | string; label?: string }>) {
  const config = variantConfig[variant as StatusVariant] ?? {
    label: occupancyLabel(variant) || deviceHealthLabel(variant),
    className: "status-neutral",
    Icon: ClockIcon,
  };
  const displayLabel = label ?? config.label;
  const Icon = config.Icon;

  return (
    <span className={`status-badge ${config.className}`}>
      <Icon aria-hidden="true" size={14} weight="bold" />
      {displayLabel}
    </span>
  );
}
