import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import {
  revokeAPISession,
  sessionCookieName,
  unsignValue,
  webBaseURL,
} from "../../../../lib/auth";

export const runtime = "nodejs";

export async function POST() {
  const jar = await cookies();
  const secret = unsignValue(jar.get(sessionCookieName)?.value);
  if (secret) {
    await revokeAPISession(secret);
  }
  jar.delete(sessionCookieName);
  return NextResponse.redirect(new URL("/", webBaseURL()));
}
