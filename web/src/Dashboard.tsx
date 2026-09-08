import { useEffect, useState } from "react";

export type DashboardView = "overview" | "services" | "sessions" | "projects";

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

type Service = {
  project: string;
  process: string;
  port: number;
};

const previewLimit = 3;

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

function SectionHeading({
  title,
  count,
  viewAll,
}: {
  title: string;
  count: number | null;
  viewAll?: string;
}) {
  return (
    <div className="section-heading">
      <h2>{title}</h2>
      {count !== null && <span>{count}</span>}
      {viewAll && <a href={viewAll}>View all ›</a>}
    </div>
  );
}

function ServicesSection({
  services,
  failed,
  preview,
}: {
  services: Service[] | null;
  failed: boolean;
  preview: boolean;
}) {
  const visibleServices = preview ? services?.slice(0, previewLimit) : services;

  return (
    <section className="dashboard-section">
      <SectionHeading
        count={services?.length ?? null}
        title="services"
        viewAll={preview && (services?.length ?? 0) > previewLimit ? "/services" : undefined}
      />
      <div className="session-list">
        {failed && services === null && (
          <div className="session-message">Unable to load services</div>
        )}
        {!failed && services === null && <div className="session-message">Loading</div>}
        {visibleServices?.map((service) => (
          <div
            className="service-row"
            key={`${service.project}:${service.process}:${service.port}`}
          >
            <span className="session-name">{service.project}</span>
            <span className="session-meta">{service.process}</span>
            <span className="service-port">:{service.port}</span>
          </div>
        ))}
        {services?.length === 0 && <div className="session-message">No project services</div>}
      </div>
    </section>
  );
}

function SessionsSection({
  sessions,
  failed,
  preview,
}: {
  sessions: TmuxSession[] | null;
  failed: boolean;
  preview: boolean;
}) {
  const visibleSessions = preview ? sessions?.slice(0, previewLimit) : sessions;

  return (
    <section className="dashboard-section">
      <SectionHeading
        count={sessions?.length ?? null}
        title="tmux sessions"
        viewAll={preview && (sessions?.length ?? 0) > previewLimit ? "/sessions" : undefined}
      />
      <div className="session-list">
        {failed && sessions === null && (
          <div className="session-message">Unable to load sessions</div>
        )}
        {!failed && sessions === null && <div className="session-message">Loading</div>}
        {visibleSessions?.map((session) => (
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
    </section>
  );
}

function ProjectsSection({
  projects,
  failed,
  preview,
  sessionNames,
  openingProject,
  openFailed,
  openProject,
}: {
  projects: Project[] | null;
  failed: boolean;
  preview: boolean;
  sessionNames: Set<string>;
  openingProject: string | null;
  openFailed: boolean;
  openProject: (project: Project) => void;
}) {
  const visibleProjects = preview ? projects?.slice(0, previewLimit) : projects;

  return (
    <section className="dashboard-section">
      <SectionHeading
        count={projects?.length ?? null}
        title="projects"
        viewAll={preview && (projects?.length ?? 0) > previewLimit ? "/projects" : undefined}
      />
      <div className="session-list">
        {failed && projects === null && (
          <div className="session-message">Unable to load projects</div>
        )}
        {!failed && projects === null && <div className="session-message">Loading</div>}
        {visibleProjects?.map((project) => (
          <button
            className="session-row project-row"
            disabled={openingProject !== null}
            key={project.name}
            onClick={() => openProject(project)}
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
  );
}

export function Dashboard({ view = "overview" }: { view?: DashboardView }) {
  const [sessions, setSessions] = useState<TmuxSession[] | null>(null);
  const [sessionsFailed, setSessionsFailed] = useState(false);
  const [projects, setProjects] = useState<Project[] | null>(null);
  const [projectsFailed, setProjectsFailed] = useState(false);
  const [services, setServices] = useState<Service[] | null>(null);
  const [servicesFailed, setServicesFailed] = useState(false);
  const [openingProject, setOpeningProject] = useState<string | null>(null);
  const [openFailed, setOpenFailed] = useState(false);

  useEffect(() => {
    let active = true;
    const needsSessions = view === "overview" || view === "sessions" || view === "projects";
    const needsProjects = view === "overview" || view === "projects";
    const needsServices = view === "overview" || view === "services";

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

    const loadServices = async () => {
      try {
        const response = await fetch("/api/services", { cache: "no-store" });
        if (!response.ok) {
          throw new Error(`service request failed: ${response.status}`);
        }
        const nextServices = (await response.json()) as Service[];
        if (active) {
          setServices(nextServices);
          setServicesFailed(false);
        }
      } catch {
        if (active) {
          setServicesFailed(true);
        }
      }
    };

    const refresh = () => {
      if (needsSessions) void loadSessions();
      if (needsProjects) void loadProjects();
      if (needsServices) void loadServices();
    };

    refresh();
    const refreshTimer = window.setInterval(refresh, 5_000);
    return () => {
      active = false;
      window.clearInterval(refreshTimer);
    };
  }, [view]);

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

  const preview = view === "overview";
  const sessionNames = new Set(sessions?.map((session) => session.name));

  return (
    <main className="dashboard">
      <div className="dashboard-shell">
        <header className="dashboard-header">
          <a className="dashboard-brand" href="/">
            <img src="/apple-touch-icon.png" alt="" />
            <h1>Pickle</h1>
          </a>
        </header>

        {(view === "overview" || view === "services") && (
          <ServicesSection services={services} failed={servicesFailed} preview={preview} />
        )}
        {(view === "overview" || view === "sessions") && (
          <SessionsSection sessions={sessions} failed={sessionsFailed} preview={preview} />
        )}
        {(view === "overview" || view === "projects") && (
          <ProjectsSection
            projects={projects}
            failed={projectsFailed}
            preview={preview}
            sessionNames={sessionNames}
            openingProject={openingProject}
            openFailed={openFailed}
            openProject={(project) => void openProject(project)}
          />
        )}
      </div>
    </main>
  );
}
