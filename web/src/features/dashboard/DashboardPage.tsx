import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate } from "@tanstack/react-router";
import { useCallback, useState } from "react";
import {
  openProject,
  projectsQuery,
  runServiceAction,
  servicesQuery,
  sessionsQuery,
} from "./api";
import { ProjectsSection, ServicesSection, SessionsSection } from "./DashboardSections";
import { ServiceSheet } from "./ServiceSheet";

export type DashboardView = "overview" | "services" | "sessions" | "projects";

export function DashboardPage({ view }: { view: DashboardView }) {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [selectedServiceID, setSelectedServiceID] = useState<string>();
  const closeService = useCallback(() => setSelectedServiceID(undefined), []);
  const preview = view === "overview";
  const needsSessions = preview || view === "sessions" || view === "projects";
  const needsProjects = preview || view === "projects";
  const needsServices = preview || view === "services";

  const sessions = useQuery({ ...sessionsQuery, enabled: needsSessions });
  const projects = useQuery({ ...projectsQuery, enabled: needsProjects });
  const services = useQuery({ ...servicesQuery, enabled: needsServices });
  const projectMutation = useMutation({
    mutationFn: openProject,
    onSuccess: (project) => {
      void queryClient.invalidateQueries({ queryKey: sessionsQuery.queryKey });
      void navigate({ to: "/tmux/$session", params: { session: project.session } });
    },
  });
  const serviceActionMutation = useMutation({
    mutationFn: runServiceAction,
    onSuccess: async (_, variables) => {
      if (variables.action === "end") {
        closeService();
      }
      await queryClient.invalidateQueries({ queryKey: servicesQuery.queryKey });
    },
  });
  const selectService = (id: string) => {
    serviceActionMutation.reset();
    setSelectedServiceID(id);
  };

  const sessionNames = new Set(sessions.data?.map((session) => session.name));
  const selectedService = services.data?.find((service) => service.id === selectedServiceID);

  return (
    <main className="h-full min-h-dvh w-full overflow-y-auto bg-background pt-[max(24px,env(safe-area-inset-top))] pr-[max(20px,env(safe-area-inset-right))] pb-[max(24px,env(safe-area-inset-bottom))] pl-[max(20px,env(safe-area-inset-left))] [-webkit-overflow-scrolling:touch]">
      <div className="mx-auto w-full max-w-[760px]">
        <header className="mt-1 mb-10 flex max-[620px]:mb-8">
          <Link className="flex items-center gap-2.5 text-inherit no-underline" to="/">
            <img className="h-[34px] w-[34px] rounded-lg" src="/apple-touch-icon.png" alt="" />
            <h1 className="m-0 text-lg font-semibold">Pickle</h1>
          </Link>
        </header>

        {(view === "overview" || view === "services") && (
          <ServicesSection
            services={services.data}
            failed={services.isError}
            preview={preview}
            selectService={selectService}
          />
        )}
        {(view === "overview" || view === "sessions") && (
          <SessionsSection
            sessions={sessions.data}
            failed={sessions.isError}
            preview={preview}
          />
        )}
        {(view === "overview" || view === "projects") && (
          <ProjectsSection
            projects={projects.data}
            failed={projects.isError}
            preview={preview}
            sessionNames={sessionNames}
            openingProject={projectMutation.isPending ? projectMutation.variables : undefined}
            openFailed={projectMutation.isError}
            openProject={(name) => projectMutation.mutate(name)}
          />
        )}
      </div>
      <ServiceSheet
        service={selectedService}
        close={closeService}
        runAction={serviceActionMutation.mutateAsync}
        actionError={serviceActionMutation.error?.message}
        pendingAction={serviceActionMutation.isPending ? serviceActionMutation.variables?.action : undefined}
      />
    </main>
  );
}
