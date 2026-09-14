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

const platformDemoSession: SeatdSession = {
  actorRef: "user:aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaa2",
  displayName: "Platform Demo",
  organisationId: "11111111-1111-1111-1111-111111111111",
  locationId: "",
  roles: ["platform_admin"],
};

export async function getSession(): Promise<SeatdSession> {
  const jar = await cookies();
  const dev = isDevelopmentEnvironment();
  let secret: string | null = null;
  try {
    secret = unsignValue(jar.get(sessionCookieName)?.value);
  } catch (error) {
    if (!dev) {
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
    dev &&
    process.env.SEATD_WEB_DEV_SESSION !== "disabled"
  ) {
    if (process.env.SEATD_WEB_DEV_SESSION === "platform") {
      return platformDemoSession;
    }
    return demoSession;
  }

  if (!process.env.SEATD_OIDC_ISSUER || !process.env.SEATD_OIDC_CLIENT_ID) {
    throw new Error(
      "OIDC configuration is required outside local/test development",
    );
  }
  redirect("/api/auth/login?returnTo=/owner");
}

export async function getCookieSession(): Promise<SeatdSession | null> {
  const jar = await cookies();
  const dev = isDevelopmentEnvironment();
  let secret: string | null = null;
  try {
    secret = unsignValue(jar.get(sessionCookieName)?.value);
  } catch (error) {
    if (!dev) {
      throw error;
    }
  }
  if (!secret) {
    return null;
  }

  try {
    const apiSession = await validateAPISession(secret);
    return apiSession ? seatdSessionFromAPI(apiSession, secret) : null;
  } catch (error) {
    console.error("optional session validation failed", error);
    return null;
  }
}

export function canAccessOwner(session: SeatdSession): boolean {
  return session.roles.some(
    (role) => role === "organisation_owner" || role === "location_manager",
  );
}

export function canAccessPlatform(session: SeatdSession): boolean {
  return session.roles.some((role) => role === "platform_admin");
}

function seatdSessionFromAPI(
  apiSession: APIWebSession,
  secret: string,
): SeatdSession {
  const locations = apiSession.locations ?? [];
  const memberships = apiSession.memberships ?? [];
  const location =
    locations.find((item) => item.status === "active") ?? locations[0];
  const organisationId =
    location?.organisationId ?? memberships[0]?.organisationId ?? "";
  const tenantRoles = memberships
    .filter((membership) => membership.organisationId === organisationId)
    .filter(
      (membership) =>
        !location ||
        !membership.locationId ||
        membership.locationId === location.id,
    )
    .map((membership) => membership.role as SeatdRole);
  const platformRoles = memberships
    .filter((membership) => !membership.organisationId)
    .map((membership) => membership.role as SeatdRole);
  return {
    actorRef: apiSession.actorRef,
    displayName: apiSession.displayName,
    organisationId,
    locationId: location?.id ?? "",
    roles: Array.from(new Set([...tenantRoles, ...platformRoles])),
    sessionSecret: secret,
  };
}

function isDevelopmentEnvironment() {
  return process.env.SEATD_ENV === "local" || process.env.SEATD_ENV === "test";
}
