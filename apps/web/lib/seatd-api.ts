import { getSession } from "./session";

export class SeatdAPIError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly body: string,
  ) {
    super(message);
  }
}

export async function seatdFetch<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const session = await getSession();
  const baseUrl = process.env.SEATD_API_BASE_URL ?? "http://localhost:8080";
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  headers.set("X-Seatd-Organisation-ID", session.organisationId);
  if (session.locationId) {
    headers.set("X-Seatd-Location-ID", session.locationId);
  }
  if (session.sessionSecret) {
    headers.set("Authorization", `Bearer ${session.sessionSecret}`);
  } else if (
    (process.env.SEATD_ENV ?? "local") === "local" ||
    process.env.SEATD_ENV === "test"
  ) {
    headers.set("X-Seatd-Actor-Ref", session.actorRef);
  } else {
    throw new Error("Seatd API session secret is required outside local/test");
  }
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(new URL(path, baseUrl), {
    ...init,
    cache: "no-store",
    headers,
  });
  if (!response.ok) {
    const body = await response.text();
    throw new SeatdAPIError(
      `Seatd API ${response.status}`,
      response.status,
      body,
    );
  }
  return response.json() as Promise<T>;
}

export async function seatdProxy(
  path: string,
  request: Request,
  method = request.method,
): Promise<Response> {
  try {
    const body =
      method === "GET" || method === "HEAD" ? undefined : await request.text();
    const data = await seatdFetch<unknown>(path, { method, body });
    return Response.json(data);
  } catch (error) {
    if (error instanceof SeatdAPIError) {
      return new Response(error.body, {
        status: error.status,
        headers: { "Content-Type": "application/json" },
      });
    }
    return Response.json(
      { error: { code: "internal_error", message: "upstream request failed" } },
      { status: 502 },
    );
  }
}
