import type { Metadata } from "next";
import type React from "react";
import "./styles.css";

export const metadata: Metadata = {
  title: "Seatd",
  description: "Seatd owner and platform web skeleton",
};

export default function RootLayout({
  children,
}: Readonly<{ children: React.ReactNode }>) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
