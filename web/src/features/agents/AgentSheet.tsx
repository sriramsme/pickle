import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef } from "react";
import { AgentMark, agentLabels } from "./AgentMark";
import {
  agentUsageQuery,
  type Agent,
  type AgentActivity,
  type AgentUsage,
  type AgentUsageWindow,
} from "./api";

export function AgentSheet({
  open,
  close,
  agents,
  activities,
  openAgent,
  openActivity,
}: {
  open: boolean;
  close: () => void;
  agents: Agent[];
  activities: AgentActivity[];
  openAgent: (agent: Agent) => void;
  openActivity: (activity: AgentActivity) => void;
}) {
  const usage = useQuery({ ...agentUsageQuery, enabled: open });
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const sheetRef = useRef<HTMLElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        close();
      }
      if (event.key === "Tab") {
        const buttons = [
          ...(sheetRef.current?.querySelectorAll<HTMLButtonElement>("button:not(:disabled)") ?? []),
        ];
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
  }, [open, close]);

  if (!open) {
    return null;
  }

  return (
    <div
      className="fixed inset-0 z-50 flex animate-[sheet-backdrop-in_160ms_ease-out] items-end justify-center bg-black/65 motion-reduce:animate-none"
      onClick={(event) => {
        if (event.currentTarget === event.target) {
          close();
        }
      }}
      role="presentation"
    >
      <section
        aria-labelledby="agent-sheet-title"
        aria-modal="true"
        className="max-h-[min(86dvh,760px)] w-full max-w-[580px] animate-[sheet-in_200ms_var(--ease-standard)] overflow-y-auto overscroll-contain rounded-t-2xl border border-b-0 border-border bg-surface px-5 pt-2 pb-[max(24px,env(safe-area-inset-bottom))] shadow-2xl motion-reduce:animate-none sm:mb-5 sm:rounded-2xl sm:border sm:pb-6"
        ref={sheetRef}
        role="dialog"
      >
        <div className="mx-auto mb-3 h-1 w-9 rounded-full bg-quiet sm:hidden" />
        <header className="flex items-center gap-4 py-2">
          <h2 className="m-0 min-w-0 flex-1 text-base font-semibold text-foreground" id="agent-sheet-title">
            agents
          </h2>
          <button
            aria-label="Close agents"
            className="h-8 w-8 shrink-0 cursor-pointer rounded-full border border-border bg-background text-lg leading-none text-foreground-subtle hover:border-accent hover:text-accent focus-visible:border-accent focus-visible:text-accent focus-visible:outline-none"
            onClick={close}
            ref={closeButtonRef}
            type="button"
          >
            ×
          </button>
        </header>

        {(usage.isPending || (usage.data?.length ?? 0) > 0) && (
          <section className="mt-5">
            {usage.isPending ? <UsageSkeleton /> : usage.data?.map((item) => <UsageCard key={item.kind} usage={item} />)}
          </section>
        )}

        {activities.length > 0 && (
          <section className="mt-7">
            <SectionHeading count={activities.length} title="activity" />
            <div className="flex flex-col gap-2">
              {activities.map((activity) => (
                <button
                  aria-label={`Open ${activity.session}: ${activity.message}`}
                  className={rowClass}
                  key={activity.id}
                  onClick={() => openActivity(activity)}
                  type="button"
                >
                  <AgentBadge kind={activity.kind} />
                  <span className="min-w-0 flex-1">
                    <span className="block overflow-hidden text-ellipsis whitespace-nowrap text-sm text-foreground">
                      {activity.message}
                    </span>
                    <span className="mt-1 block font-mono text-[11px] text-muted-foreground">
                      {activity.session}:{activity.window}.{activity.pane}
                    </span>
                  </span>
                  <span className="text-lg text-muted" aria-hidden="true">
                    ›
                  </span>
                </button>
              ))}
            </div>
          </section>
        )}

        <section className="mt-7">
          <SectionHeading count={agents.length} title="active" />
          {agents.length > 0 ? (
            <div className="flex flex-col gap-2">
              {agents.map((agent) => (
                <button
                  aria-label={`Open ${agentLabels[agent.kind]} in ${agent.session}`}
                  className={rowClass}
                  key={agent.id}
                  onClick={() => openAgent(agent)}
                  type="button"
                >
                  <AgentBadge kind={agent.kind} />
                  <span className="min-w-0 flex-1">
                    <span className="block overflow-hidden text-ellipsis whitespace-nowrap font-mono text-sm text-foreground">
                      {agent.project || agent.session}
                    </span>
                    <span className="mt-1 block font-mono text-[11px] text-muted-foreground">
                      {agent.session}:{agent.window}.{agent.pane}
                    </span>
                  </span>
                  <span className="text-lg text-muted" aria-hidden="true">
                    ›
                  </span>
                </button>
              ))}
            </div>
          ) : (
            <div className="rounded-xl bg-background px-4 py-5 text-xs text-muted-foreground">
              No agents running
            </div>
          )}
        </section>
      </section>
    </div>
  );
}

function SectionHeading({ title, count }: { title: string; count?: number }) {
  return (
    <div className="mb-3 flex items-baseline gap-2 px-1">
      <h3 className="m-0 text-xs font-semibold text-foreground-subtle">{title}</h3>
      {count !== undefined && <span className="text-[11px] text-muted">{count}</span>}
    </div>
  );
}

function AgentBadge({ kind }: { kind?: Agent["kind"] }) {
  return (
    <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-surface-subtle text-foreground-subtle">
      {kind ? <AgentMark className="h-[18px] w-[18px]" kind={kind} /> : <span className="h-2 w-2 rounded-full bg-accent" />}
    </span>
  );
}

function UsageCard({ usage }: { usage: AgentUsage }) {
  return (
    <div className="rounded-2xl bg-background p-4">
      <div className="mb-4 flex items-center gap-2.5">
        <AgentMark className="h-5 w-5 text-foreground" kind={usage.kind} />
        <span className="text-xs font-medium capitalize text-foreground-subtle">
          {usage.plan?.replaceAll("_", " ")}
        </span>
      </div>
      <div className="grid grid-cols-2 gap-4">
        {usage.windows.map((window) => (
          <UsageWindow key={`${window.durationMinutes ?? 0}:${window.resetsAt ?? ""}`} window={window} />
        ))}
      </div>
    </div>
  );
}

function UsageWindow({ window }: { window: AgentUsageWindow }) {
  const used = Math.min(100, Math.max(0, window.usedPercent));
  return (
    <div>
      <div className="mb-1.5 flex min-w-0 items-baseline gap-1.5">
        <span className="shrink-0 text-[11px] text-foreground-subtle">
          {windowLabel(window.durationMinutes)}
        </span>
        {window.resetsAt && (
          <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap text-[10px] text-muted-foreground">
            · {resetLabel(window.resetsAt)}
          </span>
        )}
        <span className="ml-auto shrink-0 font-mono text-xs text-foreground">{used}%</span>
      </div>
      <div
        aria-label={`${used}% used`}
        aria-valuemax={100}
        aria-valuemin={0}
        aria-valuenow={used}
        className="h-1.5 overflow-hidden rounded-full bg-quiet"
        role="progressbar"
      >
        <div className="h-full rounded-full bg-accent" style={{ width: `${used}%` }} />
      </div>
    </div>
  );
}

function UsageSkeleton() {
  return (
    <div aria-label="Loading usage" className="animate-pulse rounded-2xl bg-background p-4">
      <div className="mb-4 h-5 w-20 rounded bg-quiet" />
      <div className="grid grid-cols-2 gap-4">
        <div className="h-8 rounded bg-surface-subtle" />
        <div className="h-8 rounded bg-surface-subtle" />
      </div>
    </div>
  );
}

function windowLabel(minutes?: number) {
  switch (minutes) {
    case 300:
      return "5-hour";
    case 1_440:
      return "Daily";
    case 10_080:
      return "Weekly";
    default:
      return minutes ? `${Math.round(minutes / 60)} hour` : "Limit";
  }
}

function resetLabel(value: string) {
  const seconds = Math.max(0, Math.round((new Date(value).getTime() - Date.now()) / 1_000));
  if (seconds < 60) {
    return "resets soon";
  }
  const hours = Math.floor(seconds / 3_600);
  const minutes = Math.floor((seconds % 3_600) / 60);
  if (hours < 24) {
    return `resets in ${hours > 0 ? `${hours}h ` : ""}${minutes}m`;
  }
  const days = Math.floor(hours / 24);
  return `resets in ${days}d ${hours % 24}h`;
}

const rowClass =
  "flex min-h-16 w-full cursor-pointer touch-manipulation items-center gap-3 rounded-xl border-0 bg-background px-3 py-2.5 text-left font-sans text-inherit hover:bg-surface-subtle focus-visible:bg-surface-subtle focus-visible:outline-none active:bg-accent-subtle";
