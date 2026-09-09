import { queryOptions } from "@tanstack/react-query";
import { requestJSON } from "../../lib/http";

export type HostSettings = {
  configured: boolean;
  projectsDirectory: string;
  projectsDirectoryLocked: boolean;
};

export const settingsQuery = queryOptions({
  queryKey: ["settings"],
  queryFn: () => requestJSON<HostSettings>("/api/settings", { cache: "no-store" }),
});

export function saveSettings(projectsDirectory: string) {
  return requestJSON<HostSettings>("/api/settings", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ projectsDirectory }),
  });
}
