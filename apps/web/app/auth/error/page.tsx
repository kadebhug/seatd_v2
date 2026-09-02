import Link from "next/link";

export default function AuthErrorPage() {
  return (
    <main className="page" id="main">
      <section className="page-heading">
        <p className="eyebrow">Authentication</p>
        <h1>Sign-in failed</h1>
      </section>
      <section className="panel">
        <p>Your identity provider did not complete the sign-in flow.</p>
        <Link href="/api/auth/login?returnTo=/owner">Try again</Link>
      </section>
    </main>
  );
}
