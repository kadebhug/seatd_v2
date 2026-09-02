import { cookies } from "next/headers";
import { NextResponse } from "next/server";
import {
  rotateAPISession,
  sessionCookieName,
  sessionTTLSeconds,
  signValue,
  unsignValue,
  webBaseURL,
} from "../../../../lib/auth";

export const runtime = "nodejs";

export async function POST() {
  const jar = await cookies();
  const secret = unsignValue(jar.get(sessionCookieName)?.value);
  if (!secret) {
    return NextResponse.json({ error: "unauthorized" }, { status: 401 });
  }
  const session = await rotateAPISession(secret);
  if (!session.sessionSecret) {
    return NextResponse.json(
      { error: "session rotation failed" },
      { status: 502 },
    );
  }
  jar.set(sessionCookieName, signValue(session.sessionSecret), {
    httpOnly: true,
    sameSite: "lax",
    secure: webBaseURL().startsWith("https://"),
    path: "/",
    maxAge: sessionTTLSeconds(),
  });
  return NextResponse.json({ ok: true });
}
