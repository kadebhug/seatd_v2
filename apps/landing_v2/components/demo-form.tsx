"use client";

import {
  useEffect,
  useRef,
  useState,
  type FormEvent,
  type HTMLAttributes,
} from "react";

type FieldErrors = Partial<
  Record<"name" | "restaurant" | "email" | "tables", string>
>;

type FormState =
  | { kind: "idle" }
  | { kind: "submitting" }
  | { kind: "error"; message: string; fields: FieldErrors }
  | { kind: "success" };

const initial = {
  name: "",
  restaurant: "",
  email: "",
  phone: "",
  tables: "",
};

export function DemoForm() {
  const [values, setValues] = useState(initial);
  const [state, setState] = useState<FormState>({ kind: "idle" });
  const [dirty, setDirty] = useState(false);
  const firstError = useRef<HTMLInputElement | HTMLSelectElement | null>(null);

  useEffect(() => {
    function warn(event: BeforeUnloadEvent) {
      if (dirty && state.kind !== "success" && state.kind !== "submitting") {
        event.preventDefault();
      }
    }
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty, state.kind]);

  useEffect(() => {
    if (state.kind === "error") {
      firstError.current?.focus();
    }
  }, [state]);

  async function onSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setState({ kind: "submitting" });
    try {
      const response = await fetch("/api/demo", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
      const payload = (await response.json()) as {
        ok?: boolean;
        message?: string;
        fields?: FieldErrors;
      };
      if (!response.ok) {
        setState({
          kind: "error",
          message: payload.message ?? "Check the highlighted fields and try again.",
          fields: payload.fields ?? {},
        });
        return;
      }
      setDirty(false);
      setState({ kind: "success" });
    } catch {
      setState({
        kind: "error",
        message: "Could not send that just now. Try again in a moment.",
        fields: {},
      });
    }
  }

  const fields = state.kind === "error" ? state.fields : {};

  if (state.kind === "success") {
    return (
      <div
        aria-live="polite"
        className="rounded-[24px] border border-line bg-surface p-8"
        role="status"
      >
        <h3 className="text-2xl font-semibold tracking-tight">
          Demo request received
        </h3>
        <p className="mt-3 max-w-[48ch] text-muted">
          We’ll use your venue details to prepare a walkthrough of the owner,
          waiter, display, and guest experience.
        </p>
      </div>
    );
  }

  return (
    <form
      className="grid gap-5 rounded-[24px] border border-line bg-surface p-6 md:p-8"
      noValidate
      onSubmit={(event) => void onSubmit(event)}
    >
      <div className="grid gap-5 md:grid-cols-2">
        <Field
          autoComplete="name"
          error={fields.name}
          inputRef={(node) => {
            if (fields.name) firstError.current = node;
          }}
          label="Name"
          name="name"
          onChange={(value) => {
            setDirty(true);
            setValues((current) => ({ ...current, name: value }));
          }}
          placeholder="Alex Moyo…"
          value={values.name}
        />
        <Field
          autoComplete="organization"
          error={fields.restaurant}
          inputRef={(node) => {
            if (!fields.name && fields.restaurant) firstError.current = node;
          }}
          label="Restaurant / group name"
          name="restaurant"
          onChange={(value) => {
            setDirty(true);
            setValues((current) => ({ ...current, restaurant: value }));
          }}
          placeholder="Marlowe’s…"
          value={values.restaurant}
        />
      </div>
      <div className="grid gap-5 md:grid-cols-2">
        <Field
          autoComplete="email"
          error={fields.email}
          inputMode="email"
          inputRef={(node) => {
            if (!fields.name && !fields.restaurant && fields.email) {
              firstError.current = node;
            }
          }}
          label="Email"
          name="email"
          onChange={(value) => {
            setDirty(true);
            setValues((current) => ({ ...current, email: value }));
          }}
          placeholder="you@venue.co.za…"
          spellCheck={false}
          type="email"
          value={values.email}
        />
        <Field
          autoComplete="tel"
          inputMode="tel"
          label="Phone (optional)"
          name="phone"
          onChange={(value) => {
            setDirty(true);
            setValues((current) => ({ ...current, phone: value }));
          }}
          placeholder="+27 82 000 0000…"
          type="tel"
          value={values.phone}
        />
      </div>
      <label className="form-field">
        <span>Number of tables or locations</span>
        <select
          autoComplete="off"
          name="tables"
          onChange={(event) => {
            setDirty(true);
            setValues((current) => ({ ...current, tables: event.target.value }));
          }}
          ref={(node) => {
            if (
              !fields.name &&
              !fields.restaurant &&
              !fields.email &&
              fields.tables
            ) {
              firstError.current = node;
            }
          }}
          required
          value={values.tables}
        >
          <option value="">Select a range…</option>
          <option value="under-25">Under 25 tables</option>
          <option value="25-50">25-50 tables</option>
          <option value="50-plus">50+ tables</option>
          <option value="multi-location">Multiple locations</option>
        </select>
        {fields.tables ? <span className="error">{fields.tables}</span> : null}
      </label>
      {state.kind === "error" ? (
        <p aria-live="polite" className="error" role="alert">
          {state.message}
        </p>
      ) : null}
      <button
        aria-busy={state.kind === "submitting"}
        className="cta-primary inline-flex w-fit"
        disabled={state.kind === "submitting"}
        type="submit"
      >
        {state.kind === "submitting" ? "Sending…" : "Book a Demo"}
        <span className="cta-icon" aria-hidden="true">
          →
        </span>
      </button>
    </form>
  );
}

function Field({
  autoComplete,
  error,
  inputMode,
  inputRef,
  label,
  name,
  onChange,
  placeholder,
  spellCheck,
  type = "text",
  value,
}: Readonly<{
  autoComplete: string;
  error?: string;
  inputMode?: HTMLAttributes<HTMLInputElement>["inputMode"];
  inputRef?: (node: HTMLInputElement | null) => void;
  label: string;
  name: string;
  onChange: (value: string) => void;
  placeholder: string;
  spellCheck?: boolean;
  type?: string;
  value: string;
}>) {
  const id = `demo-${name}`;
  return (
    <div className="form-field">
      <label htmlFor={id}>{label}</label>
      <input
        autoComplete={autoComplete}
        id={id}
        inputMode={inputMode}
        name={name}
        onChange={(event) => onChange(event.target.value)}
        placeholder={placeholder}
        ref={inputRef}
        required={name !== "phone"}
        spellCheck={spellCheck}
        type={type}
        value={value}
      />
      {error ? (
        <span className="error" id={`${id}-error`}>
          {error}
        </span>
      ) : null}
    </div>
  );
}
