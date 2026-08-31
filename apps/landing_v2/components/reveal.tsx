"use client";

import { motion, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

export function Reveal({
  children,
  className = "",
  delay = 0,
}: Readonly<{ children: ReactNode; className?: string; delay?: number }>) {
  const reduce = useReducedMotion();
  return (
    <motion.div
      className={className}
      initial={reduce ? false : { opacity: 0, y: 24 }}
      transition={{
        delay,
        duration: 0.7,
        ease: [0.32, 0.72, 0, 1],
      }}
      viewport={{ amount: 0.24, once: true }}
      whileInView={{ opacity: 1, y: 0 }}
    >
      {children}
    </motion.div>
  );
}
