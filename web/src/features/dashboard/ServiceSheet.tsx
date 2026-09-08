import { useEffect, useRef, useState } from "react";
import type { ReactNode } from "react";
import type { Service, ServiceAction, ServicePort } from "./api";

type RunAction = (input: { id: string; action: ServiceAction; port?: number }) => Promise<unknown>;

export function ServiceSheet({
  service,
  close,
  runAction,
  actionError,
  pendingAction,
}: {
  service?: Service;
  close: () => void;
  runAction: RunAction;
  actionError?: string;
  pendingAction?: ServiceAction;
}) {
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const sheetRef = useRef<HTMLElement>(null);
  const [confirmEnd, setConfirmEnd] = useState(false);

  useEffect(() => {
    if (!service) {
      return;
    }

    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        close();
      }
      if (event.key === "Tab") {
        const buttons = [...(sheetRef.current?.querySelectorAll<HTMLButtonElement>("button:not(:disabled)") ?? [])];
        if (buttons.length === 0) {
          return;
        }
        const current = buttons.indexOf(document.activeElement as HTMLButtonElement);
        const next = event.shiftKey
          ? buttons[(current - 1 + buttons.length) % buttons.length]
          : buttons[(current + 1) % buttons.length];
        event.preventDefault();
        next.focus();
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    closeButtonRef.current?.focus();
    return () => {
      window.removeEventListener("keydown", handleKeyDown);
      previousFocus?.focus();
    };
  }, [service?.id, close]);

  useEffect(() => {
    setConfirmEnd(false);
  }, [service?.id]);

  if (!service) {
    return null;
  }

  const unhealthy = service.health === "unhealthy" || service.state === "exited";
  const httpPorts = service.ports.filter((port) => port.host && port.scheme === "http");
  const actionPending = pendingAction !== undefined;

  return (
    <div
      className="fixed inset-0 z-50 flex animate-[sheet-backdrop-in_160ms_ease-out] items-end justify-center bg-black/65 motion-reduce:animate-none"
      onClick={(event) => {
        if (!actionPending && event.currentTarget === event.target) {
          close();
        }
      }}
      role="presentation"
    >
      <section
        className="max-h-[min(82dvh,720px)] w-full max-w-[560px] animate-[sheet-in_200ms_var(--ease-standard)] overflow-y-auto overscroll-contain rounded-t-2xl border border-b-0 border-border bg-surface px-5 pt-2 pb-[max(24px,env(safe-area-inset-bottom))] shadow-2xl motion-reduce:animate-none sm:mb-5 sm:rounded-2xl sm:border sm:pb-6"
        aria-labelledby="service-sheet-title"
        aria-modal="true"
        ref={sheetRef}
        role="dialog"
      >
        <div className="mx-auto mb-3 h-1 w-9 rounded-full bg-quiet sm:hidden" />
        <header className="flex items-start gap-4 py-2">
          <div className="min-w-0 flex-1">
            <div className="mb-1 flex items-center gap-2">
              <span
                className={`h-1.5 w-1.5 shrink-0 rounded-full ${unhealthy ? "bg-danger" : "bg-accent"}`}
                aria-hidden="true"
              />
              <span className="text-[11px] text-muted-foreground">{service.project}</span>
            </div>
            <h2
              className="m-0 overflow-hidden text-ellipsis font-mono text-base font-medium text-foreground"
              id="service-sheet-title"
            >
              {service.name}
            </h2>
          </div>
          <button
            className="h-8 w-8 shrink-0 cursor-pointer rounded-full border border-border bg-background text-lg leading-none text-foreground-subtle hover:border-accent hover:text-accent focus-visible:border-accent focus-visible:text-accent focus-visible:outline-none"
            onClick={close}
            ref={closeButtonRef}
            type="button"
            aria-label="Close service details"
          >
            ×
          </button>
        </header>

        <dl className="mt-5 grid grid-cols-[100px_minmax(0,1fr)] gap-x-4 gap-y-4 text-xs">
          <Detail label="Runtime">{service.runtime === "docker" ? "Docker Compose" : "Host process"}</Detail>
          <Detail label="Status">
            <span className={unhealthy ? "text-danger" : "text-foreground-subtle"}>
              {service.state ?? "unknown"}
            </span>
          </Detail>
          {service.health && <Detail label="Health">{service.health}</Detail>}
          {service.container && <Detail label="Container">{service.container}</Detail>}
          {service.image && <Detail label="Image">{service.image}</Detail>}
          <Detail label="Ports">
            {service.ports.length === 0 ? (
              <span className="text-muted-foreground">None published</span>
            ) : (
              <div className="flex flex-col gap-2">
                {service.ports.map((port) => (
                  <div
                    className="flex min-w-0 items-center gap-2 font-mono text-foreground-subtle"
                    key={`${port.host ?? 0}:${port.container ?? 0}:${port.protocol}`}
                  >
                    <span className="min-w-0 overflow-hidden text-ellipsis">
                      {formatPort(port)}
                    </span>
                    {port.scheme && (
                      <span className="rounded bg-accent-subtle px-1.5 py-0.5 font-sans text-[10px] font-semibold uppercase text-accent">
                        {port.scheme}
                      </span>
                    )}
                  </div>
                ))}
              </div>
            )}
          </Detail>
        </dl>

        {httpPorts.length > 0 && (
          <div className="mt-6 flex flex-col gap-3">
            {httpPorts.map((port) => (
              <div className="rounded-xl bg-background p-3" key={port.host}>
                <div className="mb-3 flex items-center justify-between">
                  <span className="font-mono text-xs text-foreground-subtle">:{port.host}</span>
                  <span className="text-[11px] text-muted-foreground">
                    {port.exposedUrl ? "Available on tailnet" : "Local only"}
                  </span>
                </div>
                {port.exposedUrl ? (
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      className={primaryButtonClass}
                      onClick={() => window.open(port.exposedUrl, "_blank", "noopener,noreferrer")}
                      type="button"
                    >
                      Open
                    </button>
                    <button
                      className={secondaryButtonClass}
                      disabled={actionPending}
                      onClick={() => void runAction({ id: service.id, action: "unexpose", port: port.host })}
                      type="button"
                    >
                      {pendingAction === "unexpose" ? "Stopping…" : "Stop sharing"}
                    </button>
                  </div>
                ) : (
                  <button
                    className={`${primaryButtonClass} w-full`}
                    disabled={actionPending}
                    onClick={() => void runAction({ id: service.id, action: "expose", port: port.host })}
                    type="button"
                  >
                    {pendingAction === "expose" ? "Exposing…" : "Expose to tailnet"}
                  </button>
                )}
              </div>
            ))}
          </div>
        )}

        {actionError && <p className="mt-4 mb-0 text-xs text-danger">{actionError}</p>}

        <div className="mt-4">
          {confirmEnd ? (
            <div className="rounded-xl bg-danger/10 p-3">
              <p className="mt-0 mb-1 text-sm font-medium text-foreground">End {service.name}?</p>
              <p className="mt-0 mb-3 text-xs text-muted-foreground">
                This stops the {service.runtime === "docker" ? "container" : "process"}.
              </p>
              <div className="grid grid-cols-2 gap-2">
                <button
                  className={secondaryButtonClass}
                  disabled={actionPending}
                  onClick={() => setConfirmEnd(false)}
                  type="button"
                >
                  Keep running
                </button>
                <button
                  className={dangerButtonClass}
                  disabled={actionPending}
                  onClick={() => void runAction({ id: service.id, action: "end" })}
                  type="button"
                >
                  {pendingAction === "end" ? "Ending…" : "End now"}
                </button>
              </div>
            </div>
          ) : (
            <button
              className={`${dangerButtonClass} w-full`}
              disabled={actionPending}
              onClick={() => setConfirmEnd(true)}
              type="button"
            >
              End service…
            </button>
          )}
        </div>
      </section>
    </div>
  );
}

const buttonBase =
  "h-11 cursor-pointer rounded-lg border px-3 text-xs font-medium disabled:cursor-default disabled:opacity-50 focus-visible:outline-none";
const primaryButtonClass = `${buttonBase} border-accent bg-accent text-background hover:bg-accent/90 focus-visible:ring-2 focus-visible:ring-accent/40`;
const secondaryButtonClass = `${buttonBase} border-border bg-background text-foreground-subtle hover:border-accent hover:text-accent focus-visible:border-accent`;
const dangerButtonClass = `${buttonBase} border-danger/50 bg-transparent text-danger hover:border-danger focus-visible:border-danger`;

function Detail({ label, children }: { label: string; children: ReactNode }) {
  return (
    <>
      <dt className="text-muted-foreground">{label}</dt>
      <dd className="m-0 min-w-0 break-words text-foreground-subtle">{children}</dd>
    </>
  );
}

function formatPort(port: ServicePort) {
  if (port.host && port.container) {
    return `${port.host} → ${port.container}/${port.protocol}`;
  }
  if (port.host) {
    return `${port.host}/${port.protocol}`;
  }
  return `${port.container}/${port.protocol} · container only`;
}
