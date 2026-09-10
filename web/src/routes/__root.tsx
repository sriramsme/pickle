import { useQuery } from "@tanstack/react-query";
import { createRootRoute, Navigate, Outlet, useRouterState } from "@tanstack/react-router";
import { AgentIsland } from "../features/agents/AgentIsland";
import { settingsQuery } from "../features/settings/api";

export const Route = createRootRoute({
  component: RootLayout,
});

function RootLayout() {
  const pathname = useRouterState({ select: (state) => state.location.pathname });
  const settings = useQuery(settingsQuery);
  const settingsPage = pathname === "/setup" || pathname === "/settings";

  if (settings.isPending) {
    return <main className="min-h-dvh bg-background" />;
  }
  if (settings.isError) {
    return (
      <main className="flex min-h-dvh items-center justify-center bg-background p-6">
        <div className="text-center">
          <p className="m-0 text-sm text-danger">Unable to load Pickle settings</p>
          <button
            className="mt-4 cursor-pointer border-0 bg-transparent text-sm text-accent"
            onClick={() => void settings.refetch()}
            type="button"
          >
            Retry
          </button>
        </div>
      </main>
    );
  }
  if (!settings.data.configured && !settingsPage) {
    return <Navigate replace to="/setup" />;
  }
  if (settings.data.configured && pathname === "/setup") {
    return <Navigate replace to="/" />;
  }
  return (
    <>
      {pathname !== "/setup" && <AgentIsland />}
      <Outlet />
    </>
  );
}
