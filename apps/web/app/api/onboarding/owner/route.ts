import { getSession } from "../../../../lib/session";

export async function POST(request: Request) {
  const session = await getSession();
  if (!session.sessionSecret) {
    return Response.json(
      { error: { code: "unauthorized", message: "web session is required" } },
      { status: 401 },
    );
  }

  const baseUrl = process.env.SEATD_API_BASE_URL ?? "http://localhost:8080";
  const response = await fetch(new URL("/v1/onboarding/owner", baseUrl), {
    method: "POST",
    cache: "no-store",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${session.sessionSecret}`,
      "Content-Type": "application/json",
    },
    body: await request.text(),
  });

  const body = await response.text();
  return new Response(body, {
    status: response.status,
    headers: { "Content-Type": "application/json" },
  });
}
