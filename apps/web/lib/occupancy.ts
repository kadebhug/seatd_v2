export function occupancyLabel(status: string): string {
  switch (status) {
    case "available":
      return "Available";
    case "occupied":
      return "Occupied";
    case "attention":
      return "Attention";
    default:
      return status.charAt(0).toUpperCase() + status.slice(1);
  }
}

export function deviceHealthLabel(health: string): string {
  switch (health) {
    case "online":
      return "Online";
    case "offline":
      return "Offline";
    case "revoked":
      return "Revoked";
    default:
      return health.charAt(0).toUpperCase() + health.slice(1);
  }
}
