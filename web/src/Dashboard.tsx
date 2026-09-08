import { useEffect, useState } from "react";

type TmuxSession = {
  name: string;
  windows: number;
  attached: number;
  lastActivity: string;
};

type Project = {
  name: string;
  session: string;
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
  const [sessionsFailed, setSessionsFailed] = useState(false);
  const [projects, setProjects] = useState<Project[] | null>(null);
  const [projectsFailed, setProjectsFailed] = useState(false);
  const [openingProject, setOpeningProject] = useState<string | null>(null);
  const [openFailed, setOpenFailed] = useState(false);

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
          setSessionsFailed(false);
        }
      } catch {
        if (active) {
          setSessionsFailed(true);
        }
      }
    };

    const loadProjects = async () => {
      try {
        const response = await fetch("/api/projects", { cache: "no-store" });
        if (!response.ok) {
          throw new Error(`project request failed: ${response.status}`);
        }
        const nextProjects = (await response.json()) as Project[];
        if (active) {
          setProjects(nextProjects);
          setProjectsFailed(false);
        }
      } catch {
        if (active) {
          setProjectsFailed(true);
        }
      }
    };

    void loadSessions();
    void loadProjects();
    const refreshTimer = window.setInterval(() => {
      void loadSessions();
      void loadProjects();
    }, 5_000);
    return () => {
      active = false;
      window.clearInterval(refreshTimer);
    };
  }, []);

  const openProject = async (project: Project) => {
    setOpeningProject(project.name);
    setOpenFailed(false);
    try {
      const response = await fetch("/api/projects", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: project.name }),
      });
      if (!response.ok) {
        throw new Error(`open project failed: ${response.status}`);
      }
      const openedProject = (await response.json()) as Project;
      window.location.assign(sessionURL(openedProject.session));
    } catch {
      setOpenFailed(true);
      setOpeningProject(null);
    }
  };

  const sessionNames = new Set(sessions?.map((session) => session.name));

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
          {sessionsFailed && sessions === null && (
            <div className="session-message">Unable to load sessions</div>
          )}
          {!sessionsFailed && sessions === null && <div className="session-message">Loading</div>}
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

        <section className="project-section">
          <div className="session-heading">
            <h2>projects</h2>
            {projects && <span>{projects.length}</span>}
          </div>

          <div className="session-list">
            {projectsFailed && projects === null && (
              <div className="session-message">Unable to load projects</div>
            )}
            {!projectsFailed && projects === null && <div className="session-message">Loading</div>}
            {projects?.map((project) => (
              <button
                className="session-row project-row"
                disabled={openingProject !== null}
                key={project.name}
                onClick={() => void openProject(project)}
                type="button"
              >
                <span className="session-name">{project.name}</span>
                <span className="session-meta">
                  {openingProject === project.name
                    ? "opening"
                    : sessionNames.has(project.session)
                      ? "open session"
                      : "start session"}
                </span>
                <span className="session-arrow" aria-hidden="true">
                  ›
                </span>
              </button>
            ))}
            {projects?.length === 0 && <div className="session-message">No projects</div>}
            {openFailed && <div className="session-message error">Unable to open project</div>}
          </div>
        </section>
      </div>
    </main>
  );
}
