import type { Metadata, Viewport } from "next";
import type React from "react";
import { archivo, sourceSerif } from "../lib/fonts";
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
  themeColor: "#E3DDD6",
  colorScheme: "light",
};

const contract = `THESIS: Seatd as the laminated host-stand chart. The first viewport is a split desk: the offer on cream paper, the live floor clipped to timber. It refuses a SaaS dashboard floating in a device.
OWN-WORLD: Desk cream #E3DDD6, ink #141411, clip coral #E54B2D, jade and amber as dry-erase fills. Source Serif 4 for display, Archivo for the sheet. Laminate paper, metal clip, walnut timber. Controls are chart stamps (10px radius), not pills.
STORY: An owner recognizes the floor problem, sees Marlowe's live, believes Seatd sits beside POS, and books a demo. Sales can walk the chart.
FIRST VIEWPORT: Flat header with lockup left, nav center, Book a Demo right. Left: serif headline with a coral period, short subtext, filled and outlined stamps. Right: timber field, wooden clipboard, live Marlowe's chart with a legend. Primary action sits in the header and under the headline.
FORM: The Host Chart, grounded pick, seed 1ed23189. Composition: Clipboard on the desk.
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance`;

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html
      className={`${archivo.variable} ${sourceSerif.variable}`}
      lang="en"
    >
      <body>
        <span
          aria-hidden="true"
          className="visually-hidden"
          dangerouslySetInnerHTML={{ __html: `<!--\n${contract}\n-->` }}
        />
        <a className="skip-link" href="#main">
          Skip to main content
        </a>
        {children}
      </body>
    </html>
  );
}
