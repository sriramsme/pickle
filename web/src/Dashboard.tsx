import { useEffect, useState } from "react";

type TmuxSession = {
  name: string;
  windows: number;
  attached: number;
  lastActivity: string;
};

function sessionURL(name: string) {
  return `/tmux/${encodeURIComponent(name)}`;
}

function formatActivity(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}

export function Dashboard() {
  const [sessions, setSessions] = useState<TmuxSession[] | null>(null);
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    let active = true;

    const loadSessions = async () => {
      try {
        const response = await fetch("/api/tmux/sessions", { cache: "no-store" });
        if (!response.ok) {
          throw new Error(`session request failed: ${response.status}`);
        }
        const nextSessions = (await response.json()) as TmuxSession[];
        if (active) {
          setSessions(nextSessions);
          setFailed(false);
        }
      } catch {
        if (active) {
          setFailed(true);
        }
      }
    };

    void loadSessions();
    const refreshTimer = window.setInterval(loadSessions, 5_000);
    return () => {
      active = false;
      window.clearInterval(refreshTimer);
    };
  }, []);

  return (
    <main className="dashboard">
      <div className="dashboard-shell">
        <header className="dashboard-header">
          <img src="/apple-touch-icon.png" alt="" />
          <h1>Pickle</h1>
        </header>

        <div className="session-heading">
          <h2>tmux sessions</h2>
          {sessions && <span>{sessions.length}</span>}
        </div>

        <div className="session-list">
          {failed && sessions === null && <div className="session-message">Unable to load sessions</div>}
          {!failed && sessions === null && <div className="session-message">Loading</div>}
          {sessions?.map((session) => (
            <a className="session-row" href={sessionURL(session.name)} key={session.name}>
              <span className="session-name">
                <span className={session.attached > 0 ? "session-dot active" : "session-dot"} />
                {session.name}
              </span>
              <span className="session-meta">
                {session.windows} {session.windows === 1 ? "window" : "windows"}
                {session.attached > 0 && ` · ${session.attached} attached`}
              </span>
              <time dateTime={session.lastActivity}>{formatActivity(session.lastActivity)}</time>
              <span className="session-arrow" aria-hidden="true">
                ›
              </span>
            </a>
          ))}
          {sessions?.length === 0 && (
            <a className="session-row empty" href={sessionURL("pickle")}>
              <span className="session-name">pickle</span>
              <span className="session-meta">start default session</span>
              <span className="session-arrow" aria-hidden="true">
                ›
              </span>
            </a>
          )}
        </div>
      </div>
    </main>
  );
}
