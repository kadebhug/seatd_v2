import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import {
  createAPISession,
  decodeTransaction,
  exchangeCode,
  fetchDiscovery,
  oidcTransactionCookieName,
  safeReturnTo,
  sessionCookieName,
  sessionTTLSeconds,
  signValue,
  verifyIDToken,
  webBaseURL,
} from "../../../../lib/auth";
import { homePath, type SeatdRole } from "../../../../lib/session";

export const runtime = "nodejs";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const code = url.searchParams.get("code");
  const state = url.searchParams.get("state");
  const jar = await cookies();
  const transaction = decodeTransaction(
    jar.get(oidcTransactionCookieName)?.value,
  );
  jar.delete(oidcTransactionCookieName);
  if (!code || !state || !transaction || transaction.state !== state) {
    return NextResponse.redirect(new URL("/auth/error", webBaseURL()));
  }
  try {
    const discovery = await fetchDiscovery();
    const token = await exchangeCode(discovery, code, transaction.verifier);
    if (!token.id_token) {
      throw new Error("OIDC token response did not include id_token");
    }
    const claims = await verifyIDToken(
      discovery,
      token.id_token,
      transaction.nonce,
    );
    const session = await createAPISession(claims);
    if (!session.sessionSecret) {
      throw new Error("Seatd API did not return a session secret");
    }
    jar.set(sessionCookieName, signValue(session.sessionSecret), {
      httpOnly: true,
      sameSite: "lax",
      secure: webBaseURL().startsWith("https://"),
      path: "/",
      maxAge: sessionTTLSeconds(),
    });
    const returnTo = safeReturnTo(transaction.returnTo);
    // Default post-login target is /owner; platform admins should land on /platform.
    const destination =
      returnTo === "/owner"
        ? homePath({
            actorRef: session.actorRef,
            displayName: session.displayName,
            organisationId: "",
            locationId: "",
            roles: (session.memberships ?? []).map(
              (membership) => membership.role as SeatdRole,
            ),
          })
        : returnTo;
    return NextResponse.redirect(new URL(destination, webBaseURL()));
  } catch {
    return NextResponse.redirect(new URL("/auth/error", webBaseURL()));
  }
}
