"use client";

import { useEffect, useState } from "react";
import { BookDemoButton } from "./book-demo-button";

export function StickyDemoBar() {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    function onScroll() {
      setVisible(window.scrollY > window.innerHeight * 0.72);
    }
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  return (
    <div
      aria-hidden={!visible}
      className={`sticky-demo-bar ${visible ? "is-visible" : ""}`}
    >
      <div className="shell flex flex-wrap items-center justify-between gap-3 py-3">
        <p className="max-w-[36ch] text-sm font-semibold">
          See Seatd on your floor — live chart, table QR, one operational view.
        </p>
        <BookDemoButton className="shrink-0" source="sticky-bar" />
      </div>
    </div>
  );
}
