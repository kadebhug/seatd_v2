"use client";

import { ArrowRightIcon } from "@phosphor-icons/react";
import { useEffect, useState } from "react";

const beats = [
  {
    href: "/#product",
    moment: "Open",
    line: "Your team shouldn’t have to walk the room to know what’s free.",
    target: "Live floor",
  },
  {
    href: "/#scenarios",
    moment: "Show the room",
    line: "Walk-in, service, and table — three moments, one picture.",
    target: "Service scenarios",
  },
  {
    href: "/#guest-assist",
    moment: "Guest proof",
    line: "No app. Scan the QR, request help, staff see the table.",
    target: "Table QR",
  },
  {
    href: "/#beside",
    moment: "Position",
    line: "POS runs the transaction. Seatd runs the floor.",
    target: "Beside your stack",
  },
  {
    href: "/#demo",
    moment: "Close",
    line: "We’ll map Marlowe’s to your venue on the call.",
    target: "Book a Demo",
  },
] as const;

export function SalesWalkthrough() {
  const [active, setActive] = useState(0);

  useEffect(() => {
    const sections = beats
      .map((beat) => document.querySelector(beat.href.replace("/", "")))
      .filter(Boolean) as HTMLElement[];

    if (sections.length === 0) {
      return;
    }

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            const index = sections.indexOf(entry.target as HTMLElement);
            if (index >= 0) {
              setActive(index);
            }
          }
        }
      },
      { rootMargin: "-20% 0px -55% 0px", threshold: 0.1 },
    );

    for (const section of sections) {
      observer.observe(section);
    }

    return () => observer.disconnect();
  }, []);

  return (
    <div className="call-sheet">
      <p className="call-sheet-label">On a sales call, walk here</p>
      <ol className="call-sheet-beats">
        {beats.map((beat, index) => {
          const current = index === active;
          return (
            <li key={beat.href}>
              <a
                className={`call-sheet-beat focus-ring ${current ? "is-active" : ""}`}
                href={beat.href}
              >
                <span className="call-sheet-moment">{beat.moment}</span>
                <span className="call-sheet-line">{beat.line}</span>
                <span className="call-sheet-target">
                  {beat.target}
                  <ArrowRightIcon aria-hidden="true" size={14} weight="bold" />
                </span>
              </a>
            </li>
          );
        })}
      </ol>
    </div>
  );
}
