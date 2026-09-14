import type { Metadata, Viewport } from "next";
import type React from "react";
import { dmMono, instrumentSans } from "../lib/fonts";
import "./styles.css";

export const metadata: Metadata = {
  metadataBase: new URL(
    process.env.SEATD_WEB_BASE_URL?.trim() || "http://localhost:3000",
  ),
  title: {
    default: "Seatd",
    template: "%s · Seatd",
  },
  description:
    "Seatd helps restaurant teams see table activity, respond to guest requests, and run the floor with less guesswork.",
  icons: {
    icon: [
      { url: "/favicon.ico" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: [{ url: "/apple-touch-icon.png" }],
  },
  openGraph: {
    title: "Seatd",
    description: "Restaurant floor operations. Every table, in sync.",
    images: [{ url: "/og-brand.png" }],
  },
};

export const viewport: Viewport = {
  themeColor: "#F3EEE4",
  colorScheme: "light",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html className={`${instrumentSans.variable} ${dmMono.variable}`} lang="en">
      <body>
        <a className="skip-link" href="#main">
          Skip to main content
        </a>
        {children}
      </body>
    </html>
  );
}
