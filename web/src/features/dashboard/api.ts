import { queryOptions } from "@tanstack/react-query";
import { requestJSON } from "../../lib/http";

export type TmuxSession = {
  name: string;
  windows: number;
  attached: number;
  lastActivity: string;
};

export type Project = {
  name: string;
  session: string;
};

export type Service = {
  project: string;
  process: string;
  port: number;
};

const refreshInterval = 5_000;

export const sessionsQuery = queryOptions({
  queryKey: ["tmux", "sessions"],
  queryFn: () => requestJSON<TmuxSession[]>("/api/tmux/sessions", { cache: "no-store" }),
  refetchInterval: refreshInterval,
});

export const projectsQuery = queryOptions({
  queryKey: ["projects"],
  queryFn: () => requestJSON<Project[]>("/api/projects", { cache: "no-store" }),
  refetchInterval: refreshInterval,
});

export const servicesQuery = queryOptions({
  queryKey: ["services"],
  queryFn: () => requestJSON<Service[]>("/api/services", { cache: "no-store" }),
  refetchInterval: refreshInterval,
});

export function openProject(name: string) {
  return requestJSON<Project>("/api/projects", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ name }),
  });
}
