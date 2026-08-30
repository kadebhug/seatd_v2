import React, { useEffect, useMemo, useRef, useState } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";

type Occupancy = {
  status: string;
  currentSessionId?: string;
  version: number;
  updatedAt: string;
};

type GuestAction = {
  key: string;
  label: string;
};

type Assist = {
  id: string;
  tableId: string;
  tableSessionId?: string;
  status: "pending" | "acknowledged" | "resolved" | "cancelled";
  requestedAt: string;
  version: number;
  actionKey?: string;
};

type GuestContext = {
  locationName: string;
  tableLabel: string;
  occupancy: Occupancy;
  actions: GuestAction[];
  activeRequest?: Assist;
};

type LoadState =
  | { kind: "loading" }
  | { kind: "ready"; context: GuestContext; activeRequest?: Assist }
  | { kind: "error"; message: string };

const apiBase = import.meta.env.VITE_SEATD_API_BASE_URL ?? "";

function App() {
  const token = useMemo(readToken, []);
  const [state, setState] = useState<LoadState>({ kind: "loading" });
  const [network, setNetwork] = useState<"online" | "slow" | "offline">(
    navigator.onLine ? "online" : "offline",
  );
  const [submitting, setSubmitting] = useState<string | null>(null);
  const activeRequestRef = useRef<Assist | undefined>(undefined);

  useEffect(() => {
    const online = () => setNetwork("online");
    const offline = () => setNetwork("offline");
    window.addEventListener("online", online);
    window.addEventListener("offline", offline);
    return () => {
      window.removeEventListener("online", online);
      window.removeEventListener("offline", offline);
    };
  }, []);

  useEffect(() => {
    if (!token) {
      setState({
        kind: "error",
        message: "This QR link is incomplete. Ask staff for help.",
      });
      return;
    }

    let cancelled = false;
    void loadGuestContext(token)
      .then((context) => {
        if (cancelled) {
          return;
        }
        activeRequestRef.current = context.activeRequest;
        setState({
          kind: "ready",
          context,
          activeRequest: context.activeRequest,
        });
      })
      .catch(() => {
        if (!cancelled) {
          setState({
            kind: "error",
            message: "This QR code is not available right now.",
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  useEffect(() => {
    if (!token || !activeRequestRef.current) {
      return;
    }
    let cancelled = false;
    let timer: number | undefined;
    let interval = 3000;

    const poll = async () => {
      const current = activeRequestRef.current;
      if (
        !current ||
        current.status === "resolved" ||
        current.status === "cancelled"
      ) {
        return;
      }
      try {
        const next = await fetchGuestRequest(token, current.id);
        if (cancelled) {
          return;
        }
        activeRequestRef.current = next;
        setNetwork("online");
        setState((previous) =>
          previous.kind === "ready"
            ? { ...previous, activeRequest: next }
            : previous,
        );
        interval = next.status === "pending" ? 3000 : 5000;
      } catch {
        if (!cancelled) {
          setNetwork(navigator.onLine ? "slow" : "offline");
          interval = Math.min(interval * 2, 15000);
        }
      } finally {
        if (!cancelled) {
          timer = window.setTimeout(poll, interval);
        }
      }
    };

    timer = window.setTimeout(poll, interval);
    return () => {
      cancelled = true;
      if (timer) {
        window.clearTimeout(timer);
      }
    };
  }, [token, state.kind === "ready" ? state.activeRequest?.id : undefined]);

  const requestAction = async (action: GuestAction) => {
    if (!token || state.kind !== "ready") {
      return;
    }
    setSubmitting(action.key);
    try {
      const assist = await createGuestRequest(token, action.key);
      activeRequestRef.current = assist;
      setState({ ...state, activeRequest: assist });
      setNetwork("online");
    } catch (error) {
      setState({
        ...state,
        activeRequest: undefined,
      });
      window.alert(error instanceof Error ? error.message : "Request failed");
    } finally {
      setSubmitting(null);
    }
  };

  const cancelRequest = async () => {
    if (!token || state.kind !== "ready" || !state.activeRequest) {
      return;
    }
    setSubmitting("cancel");
    try {
      const assist = await cancelGuestRequest(token, state.activeRequest.id);
      activeRequestRef.current = assist;
      setState({ ...state, activeRequest: assist });
    } catch (error) {
      window.alert(error instanceof Error ? error.message : "Cancel failed");
    } finally {
      setSubmitting(null);
    }
  };

  if (state.kind === "loading") {
    return (
      <Shell network={network} title="Seatd">
        <StatusPanel tone="neutral" title="Loading table" />
      </Shell>
    );
  }

  if (state.kind === "error") {
    return (
      <Shell network={network} title="Seatd">
        <StatusPanel tone="danger" title={state.message} />
      </Shell>
    );
  }

  const active = state.activeRequest;
  const canRequest = state.context.occupancy.status === "occupied" && !active;

  return (
    <Shell network={network} title={state.context.locationName}>
      <section className="table">
        <span>Table</span>
        <strong>{state.context.tableLabel}</strong>
      </section>

      {active ? (
        <RequestStatus
          assist={active}
          onCancel={cancelRequest}
          busy={submitting === "cancel"}
        />
      ) : canRequest ? (
        <section className="actions" aria-label="Guest actions">
          {state.context.actions.map((action) => (
            <button
              className="action"
              disabled={submitting !== null}
              key={action.key}
              onClick={() => void requestAction(action)}
              type="button"
            >
              {submitting === action.key ? "Sending..." : action.label}
            </button>
          ))}
        </section>
      ) : (
        <StatusPanel
          tone="neutral"
          title="Service is available once staff opens the table."
        />
      )}
    </Shell>
  );
}

function Shell({
  children,
  network,
  title,
}: {
  children: React.ReactNode;
  network: "online" | "slow" | "offline";
  title: string;
}) {
  return (
    <main>
      <header>
        <strong>{title}</strong>
        <span className={`network ${network}`}>{network}</span>
      </header>
      {children}
    </main>
  );
}

function StatusPanel({
  title,
  tone,
}: {
  title: string;
  tone: "neutral" | "success" | "danger";
}) {
  return (
    <section className={`status ${tone}`} role="status">
      <strong>{title}</strong>
    </section>
  );
}

function RequestStatus({
  assist,
  busy,
  onCancel,
}: {
  assist: Assist;
  busy: boolean;
  onCancel: () => void;
}) {
  const steps = ["pending", "acknowledged", "resolved"];
  const activeIndex = steps.indexOf(assist.status);
  return (
    <section className="request" aria-live="polite">
      <StatusPanel
        tone={assist.status === "resolved" ? "success" : "neutral"}
        title={statusText(assist.status)}
      />
      <ol className="steps">
        {steps.map((step, index) => (
          <li className={index <= activeIndex ? "complete" : ""} key={step}>
            {step}
          </li>
        ))}
      </ol>
      {assist.status === "pending" ? (
        <button
          className="secondary"
          disabled={busy}
          onClick={onCancel}
          type="button"
        >
          {busy ? "Cancelling..." : "Cancel request"}
        </button>
      ) : null}
    </section>
  );
}

function statusText(status: Assist["status"]) {
  switch (status) {
    case "pending":
      return "Request sent";
    case "acknowledged":
      return "Staff acknowledged";
    case "resolved":
      return "Request resolved";
    case "cancelled":
      return "Request cancelled";
  }
}

async function loadGuestContext(token: string): Promise<GuestContext> {
  return requestJSON<GuestContext>(`/v1/guest/qr/${encodeURIComponent(token)}`);
}

async function createGuestRequest(
  token: string,
  actionKey: string,
): Promise<Assist> {
  return requestJSON<Assist>(
    `/v1/guest/qr/${encodeURIComponent(token)}/requests`,
    {
      method: "POST",
      body: JSON.stringify({
        commandId: crypto.randomUUID(),
        actionKey,
      }),
    },
  );
}

async function fetchGuestRequest(
  token: string,
  assistID: string,
): Promise<Assist> {
  return requestJSON<Assist>(
    `/v1/guest/qr/${encodeURIComponent(token)}/requests/${encodeURIComponent(assistID)}`,
  );
}

async function cancelGuestRequest(
  token: string,
  assistID: string,
): Promise<Assist> {
  return requestJSON<Assist>(
    `/v1/guest/qr/${encodeURIComponent(token)}/requests/${encodeURIComponent(assistID)}/cancel`,
    {
      method: "POST",
      body: JSON.stringify({ commandId: crypto.randomUUID() }),
    },
  );
}

async function requestJSON<T>(
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(`${apiBase}${path}`, {
    headers: { "Content-Type": "application/json", ...init.headers },
    ...init,
  });
  if (!response.ok) {
    const message = await readErrorMessage(response);
    throw new Error(message);
  }
  return response.json() as Promise<T>;
}

async function readErrorMessage(response: Response) {
  try {
    const body = (await response.json()) as { error?: { message?: string } };
    return body.error?.message ?? "Request failed";
  } catch {
    return "Request failed";
  }
}

function readToken() {
  const pathMatch = window.location.pathname.match(/\/qr\/([^/]+)/);
  if (pathMatch?.[1]) {
    return decodeURIComponent(pathMatch[1]);
  }
  return new URLSearchParams(window.location.search).get("token") ?? "";
}

createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
