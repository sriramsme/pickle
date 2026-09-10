import { queryOptions } from "@tanstack/react-query";
import { requestJSON } from "../../lib/http";

export type AgentKind = "codex" | "claude" | "opencode" | "pi" | "hermes";

export type Agent = {
  id: string;
  kind: AgentKind;
  project?: string;
  session: string;
  window: number;
  pane: number;
};

export type AgentActivity = {
  id: string;
  kind?: AgentKind;
  session: string;
  window: number;
  pane: number;
  message: string;
  url: string;
  updatedAt: string;
};

export type AgentUsage = {
  kind: AgentKind;
  plan?: string;
  windows: AgentUsageWindow[];
};

export type AgentUsageWindow = {
  usedPercent: number;
  durationMinutes?: number;
  resetsAt?: string;
};

const refreshInterval = 5_000;

export const agentsQuery = queryOptions({
  queryKey: ["agents"],
  queryFn: () => requestJSON<Agent[]>("/api/agents", { cache: "no-store" }),
  refetchInterval: refreshInterval,
});

export const agentActivityQuery = queryOptions({
  queryKey: ["agents", "activity"],
  queryFn: () =>
    requestJSON<AgentActivity[]>("/api/agent-activity", { cache: "no-store" }),
  refetchInterval: refreshInterval,
});

export const agentUsageQuery = queryOptions({
  queryKey: ["agents", "usage"],
  queryFn: () => requestJSON<AgentUsage[]>("/api/agent-usage", { cache: "no-store" }),
  staleTime: 60_000,
});

export function dismissAgentActivity(id: string) {
  return requestJSON<{ ok: true }>(
    `/api/agent-activity?id=${encodeURIComponent(id)}`,
    { method: "DELETE" },
  );
}
