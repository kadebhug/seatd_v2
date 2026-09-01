"use client";

import {
  BellIcon,
  ClockIcon,
  DropIcon,
  ForkKnifeIcon,
  ReceiptIcon,
} from "@phosphor-icons/react";
import { useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";
import { DEMO_VENUE, GUEST_ACTIONS } from "../lib/demo-floor";

const ICONS = {
  call_waiter: BellIcon,
  request_bill: ReceiptIcon,
  request_water: DropIcon,
  request_service: ForkKnifeIcon,
};

export function GuestAssistStage() {
  const reduce = useReducedMotion();
  const [elapsed, setElapsed] = useState(36);
  const [pending, setPending] = useState(true);

  useEffect(() => {
    if (reduce) {
      return;
    }
    const tick = window.setInterval(() => {
      setElapsed((value) => (value >= 58 ? 12 : value + 1));
    }, 1000);
    const flip = window.setInterval(() => {
      setPending((value) => !value);
    }, 5200);
    return () => {
      window.clearInterval(tick);
      window.clearInterval(flip);
    };
  }, [reduce]);

  const clock = `00:${String(elapsed).padStart(2, "0")}`;

  return (
    <div className="grid items-center gap-8 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)]">
      <PhoneMock pending={!pending} />
      <WaiterTicket clock={clock} pending={pending} />
    </div>
  );
}

function PhoneMock({ pending }: Readonly<{ pending: boolean }>) {
  return (
    <div className="mx-auto w-full max-w-[340px]">
      <div className="phone-shell">
        <div className="phone-screen">
          <div className="mb-6 flex items-start justify-between gap-3">
            <div className="min-w-0">
              <h3 className="text-[1.35rem] font-semibold tracking-tight">
                <span translate="no">{DEMO_VENUE}</span>
              </h3>
            </div>
            <span className="rounded-full border border-[color-mix(in_srgb,#141411_14%,transparent)] px-2.5 py-1 text-[0.7rem] font-semibold">
              Table 14
            </span>
          </div>
          {pending ? (
            <div className="rounded-[16px] border border-[color-mix(in_srgb,#d99a2b_42%,transparent)] bg-[color-mix(in_srgb,#d99a2b_12%,#ffffff)] p-4">
              <p className="font-semibold">Request sent</p>
              <p className="mt-1 text-[0.92rem] text-[color-mix(in_srgb,#141411_62%,transparent)]">
                Staff can see Table 14 needs the bill.
              </p>
            </div>
          ) : (
            <div className="grid gap-3" aria-label="Guest actions">
              <p className="text-[0.95rem] text-[color-mix(in_srgb,#141411_62%,transparent)]">
                How can we help?
              </p>
              {GUEST_ACTIONS.map((action, index) => {
                const Icon = ICONS[action.key];
                return (
                  <span
                    className={`flex min-h-12 items-center gap-3 rounded-[8px] px-4 py-3 text-[0.95rem] font-semibold ${
                      index === 0
                        ? "bg-signal text-white"
                        : "border border-[color-mix(in_srgb,#141411_14%,transparent)] bg-white"
                    }`}
                    key={action.key}
                  >
                    <Icon aria-hidden="true" size={20} weight="bold" />
                    {action.label}
                  </span>
                );
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function WaiterTicket({
  clock,
  pending,
}: Readonly<{ clock: string; pending: boolean }>) {
  return (
    <div className="ticket-shell grid gap-5">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-sm font-semibold text-muted">Waiter view</p>
          <h3 className="mt-1 text-3xl font-semibold tracking-tight">
            Table 14
          </h3>
        </div>
        <span className="inline-flex items-center gap-2 rounded-full bg-[color-mix(in_srgb,var(--color-attention)_16%,transparent)] px-3 py-1.5 text-sm font-semibold">
          <ClockIcon aria-hidden="true" size={16} weight="bold" />
          <span className="font-mono tabular-nums">{clock}</span>
        </span>
      </div>
      <p className="text-lg">
        Request: <strong>Bill</strong>
      </p>
      <p className="max-w-[42ch] text-muted">
        {pending
          ? "The request is waiting. Staff see exactly which table needs attention."
          : "The guest is about to send a new request from the table QR."}
      </p>
      <div className="flex flex-wrap gap-2">
        <span className="rounded-full border border-line px-3 py-1.5 text-sm font-semibold">
          Main floor
        </span>
        <span className="rounded-full border border-line px-3 py-1.5 text-sm font-semibold">
          Seats 4
        </span>
        <span className="rounded-full bg-[color-mix(in_srgb,var(--color-signal)_12%,transparent)] px-3 py-1.5 text-sm font-semibold text-signal">
          Occupied
        </span>
      </div>
    </div>
  );
}
