import type { Metadata, Viewport } from "next";
import type React from "react";
import { dmMono, instrumentSans } from "../lib/fonts";
import "./globals.css";

export const metadata: Metadata = {
  metadataBase: new URL("https://seatd.app"),
  title: {
    default: "Seatd · Know what's happening on your floor",
    template: "%s · Seatd",
  },
  description:
    "Seatd is restaurant floor operations software. See table status live, let guests request help from the table, and run service with less guesswork.",
  icons: {
    icon: [
      { url: "/favicon.ico" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: [{ url: "/apple-touch-icon.png" }],
  },
  openGraph: {
    title: "Seatd · Know what's happening on your floor",
    description:
      "Live floor visibility and table QR guest assistance for busy restaurants.",
    images: [{ url: "/og-brand.png" }],
    type: "website",
  },
};

export const viewport: Viewport = {
  themeColor: [
    { media: "(prefers-color-scheme: light)", color: "#F3EEE4" },
    { media: "(prefers-color-scheme: dark)", color: "#141411" },
  ],
  colorScheme: "light dark",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      className={`${instrumentSans.variable} ${dmMono.variable}`}
      lang="en"
    >
      <body>
        <a className="skip-link" href="#main">
          Skip to main content
        </a>
        {children}
      </body>
    </html>
  );
}
