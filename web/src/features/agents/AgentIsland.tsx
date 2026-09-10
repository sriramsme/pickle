import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useCallback, useState } from "react";
import { AgentMark } from "./AgentMark";
import { AgentSheet } from "./AgentSheet";
import {
  agentActivityQuery,
  agentsQuery,
  dismissAgentActivity,
  type Agent,
  type AgentActivity,
  type AgentKind,
} from "./api";

export function AgentIsland() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const [sheetOpen, setSheetOpen] = useState(false);
  const closeSheet = useCallback(() => setSheetOpen(false), []);
  const agents = useQuery(agentsQuery);
  const activity = useQuery(agentActivityQuery);
  const dismiss = useMutation({
    mutationFn: dismissAgentActivity,
    onSuccess: (_, id) => {
      queryClient.setQueryData<AgentActivity[]>(
        agentActivityQuery.queryKey,
        (events) => events?.filter((event) => event.id !== id) ?? [],
      );
    },
  });

  const latest = activity.data?.[0];
  const session = sessionFromPath(pathname);
  const sessionAgents = session
    ? agents.data?.filter((agent) => agent.session === session) ?? []
    : [];

  const openAgent = (agent: Agent) => {
    closeSheet();
    void navigate({ to: "/tmux/$session", params: { session: agent.session } });
  };
  const openActivity = async (event: AgentActivity) => {
    closeSheet();
    await dismiss.mutateAsync(event.id).catch(() => undefined);
    await navigate({ href: event.url });
  };

  return (
    <>
      <button
        aria-expanded={sheetOpen}
        aria-haspopup="dialog"
        aria-label={latest ? `Open agents: ${latest.message}` : "Open agents"}
        className="fixed top-[max(10px,env(safe-area-inset-top))] left-1/2 z-40 flex h-8 max-w-[min(360px,46vw)] -translate-x-1/2 cursor-pointer touch-manipulation items-center gap-2 overflow-hidden rounded-full border border-border bg-surface/95 px-3 text-left text-[11px] text-foreground-subtle shadow-[0_8px_28px_rgba(0,0,0,0.28)] backdrop-blur-md transition-colors duration-200 hover:border-accent/50 hover:bg-surface focus-visible:border-accent focus-visible:outline-none active:bg-accent-subtle max-[620px]:max-w-[40vw]"
        onClick={() => setSheetOpen(true)}
        type="button"
      >
        {latest ? (
          <>
            {latest.kind ? (
              <AgentMark className="h-3.5 w-3.5 shrink-0 text-accent" kind={latest.kind} />
            ) : (
              <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-accent" aria-hidden="true" />
            )}
            <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">
              {latest.message}
            </span>
          </>
        ) : session ? (
          <>
            <span className="min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">{session}</span>
            <AgentMarks kinds={sessionAgents.map((agent) => agent.kind)} />
          </>
        ) : (agents.data?.length ?? 0) > 0 ? (
          <>
            <AgentMarks kinds={agents.data?.map((agent) => agent.kind) ?? []} />
            <span>{agents.data?.length}</span>
          </>
        ) : (
          <>
            <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-muted" aria-hidden="true" />
            <span>Pickle</span>
          </>
        )}
      </button>
      <AgentSheet
        activities={activity.data ?? []}
        agents={agents.data ?? []}
        close={closeSheet}
        open={sheetOpen}
        openActivity={(event) => void openActivity(event)}
        openAgent={openAgent}
      />
    </>
  );
}

function AgentMarks({ kinds }: { kinds: AgentKind[] }) {
  const visible = [...new Set(kinds)].slice(0, 3);
  if (visible.length === 0) {
    return null;
  }
  return (
    <span className="flex shrink-0 -space-x-1" aria-hidden="true">
      {visible.map((kind) => (
        <span
          className="flex h-[18px] w-[18px] items-center justify-center rounded-full border border-surface bg-surface-subtle text-accent"
          key={kind}
        >
          <AgentMark className="h-2.5 w-2.5" kind={kind} />
        </span>
      ))}
    </span>
  );
}

function sessionFromPath(pathname: string) {
  if (!pathname.startsWith("/tmux/")) {
    return undefined;
  }
  try {
    return decodeURIComponent(pathname.slice("/tmux/".length));
  } catch {
    return pathname.slice("/tmux/".length);
  }
}
