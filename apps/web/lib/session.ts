import { cookies } from "next/headers";

export type SeatdRole =
  | "platform_admin"
  | "organisation_owner"
  | "location_manager"
  | "waiter"
  | "read_only"
  | "support";

export interface SeatdSession {
  actorRef: string;
  displayName: string;
  organisationId: string;
  locationId: string;
  roles: SeatdRole[];
}

const demoSession: SeatdSession = {
  actorRef: "user:owner-demo",
  displayName: "Owner Demo",
  organisationId: "11111111-1111-1111-1111-111111111111",
  locationId: "22222222-2222-2222-2222-222222222222",
  roles: ["organisation_owner"],
};

export async function getSession(): Promise<SeatdSession> {
  const jar = await cookies();
  const encoded = jar.get("seatd_web_session")?.value;
  if (encoded) {
    return JSON.parse(Buffer.from(encoded, "base64url").toString("utf8"));
  }

  const env = process.env.SEATD_ENV ?? "local";
  if (
    (env === "local" || env === "test") &&
    process.env.SEATD_WEB_DEV_SESSION !== "disabled"
  ) {
    return demoSession;
  }

  if (!process.env.SEATD_OIDC_ISSUER || !process.env.SEATD_OIDC_CLIENT_ID) {
    throw new Error(
      "OIDC configuration is required outside local/test development",
    );
  }
  throw new Error("OIDC callback flow is not configured for this environment");
}

export function canAccessOwner(session: SeatdSession): boolean {
  return session.roles.some(
    (role) => role === "organisation_owner" || role === "location_manager",
  );
}

export function canAccessPlatform(session: SeatdSession): boolean {
  return session.roles.some(
    (role) => role === "platform_admin" || role === "support",
  );
}
