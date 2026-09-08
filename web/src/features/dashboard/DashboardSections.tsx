import { Link } from "@tanstack/react-router";
import type { ReactNode } from "react";
import type { Project, Service, TmuxSession } from "./api";

const previewLimit = 3;
const nameClass =
  "flex min-w-0 items-center gap-[9px] overflow-hidden text-ellipsis whitespace-nowrap font-mono text-[13px] text-foreground";
const metaClass = "whitespace-nowrap text-[11px] text-muted-foreground";
const interactiveRowClass =
  "grid min-h-14 items-center gap-[18px] px-1 py-[11px] text-inherit no-underline hover:bg-surface-subtle focus-visible:bg-surface-subtle focus-visible:outline-none";

function SectionHeading({
  title,
  count,
  children,
}: {
  title: string;
  count?: number;
  children?: ReactNode;
}) {
  return (
    <div className="flex items-baseline gap-2 px-1 pb-3">
      <h2 className="m-0 text-[13px] font-semibold text-foreground">{title}</h2>
      {count !== undefined && <span className="text-xs text-muted">{count}</span>}
      {children}
    </div>
  );
}

function ViewAll({ to }: { to: "/services" | "/sessions" | "/projects" }) {
  return (
    <Link
      className="ml-auto text-[11px] text-muted-foreground no-underline hover:text-accent focus-visible:text-accent"
      to={to}
    >
      View all ›
    </Link>
  );
}

function Message({ children, error = false }: { children: ReactNode; error?: boolean }) {
  return (
    <div className={`px-1 py-5 text-xs ${error ? "text-danger" : "text-muted-foreground"}`}>
      {children}
    </div>
  );
}

export function ServicesSection({
  services,
  failed,
  preview,
}: {
  services?: Service[];
  failed: boolean;
  preview: boolean;
}) {
  const visibleServices = preview ? services?.slice(0, previewLimit) : services;

  return (
    <section className="[&+&]:mt-[42px]">
      <SectionHeading count={services?.length} title="services">
        {preview && (services?.length ?? 0) > previewLimit && <ViewAll to="/services" />}
      </SectionHeading>
      <div className="border-t border-border">
        {failed && services === undefined && <Message>Unable to load services</Message>}
        {!failed && services === undefined && <Message>Loading</Message>}
        {visibleServices?.map((service) => (
          <div
            className="grid min-h-14 grid-cols-[minmax(140px,1fr)_auto_auto] items-center gap-[18px] px-1 py-[11px] max-[620px]:grid-cols-[minmax(0,1fr)_auto] max-[620px]:gap-2.5"
            key={`${service.project}:${service.process}:${service.port}`}
          >
            <span className={nameClass}>{service.project}</span>
            <span className={`${metaClass} max-[620px]:col-start-1 max-[620px]:pl-[15px]`}>
              {service.process}
            </span>
            <span className="font-mono text-xs text-accent/80 max-[620px]:col-start-2 max-[620px]:row-span-2 max-[620px]:row-start-1">
              :{service.port}
            </span>
          </div>
        ))}
        {services?.length === 0 && <Message>No project services</Message>}
      </div>
    </section>
  );
}

export function SessionsSection({
  sessions,
  failed,
  preview,
}: {
  sessions?: TmuxSession[];
  failed: boolean;
  preview: boolean;
}) {
  const visibleSessions = preview ? sessions?.slice(0, previewLimit) : sessions;

  return (
    <section className="[&+&]:mt-[42px]">
      <SectionHeading count={sessions?.length} title="tmux sessions">
        {preview && (sessions?.length ?? 0) > previewLimit && <ViewAll to="/sessions" />}
      </SectionHeading>
      <div className="border-t border-border">
        {failed && sessions === undefined && <Message>Unable to load sessions</Message>}
        {!failed && sessions === undefined && <Message>Loading</Message>}
        {visibleSessions?.map((session) => (
          <Link
            className={`${interactiveRowClass} grid-cols-[minmax(140px,1fr)_auto_auto_14px] max-[620px]:grid-cols-[minmax(0,1fr)_auto_14px] max-[620px]:gap-2.5`}
            key={session.name}
            params={{ session: session.name }}
            to="/tmux/$session"
          >
            <span className={nameClass}>
              <span
                className={`h-1.5 w-1.5 shrink-0 rounded-full ${session.attached > 0 ? "bg-accent" : "bg-quiet"}`}
              />
              {session.name}
            </span>
            <span className={`${metaClass} max-[620px]:col-start-1 max-[620px]:pl-[15px]`}>
              {session.windows} {session.windows === 1 ? "window" : "windows"}
              {session.attached > 0 && ` · ${session.attached} attached`}
            </span>
            <time
              className={`${metaClass} max-[620px]:col-start-2 max-[620px]:row-span-2 max-[620px]:row-start-1`}
              dateTime={session.lastActivity}
            >
              {formatActivity(session.lastActivity)}
            </time>
            <span
              className="text-lg text-muted max-[620px]:col-start-3 max-[620px]:row-span-2 max-[620px]:row-start-1"
              aria-hidden="true"
            >
              ›
            </span>
          </Link>
        ))}
        {sessions?.length === 0 && (
          <Link
            className={`${interactiveRowClass} grid-cols-[minmax(140px,1fr)_auto_14px] max-[620px]:grid-cols-[minmax(0,1fr)_14px] max-[620px]:gap-2.5`}
            params={{ session: "pickle" }}
            to="/tmux/$session"
          >
            <span className={nameClass}>pickle</span>
            <span className={`${metaClass} max-[620px]:col-start-1 max-[620px]:pl-[15px]`}>
              start default session
            </span>
            <span className="text-lg text-muted" aria-hidden="true">
              ›
            </span>
          </Link>
        )}
      </div>
    </section>
  );
}

export function ProjectsSection({
  projects,
  failed,
  preview,
  sessionNames,
  openingProject,
  openFailed,
  openProject,
}: {
  projects?: Project[];
  failed: boolean;
  preview: boolean;
  sessionNames: Set<string>;
  openingProject?: string;
  openFailed: boolean;
  openProject: (name: string) => void;
}) {
  const visibleProjects = preview ? projects?.slice(0, previewLimit) : projects;

  return (
    <section className="[&+&]:mt-[42px]">
      <SectionHeading count={projects?.length} title="projects">
        {preview && (projects?.length ?? 0) > previewLimit && <ViewAll to="/projects" />}
      </SectionHeading>
      <div className="border-t border-border">
        {failed && projects === undefined && <Message>Unable to load projects</Message>}
        {!failed && projects === undefined && <Message>Loading</Message>}
        {visibleProjects?.map((project) => (
          <button
            className={`${interactiveRowClass} w-full cursor-pointer grid-cols-[minmax(140px,1fr)_auto_14px] border-0 bg-transparent text-left font-sans disabled:cursor-default disabled:opacity-[0.55] max-[620px]:grid-cols-[minmax(0,1fr)_auto_14px] max-[620px]:gap-2.5`}
            disabled={openingProject !== undefined}
            key={project.name}
            onClick={() => openProject(project.name)}
            type="button"
          >
            <span className={nameClass}>{project.name}</span>
            <span className={`${metaClass} max-[620px]:col-start-1 max-[620px]:pl-[15px]`}>
              {openingProject === project.name
                ? "opening"
                : sessionNames.has(project.session)
                  ? "open session"
                  : "start session"}
            </span>
            <span className="text-lg text-muted" aria-hidden="true">
              ›
            </span>
          </button>
        ))}
        {projects?.length === 0 && <Message>No projects</Message>}
        {openFailed && <Message error>Unable to open project</Message>}
      </div>
    </section>
  );
}

function formatActivity(value: string) {
  return new Intl.DateTimeFormat(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(value));
}
