import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { redirect } from "next/navigation";
import {
  ArrowRight,
  Buildings,
  ShieldCheck,
} from "@phosphor-icons/react/dist/ssr";
import { safeReturnTo } from "../../lib/auth";
import { getCookieSession, homePath } from "../../lib/session";

type LoginPageProps = {
  searchParams: Promise<{
    error?: string | string[];
    returnTo?: string | string[];
  }>;
};

export const metadata: Metadata = {
  title: "Sign in",
  description: "Sign in to the Seatd owner and platform workspaces.",
};

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const session = await getCookieSession();
  if (session) {
    redirect(homePath(session));
  }

  const params = await searchParams;
  const returnTo = safeReturnTo(firstParam(params.returnTo));
  const hasError = Boolean(firstParam(params.error));
  const loginHref = `/api/auth/login?returnTo=${encodeURIComponent(returnTo)}`;

  return (
    <main className="login-page" id="main">
      <section className="login-shell" aria-labelledby="login-title">
        <Link aria-label="Seatd home" className="login-brand" href="/">
          <Image
            alt=""
            height={38}
            priority
            src="/brand/seatd-lockup-color.svg"
            width={140}
          />
        </Link>

        <div className="login-copy">
          <p className="marketing-eyebrow">Operator access</p>
          <h1 id="login-title">Sign in to Seatd.</h1>
          <p>
            Continue to the owner workspace or platform console with your
            organisation identity.
          </p>
        </div>

        {hasError ? (
          <div className="login-alert" role="alert">
            <strong>Sign-in did not complete</strong>
            <span>
              Your identity provider could not finish the request. Start a new
              sign-in attempt to continue.
            </span>
          </div>
        ) : null}

        <div className="login-action-panel">
          <div className="login-action-icon" aria-hidden="true">
            <ShieldCheck size={26} weight="regular" />
          </div>
          <div>
            <h2>Owner and platform users</h2>
            <p>
              Use your Seatd account to manage floor operations,
              configuration, analytics, and platform support.
            </p>
          </div>
          <a className="marketing-button login-primary" href={loginHref}>
            <span>Sign in</span>
            <span aria-hidden="true" className="marketing-button-icon">
              <ArrowRight size={17} weight="bold" />
            </span>
          </a>
        </div>

        <div className="login-secondary-row">
          <span className="font-mono tabular-nums">{returnTo}</span>
          <Link href="/">
            <Buildings size={17} weight="regular" />
            Public site
          </Link>
        </div>
      </section>
    </main>
  );
}

function firstParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}
