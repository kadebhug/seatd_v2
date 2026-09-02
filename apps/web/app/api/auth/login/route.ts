import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import {
  authorizationURL,
  encodeTransaction,
  fetchDiscovery,
  oidcTransactionCookieName,
  randomURLToken,
  webBaseURL,
} from "../../../../lib/auth";

export const runtime = "nodejs";

export async function GET(request: Request) {
  const url = new URL(request.url);
  const returnTo = safeReturnTo(url.searchParams.get("returnTo"));
  const transaction = {
    state: randomURLToken(),
    nonce: randomURLToken(),
    verifier: randomURLToken(),
    returnTo,
    issuedAt: Date.now(),
  };
  const discovery = await fetchDiscovery();
  const jar = await cookies();
  jar.set(oidcTransactionCookieName, encodeTransaction(transaction), {
    httpOnly: true,
    sameSite: "lax",
    secure: webBaseURL().startsWith("https://"),
    path: "/",
    maxAge: 10 * 60,
  });
  return NextResponse.redirect(authorizationURL(discovery, transaction));
}

function safeReturnTo(value: string | null): string {
  if (!value || !value.startsWith("/") || value.startsWith("//")) {
    return "/owner";
  }
  return value;
}
