"use client";

import { ArrowUpRightIcon } from "@phosphor-icons/react";

type CtaProps = Readonly<{
  href?: string;
  source: string;
  className?: string;
}>;

export function BookDemoButton({
  href = "/#demo",
  source,
  className = "inline-flex",
}: CtaProps) {
  return (
    <a
      className={`cta-primary ${className}`}
      data-cta="book-demo"
      data-cta-source={source}
      href={href}
    >
      Book a Demo
      <span className="cta-icon">
        <ArrowUpRightIcon aria-hidden="true" size={16} weight="bold" />
      </span>
    </a>
  );
}
