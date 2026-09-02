import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import {
  type APIWebSession,
  sessionCookieName,
  unsignValue,
  validateAPISession,
} from "./auth";

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
  sessionSecret?: string;
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
  const env = process.env.SEATD_ENV ?? "local";
  let secret: string | null = null;
  try {
    secret = unsignValue(jar.get(sessionCookieName)?.value);
  } catch (error) {
    if (env !== "local" && env !== "test") {
      throw error;
    }
  }
  if (secret) {
    const apiSession = await validateAPISession(secret);
    if (apiSession) {
      return seatdSessionFromAPI(apiSession, secret);
    }
  }

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
  redirect("/api/auth/login?returnTo=/owner");
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

function seatdSessionFromAPI(
  apiSession: APIWebSession,
  secret: string,
): SeatdSession {
  const location =
    apiSession.locations.find((item) => item.status === "active") ??
    apiSession.locations[0];
  const organisationId =
    location?.organisationId ?? apiSession.memberships[0]?.organisationId ?? "";
  const roles = apiSession.memberships
    .filter((membership) => membership.organisationId === organisationId)
    .filter(
      (membership) =>
        !location ||
        !membership.locationId ||
        membership.locationId === location.id,
    )
    .map((membership) => membership.role as SeatdRole);
  return {
    actorRef: apiSession.actorRef,
    displayName: apiSession.displayName,
    organisationId,
    locationId: location?.id ?? "",
    roles: Array.from(new Set(roles)),
    sessionSecret: secret,
  };
}
