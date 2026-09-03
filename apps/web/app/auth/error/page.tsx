import Link from "next/link";
import { ArrowRight, WarningCircle } from "@phosphor-icons/react/dist/ssr";
import { safeReturnTo } from "../../../lib/auth";

type AuthErrorPageProps = {
  searchParams: Promise<{
    returnTo?: string | string[];
  }>;
};

export default async function AuthErrorPage({
  searchParams,
}: AuthErrorPageProps) {
  const params = await searchParams;
  const returnTo = safeReturnTo(firstParam(params.returnTo));
  const loginHref = `/api/auth/login?returnTo=${encodeURIComponent(returnTo)}`;

  return (
    <main className="login-page" id="main">
      <section className="login-shell" aria-labelledby="auth-error-title">
        <Link aria-label="Seatd home" className="login-brand" href="/">
          Seatd
        </Link>
        <div className="login-alert login-alert-large" role="alert">
          <WarningCircle size={24} weight="regular" />
          <div>
            <strong id="auth-error-title">Sign-in failed</strong>
            <span>
              Your identity provider did not complete the sign-in flow. Start a
              fresh attempt to continue.
            </span>
          </div>
        </div>
        <a className="marketing-button login-primary" href={loginHref}>
          <span>Try again</span>
          <span aria-hidden="true" className="marketing-button-icon">
            <ArrowRight size={17} weight="bold" />
          </span>
        </a>
        <Link className="login-muted-link" href="/login">
          Back to sign in
        </Link>
      </section>
    </main>
  );
}

function firstParam(value: string | string[] | undefined): string | undefined {
  return Array.isArray(value) ? value[0] : value;
}
