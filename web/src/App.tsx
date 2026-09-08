import { Dashboard } from "./Dashboard";
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
  if (!session) {
    return <Dashboard />;
  }

  return (
    <main className="app">
      <header className="terminal-topbar">
        <a href="/">‹ sessions</a>
        <span>{session}</span>
      </header>
      <TerminalView session={session} />
    </main>
  );
}
