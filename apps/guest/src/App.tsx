import {
  BellIcon,
  CheckCircleIcon,
  ClockIcon,
  DropIcon,
  ForkKnifeIcon,
  ReceiptIcon,
  WarningCircleIcon,
  WifiHighIcon,
  WifiMediumIcon,
  WifiSlashIcon,
} from "@phosphor-icons/react";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";

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
  | { kind: "error"; message: string; detail: string };

type NetworkState = "online" | "slow" | "offline";

const apiBase = import.meta.env.VITE_SEATD_API_BASE_URL ?? "";

const requestSteps = [
  { key: "pending", label: "Sent" },
  { key: "acknowledged", label: "Seen by staff" },
  { key: "resolved", label: "Done" },
] as const;

export function App() {
  const token = useMemo(readToken, []);
  const [state, setState] = useState<LoadState>({ kind: "loading" });
  const [network, setNetwork] = useState<NetworkState>(
    navigator.onLine ? "online" : "offline",
  );
  const [submitting, setSubmitting] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [confirmCancel, setConfirmCancel] = useState(false);
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
        message: "This QR link is incomplete.",
        detail: "Ask staff for a new table code.",
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
            message: "This table code is not available right now.",
            detail: "Ask staff for help, or try scanning again.",
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
    setActionError(null);
    setConfirmCancel(false);
    try {
      const assist = await createGuestRequest(token, action.key);
      activeRequestRef.current = assist;
      setState({ ...state, activeRequest: assist });
      setNetwork("online");
    } catch (error) {
      setActionError(
        error instanceof Error
          ? nextStepMessage(error.message)
          : "Could not send that request. Try again in a moment.",
      );
    } finally {
      setSubmitting(null);
    }
  };

  const cancelRequest = async () => {
    if (!token || state.kind !== "ready" || !state.activeRequest) {
      return;
    }
    if (!confirmCancel) {
      setConfirmCancel(true);
      return;
    }
    setSubmitting("cancel");
    setActionError(null);
    try {
      const assist = await cancelGuestRequest(token, state.activeRequest.id);
      activeRequestRef.current = assist;
      setState({ ...state, activeRequest: assist });
      setConfirmCancel(false);
    } catch (error) {
      setActionError(
        error instanceof Error
          ? nextStepMessage(error.message)
          : "Could not cancel that request. Try again in a moment.",
      );
    } finally {
      setSubmitting(null);
    }
  };

  if (state.kind === "loading") {
    return (
      <Shell network={network} branded>
        <section aria-busy="true" aria-live="polite" className="splash">
          <img
            alt=""
            aria-hidden="true"
            className="brand-symbol"
            height={48}
            src="/brand/seatd-symbol-color.svg"
            width={48}
          />
          <h1 className="splash-title" translate="no">
            Seatd
          </h1>
          <p className="role-label">Guest assist</p>
          <p className="visually-hidden">Loading table…</p>
          <div className="skeleton">
            <div className="skeleton-line title" />
            <div className="skeleton-line table" />
            <div className="skeleton-line action" />
            <div className="skeleton-line action" />
          </div>
        </section>
      </Shell>
    );
  }

  if (state.kind === "error") {
    return (
      <Shell network={network} branded>
        <section className="splash">
          <img
            alt=""
            aria-hidden="true"
            className="brand-symbol"
            height={48}
            src="/brand/seatd-symbol-color.svg"
            width={48}
          />
          <h1 className="splash-title" translate="no">
            Seatd
          </h1>
          <p className="role-label">Guest assist</p>
          <StatusPanel
            detail={state.detail}
            title={state.message}
            tone="danger"
          />
        </section>
      </Shell>
    );
  }

  const active = state.activeRequest;
  const openRequest =
    active && active.status !== "resolved" && active.status !== "cancelled";
  const canRequest =
    state.context.occupancy.status === "occupied" && !openRequest;

  return (
    <Shell locationName={state.context.locationName} network={network}>
      <section className="location">
        <h1>{state.context.locationName}</h1>
      </section>

      <TableBlock
        occupancy={state.context.occupancy.status}
        tableLabel={state.context.tableLabel}
      />

      {openRequest ? (
        <RequestStatus
          actionLabel={actionLabelFor(state.context.actions, active.actionKey)}
          assist={active}
          busy={submitting === "cancel"}
          confirmCancel={confirmCancel}
          onCancel={() => void cancelRequest()}
        />
      ) : canRequest ? (
        <section className="actions" aria-label="Guest actions">
          {active?.status === "resolved" ? (
            <StatusPanel
              detail="You can send another if you need anything else."
              title="This request is complete."
              tone="success"
            />
          ) : null}
          {active?.status === "cancelled" ? (
            <StatusPanel
              detail="You can send a new request when you are ready."
              title="Request cancelled."
              tone="neutral"
            />
          ) : null}
          {state.context.actions.map((action, index) => {
            const Icon = actionIcon(action.key);
            const sending = submitting === action.key;
            return (
              <button
                aria-busy={sending}
                className={index === 0 ? "action" : "action secondary"}
                disabled={submitting !== null}
                key={action.key}
                onClick={() => void requestAction(action)}
                type="button"
              >
                <Icon aria-hidden="true" size={22} weight="bold" />
                {sending ? "Sending…" : action.label}
              </button>
            );
          })}
        </section>
      ) : (
        <StatusPanel
          detail="Ask staff when you are seated. Requests open once the table is occupied."
          title="This table is available."
          tone="neutral"
        />
      )}
      {actionError ? (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      ) : null}
    </Shell>
  );
}

function Shell({
  branded = false,
  children,
  locationName,
  network,
}: {
  branded?: boolean;
  children: ReactNode;
  locationName?: string;
  network: NetworkState;
}) {
  return (
    <>
      <a className="skip-link" href="#main">
        Skip to main content
      </a>
      <div className="app">
        <header className={branded ? "masthead branded" : "masthead"}>
          {branded ? (
            <span className="visually-hidden">Seatd guest assist</span>
          ) : (
            <div className="brand">
              <img
                alt=""
                aria-hidden="true"
                className="brand-symbol"
                height={32}
                src="/brand/seatd-symbol-color.svg"
                width={32}
              />
              <span className="brand-copy">
                <span className="brand-name" translate="no">
                  Seatd
                </span>
                <span className="role-label">Guest</span>
              </span>
            </div>
          )}
          <NetworkBadge network={network} />
        </header>
        <main className="content" id="main">
          {children}
        </main>
        <p className="footnote">
          {locationName ? "Service by " : "Every table, in sync. "}
          <span translate="no">Seatd</span>
        </p>
      </div>
    </>
  );
}

function NetworkBadge({ network }: { network: NetworkState }) {
  const Icon =
    network === "offline"
      ? WifiSlashIcon
      : network === "slow"
        ? WifiMediumIcon
        : WifiHighIcon;
  const label =
    network === "offline"
      ? "Offline"
      : network === "slow"
        ? "Slow connection"
        : "Connected";

  return (
    <span aria-label={`Connection: ${label}`} className={`network ${network}`}>
      <Icon aria-hidden="true" size={14} weight="bold" />
      {label}
    </span>
  );
}

function TableBlock({
  occupancy,
  tableLabel,
}: {
  occupancy: string;
  tableLabel: string;
}) {
  const variant = occupancyVariant(occupancy);
  const Icon =
    variant === "occupied"
      ? ClockIcon
      : variant === "attention"
        ? WarningCircleIcon
        : CheckCircleIcon;

  return (
    <section className="table-block">
      <p>Table</p>
      <div className="table-row">
        <strong className="table-label font-mono tabular-nums">
          {tableLabel}
        </strong>
        <span className={`status-badge status-${variant}`}>
          <Icon aria-hidden="true" size={14} weight="bold" />
          {occupancyLabel(occupancy)}
        </span>
      </div>
    </section>
  );
}

function StatusPanel({
  detail,
  title,
  tone,
}: {
  detail?: string;
  title: string;
  tone: "neutral" | "pending" | "success" | "danger";
}) {
  return (
    <section className={`status ${tone}`} role="status">
      <strong>{title}</strong>
      {detail ? <p>{detail}</p> : null}
    </section>
  );
}

function RequestStatus({
  actionLabel,
  assist,
  busy,
  confirmCancel,
  onCancel,
}: {
  actionLabel?: string;
  assist: Assist;
  busy: boolean;
  confirmCancel: boolean;
  onCancel: () => void;
}) {
  const activeIndex = requestSteps.findIndex(
    (step) => step.key === assist.status,
  );
  const copy = statusCopy(assist.status, actionLabel);

  return (
    <section className="request" aria-live="polite">
      <StatusPanel detail={copy.detail} title={copy.title} tone={copy.tone} />
      {assist.status === "cancelled" ? null : (
        <ol className="steps">
          {requestSteps.map((step, index) => {
            const complete = activeIndex >= 0 && index <= activeIndex;
            const current =
              index === activeIndex && assist.status !== "resolved";
            return (
              <li
                className={current ? "current" : complete ? "complete" : ""}
                key={step.key}
              >
                <span className="step-mark">
                  {complete && !current ? (
                    <CheckCircleIcon
                      aria-hidden="true"
                      size={16}
                      weight="bold"
                    />
                  ) : (
                    <ClockIcon aria-hidden="true" size={16} weight="bold" />
                  )}
                </span>
                <span className="step-copy">
                  {step.label}
                  {index === 0 ? (
                    <span>{formatRequestedAt(assist.requestedAt)}</span>
                  ) : null}
                </span>
              </li>
            );
          })}
        </ol>
      )}
      {assist.status === "pending" ? (
        <button
          aria-busy={busy}
          className="action secondary"
          disabled={busy}
          onClick={onCancel}
          type="button"
        >
          {busy
            ? "Cancelling…"
            : confirmCancel
              ? "Confirm cancel"
              : "Cancel request"}
        </button>
      ) : null}
    </section>
  );
}

function statusCopy(
  status: Assist["status"],
  actionLabel?: string,
): {
  title: string;
  detail: string;
  tone: "neutral" | "pending" | "success" | "danger";
} {
  switch (status) {
    case "pending":
      return {
        title: actionLabel ? `${actionLabel} sent.` : "Request sent.",
        detail: "Staff will be with you shortly.",
        tone: "pending",
      };
    case "acknowledged":
      return {
        title: "Staff have seen your request.",
        detail: "Someone is on the way.",
        tone: "pending",
      };
    case "resolved":
      return {
        title: "This request is complete.",
        detail: "You can send another if you need anything else.",
        tone: "success",
      };
    case "cancelled":
      return {
        title: "Request cancelled.",
        detail: "You can send a new request when you are ready.",
        tone: "neutral",
      };
  }
}

function occupancyLabel(status: string) {
  switch (status) {
    case "available":
      return "Available";
    case "occupied":
      return "Occupied";
    case "attention":
      return "Attention";
    default:
      return status
        ? status.charAt(0).toUpperCase() + status.slice(1)
        : "Unknown";
  }
}

function occupancyVariant(status: string) {
  if (
    status === "occupied" ||
    status === "attention" ||
    status === "available"
  ) {
    return status;
  }
  return "available";
}

function actionLabelFor(actions: GuestAction[], key?: string) {
  return actions.find((action) => action.key === key)?.label;
}

function actionIcon(key: string) {
  switch (key) {
    case "call_waiter":
      return BellIcon;
    case "request_bill":
      return ReceiptIcon;
    case "request_water":
      return DropIcon;
    default:
      return ForkKnifeIcon;
  }
}

function formatRequestedAt(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "";
  }
  return new Intl.DateTimeFormat(undefined, {
    hour: "numeric",
    minute: "2-digit",
  }).format(date);
}

function nextStepMessage(message: string) {
  const trimmed = message.trim();
  if (!trimmed) {
    return "Could not complete that action. Try again in a moment.";
  }
  if (/try again|ask staff|moment/i.test(trimmed)) {
    return trimmed;
  }
  return `${trimmed} Try again in a moment.`;
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
