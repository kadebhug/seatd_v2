import {
  createHash,
  createHmac,
  createPublicKey,
  createVerify,
  randomBytes,
  timingSafeEqual,
} from "node:crypto";
import type { JsonWebKey as NodeJsonWebKey } from "node:crypto";

export const sessionCookieName = "seatd_web_session";
export const oidcTransactionCookieName = "seatd_oidc_tx";

const textEncoder = new TextEncoder();

type OIDCDiscovery = {
  authorization_endpoint: string;
  token_endpoint: string;
  jwks_uri: string;
  end_session_endpoint?: string;
  issuer: string;
};

type OIDCTokenResponse = {
  id_token?: string;
  access_token?: string;
  token_type?: string;
  expires_in?: number;
};

type JWTHeader = {
  alg?: string;
  kid?: string;
  typ?: string;
};

export type IDTokenClaims = {
  iss: string;
  sub: string;
  aud: string | string[];
  exp: number;
  iat?: number;
  nonce?: string;
  email?: string;
  name?: string;
  preferred_username?: string;
};

export type OIDCTransaction = {
  state: string;
  nonce: string;
  verifier: string;
  returnTo: string;
  issuedAt: number;
};

export type APIWebSession = {
  sessionSecret?: string;
  id: string;
  userProfileId: string;
  actorRef: string;
  displayName: string;
  email?: string;
  issuedAt: string;
  expiresAt: string;
  memberships: APIMembership[];
  locations: APILocation[];
};

export type APIMembership = {
  scope: "organisation" | "location";
  organisationId: string;
  locationId?: string;
  memberRef: string;
  role: string;
};

export type APILocation = {
  id: string;
  organisationId: string;
  name: string;
  status: string;
};

type APIWebSessionResponse = {
  session: APIWebSession;
};

export function randomURLToken(bytes = 32): string {
  return randomBytes(bytes).toString("base64url");
}

export function pkceChallenge(verifier: string): string {
  return createHash("sha256").update(verifier).digest("base64url");
}

export function webBaseURL(): string {
  const value = process.env.SEATD_WEB_BASE_URL?.trim();
  if (value) {
    return value.replace(/\/$/, "");
  }
  return "http://localhost:3000";
}

export function apiBaseURL(): string {
  return process.env.SEATD_API_BASE_URL ?? "http://localhost:8080";
}

export function sessionTTLSeconds(): number {
  return positiveInteger(
    process.env.SEATD_WEB_SESSION_TTL_SECONDS,
    8 * 60 * 60,
  );
}

export function sessionRotateAfterSeconds(): number {
  return positiveInteger(
    process.env.SEATD_WEB_SESSION_ROTATE_AFTER_SECONDS,
    15 * 60,
  );
}

export function oidcScopes(): string {
  return process.env.SEATD_OIDC_SCOPES?.trim() || "openid profile email";
}

export function requireOIDCConfig() {
  const issuer = process.env.SEATD_OIDC_ISSUER?.trim();
  const clientId = process.env.SEATD_OIDC_CLIENT_ID?.trim();
  const clientSecret = process.env.SEATD_OIDC_CLIENT_SECRET?.trim();
  if (!issuer || !clientId || !clientSecret) {
    throw new Error(
      "OIDC configuration is required outside local/test development",
    );
  }
  return { issuer: issuer.replace(/\/$/, ""), clientId, clientSecret };
}

export function requireSessionSecret(): string {
  const secret = process.env.SEATD_WEB_SESSION_SECRET?.trim();
  if (!secret || secret.length < 32) {
    throw new Error("SEATD_WEB_SESSION_SECRET must be at least 32 characters");
  }
  return secret;
}

export function signValue(value: string): string {
  const secret = requireSessionSecret();
  const signature = createHmac("sha256", secret)
    .update(value)
    .digest("base64url");
  return `${value}.${signature}`;
}

export function unsignValue(value: string | undefined): string | null {
  if (!value) {
    return null;
  }
  const index = value.lastIndexOf(".");
  if (index <= 0) {
    return null;
  }
  const payload = value.slice(0, index);
  const actual = Buffer.from(value.slice(index + 1), "base64url");
  const expected = Buffer.from(
    createHmac("sha256", requireSessionSecret())
      .update(payload)
      .digest("base64url"),
    "base64url",
  );
  if (actual.length !== expected.length || !timingSafeEqual(actual, expected)) {
    return null;
  }
  return payload;
}

export function encodeTransaction(transaction: OIDCTransaction): string {
  return signValue(
    Buffer.from(JSON.stringify(transaction), "utf8").toString("base64url"),
  );
}

export function decodeTransaction(
  value: string | undefined,
): OIDCTransaction | null {
  const payload = unsignValue(value);
  if (!payload) {
    return null;
  }
  const transaction = JSON.parse(
    Buffer.from(payload, "base64url").toString("utf8"),
  ) as OIDCTransaction;
  if (
    !transaction.state ||
    !transaction.nonce ||
    !transaction.verifier ||
    Date.now() - transaction.issuedAt > 10 * 60 * 1000
  ) {
    return null;
  }
  return transaction;
}

export async function fetchDiscovery(): Promise<OIDCDiscovery> {
  const { issuer } = requireOIDCConfig();
  const response = await fetch(`${issuer}/.well-known/openid-configuration`, {
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`OIDC discovery failed: ${response.status}`);
  }
  const discovery = (await response.json()) as OIDCDiscovery;
  if (discovery.issuer.replace(/\/$/, "") !== issuer) {
    throw new Error("OIDC discovery issuer mismatch");
  }
  return discovery;
}

export function authorizationURL(
  discovery: OIDCDiscovery,
  transaction: OIDCTransaction,
): URL {
  const { clientId } = requireOIDCConfig();
  const url = new URL(discovery.authorization_endpoint);
  url.searchParams.set("client_id", clientId);
  url.searchParams.set("redirect_uri", `${webBaseURL()}/api/auth/callback`);
  url.searchParams.set("response_type", "code");
  url.searchParams.set("scope", oidcScopes());
  url.searchParams.set("state", transaction.state);
  url.searchParams.set("nonce", transaction.nonce);
  url.searchParams.set("code_challenge", pkceChallenge(transaction.verifier));
  url.searchParams.set("code_challenge_method", "S256");
  return url;
}

export async function exchangeCode(
  discovery: OIDCDiscovery,
  code: string,
  verifier: string,
): Promise<OIDCTokenResponse> {
  const { clientId, clientSecret } = requireOIDCConfig();
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    code,
    redirect_uri: `${webBaseURL()}/api/auth/callback`,
    code_verifier: verifier,
  });
  const response = await fetch(discovery.token_endpoint, {
    method: "POST",
    headers: {
      Authorization: `Basic ${Buffer.from(`${clientId}:${clientSecret}`).toString("base64")}`,
      "Content-Type": "application/x-www-form-urlencoded",
      Accept: "application/json",
    },
    body,
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`OIDC token exchange failed: ${response.status}`);
  }
  return (await response.json()) as OIDCTokenResponse;
}

export async function verifyIDToken(
  discovery: OIDCDiscovery,
  idToken: string,
  nonce: string,
): Promise<IDTokenClaims> {
  const { issuer, clientId } = requireOIDCConfig();
  const [encodedHeader, encodedPayload, encodedSignature] = idToken.split(".");
  if (!encodedHeader || !encodedPayload || !encodedSignature) {
    throw new Error("OIDC id_token is malformed");
  }
  const header = JSON.parse(
    Buffer.from(encodedHeader, "base64url").toString("utf8"),
  ) as JWTHeader;
  if (header.alg !== "RS256" || !header.kid) {
    throw new Error("OIDC id_token must use RS256 with kid");
  }
  const jwksResponse = await fetch(discovery.jwks_uri, { cache: "no-store" });
  if (!jwksResponse.ok) {
    throw new Error(`OIDC JWKS fetch failed: ${jwksResponse.status}`);
  }
  const jwks = (await jwksResponse.json()) as {
    keys?: (NodeJsonWebKey & { kid?: string })[];
  };
  const jwk = jwks.keys?.find((key) => key.kid === header.kid);
  if (!jwk) {
    throw new Error("OIDC signing key not found");
  }
  const verifier = createVerify("RSA-SHA256");
  verifier.update(`${encodedHeader}.${encodedPayload}`);
  verifier.end();
  const valid = verifier.verify(
    createPublicKey({ key: jwk, format: "jwk" }),
    Buffer.from(encodedSignature, "base64url"),
  );
  if (!valid) {
    throw new Error("OIDC id_token signature is invalid");
  }
  const claims = JSON.parse(
    Buffer.from(encodedPayload, "base64url").toString("utf8"),
  ) as IDTokenClaims;
  const audience = Array.isArray(claims.aud) ? claims.aud : [claims.aud];
  if (
    claims.iss.replace(/\/$/, "") !== issuer ||
    !audience.includes(clientId)
  ) {
    throw new Error("OIDC id_token issuer or audience is invalid");
  }
  if (
    claims.exp * 1000 <= Date.now() ||
    claims.nonce !== nonce ||
    !claims.sub
  ) {
    throw new Error("OIDC id_token claims are invalid");
  }
  return claims;
}

export async function createAPISession(
  claims: IDTokenClaims,
): Promise<APIWebSession> {
  const response = await fetch(new URL("/v1/auth/oidc/session", apiBaseURL()), {
    method: "POST",
    headers: internalHeaders(),
    body: JSON.stringify({
      issuer: claims.iss,
      subject: claims.sub,
      email: claims.email ?? "",
      displayName:
        claims.name ?? claims.preferred_username ?? claims.email ?? claims.sub,
      ttlSeconds: sessionTTLSeconds(),
    }),
    cache: "no-store",
  });
  return readAPISession(response);
}

export async function validateAPISession(
  secret: string,
): Promise<APIWebSession | null> {
  const response = await fetch(new URL("/v1/auth/session", apiBaseURL()), {
    headers: { Authorization: `Bearer ${secret}`, Accept: "application/json" },
    cache: "no-store",
  });
  if (response.status === 401) {
    return null;
  }
  return readAPISession(response);
}

export async function rotateAPISession(secret: string): Promise<APIWebSession> {
  const response = await fetch(
    new URL("/v1/auth/session/rotate", apiBaseURL()),
    {
      method: "POST",
      headers: {
        Authorization: `Bearer ${secret}`,
        Accept: "application/json",
      },
      cache: "no-store",
    },
  );
  return readAPISession(response);
}

export async function revokeAPISession(secret: string): Promise<void> {
  await fetch(new URL("/v1/auth/session/logout", apiBaseURL()), {
    method: "POST",
    headers: { Authorization: `Bearer ${secret}`, Accept: "application/json" },
    cache: "no-store",
  });
}

export function internalHeaders(): HeadersInit {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  const secret = process.env.SEATD_INTERNAL_API_SECRET?.trim();
  if (secret) {
    headers["X-Seatd-Internal-Secret"] = secret;
  }
  return headers;
}

function positiveInteger(value: string | undefined, fallback: number): number {
  const parsed = Number.parseInt(value ?? "", 10);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback;
}

async function readAPISession(response: Response): Promise<APIWebSession> {
  if (!response.ok) {
    throw new Error(`Seatd auth API failed: ${response.status}`);
  }
  const body = (await response.json()) as APIWebSessionResponse;
  return body.session;
}

export function constantTimeEqual(a: string, b: string): boolean {
  const left = textEncoder.encode(a);
  const right = textEncoder.encode(b);
  return left.length === right.length && timingSafeEqual(left, right);
}
