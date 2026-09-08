import { Dashboard, type DashboardView } from "./Dashboard";
import { TerminalView } from "./Terminal";

function sessionFromPath() {
  const prefix = "/tmux/";
  if (!window.location.pathname.startsWith(prefix)) {
    return null;
  }

  const encodedSession = window.location.pathname.slice(prefix.length);
  if (encodedSession === "" || encodedSession.includes("/")) {
    return null;
  }

  try {
    return decodeURIComponent(encodedSession);
  } catch {
    return null;
  }
}

export function App() {
  const session = sessionFromPath();
  if (session) {
    return (
      <main className="app">
        <header className="terminal-topbar">
          <a href="/sessions">‹ sessions</a>
          <span>{session}</span>
        </header>
        <TerminalView session={session} />
      </main>
    );
  }

  const dashboardRoutes: Record<string, DashboardView> = {
    "/": "overview",
    "/services": "services",
    "/sessions": "sessions",
    "/projects": "projects",
  };
  return <Dashboard view={dashboardRoutes[window.location.pathname] ?? "overview"} />;
}
