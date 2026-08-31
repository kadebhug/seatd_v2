"use client";

import { ArrowRight } from "@phosphor-icons/react";
import { type FormEvent, type ReactNode, useEffect, useState } from "react";

export function MarketingReveal({
  children,
}: Readonly<{ children: ReactNode }>) {
  return <div data-reveal>{children}</div>;
}

export function DemoForm() {
  const [status, setStatus] = useState<"idle" | "sending" | "sent" | "error">(
    "idle",
  );
  const [message, setMessage] = useState("");

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    if (!form.checkValidity()) {
      setStatus("error");
      setMessage("Please complete the required fields before booking a demo.");
      form.reportValidity();
      return;
    }
    setStatus("sending");
    setMessage("");
    window.setTimeout(() => {
      form.reset();
      setStatus("sent");
      setMessage("Thanks. Seatd can follow up with a walkthrough plan.");
    }, 420);
  }

  return (
    <form className="demo-form" noValidate onSubmit={submit}>
      <div className="form-heading">
        <h3>Book a Demo</h3>
        <p>Tell us enough to shape the walkthrough around your venue.</p>
      </div>
      <label>
        Name
        <input autoComplete="name" name="name" required />
      </label>
      <label>
        Restaurant or group name
        <input autoComplete="organization" name="restaurant" required />
      </label>
      <label>
        Email
        <input autoComplete="email" name="email" required type="email" />
      </label>
      <label>
        Phone number
        <input autoComplete="tel" name="phone" type="tel" />
      </label>
      <label>
        Number of tables or locations
        <input inputMode="numeric" name="scale" required />
      </label>
      <button
        className="marketing-button form-button"
        disabled={status === "sending"}
        type="submit"
      >
        <span>{status === "sending" ? "Sending" : "Book a Demo"}</span>
        <span aria-hidden="true" className="marketing-button-icon">
          <ArrowRight size={17} weight="bold" />
        </span>
      </button>
      {message ? (
        <p
          className={status === "error" ? "form-message error" : "form-message"}
          role="status"
        >
          {message}
        </p>
      ) : null}
    </form>
  );
}

export function LandingInteractionLayer() {
  useEffect(() => {
    const elements = Array.from(
      document.querySelectorAll<HTMLElement>("[data-reveal]"),
    );
    if (elements.length === 0) {
      return;
    }
    document.documentElement.dataset.revealReady = "true";

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            entry.target.setAttribute("data-visible", "true");
            observer.unobserve(entry.target);
          }
        }
      },
      { rootMargin: "0px 0px -12% 0px", threshold: 0.18 },
    );

    for (const element of elements) {
      if (element.getBoundingClientRect().top < window.innerHeight * 0.92) {
        element.setAttribute("data-visible", "true");
      } else {
        observer.observe(element);
      }
    }

    return () => {
      observer.disconnect();
      delete document.documentElement.dataset.revealReady;
    };
  }, []);

  return null;
}
