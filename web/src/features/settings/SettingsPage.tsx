import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState, type FormEvent } from "react";
import { AppHeader } from "../../app/AppHeader";
import { projectsQuery, servicesQuery } from "../dashboard/api";
import { saveSettings, settingsQuery, type HostSettings } from "./api";

export function SettingsPage({ setup = false }: { setup?: boolean }) {
  const settings = useQuery(settingsQuery);

  if (!settings.data) {
    return null;
  }

  return <SettingsForm key={settings.data.projectsDirectory} settings={settings.data} setup={setup} />;
}

function SettingsForm({ settings, setup }: { settings: HostSettings; setup: boolean }) {
  const queryClient = useQueryClient();
  const [projectsDirectory, setProjectsDirectory] = useState(settings.projectsDirectory);
  const mutation = useMutation({
    mutationFn: saveSettings,
    onSuccess: async (updated) => {
      queryClient.setQueryData(settingsQuery.queryKey, updated);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: projectsQuery.queryKey }),
        queryClient.invalidateQueries({ queryKey: servicesQuery.queryKey }),
      ]);
    },
  });
  const changed = !settings.configured || projectsDirectory.trim() !== settings.projectsDirectory;

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    mutation.mutate(projectsDirectory.trim());
  };

  return (
    <main className="h-full min-h-dvh w-full overflow-y-auto bg-background pt-[max(24px,env(safe-area-inset-top))] pr-[max(20px,env(safe-area-inset-right))] pb-[max(24px,env(safe-area-inset-bottom))] pl-[max(20px,env(safe-area-inset-left))] [-webkit-overflow-scrolling:touch]">
      <div className="mx-auto w-full max-w-[640px]">
        <AppHeader action={setup ? undefined : "done"} />

        <section>
          <h1 className="m-0 text-xl font-semibold">
            {setup ? "Choose your projects folder" : "Settings"}
          </h1>

          <form className="mt-8 rounded-xl bg-surface-subtle p-5 max-[620px]:p-4" onSubmit={submit}>
            <label className="block text-xs font-medium text-foreground-subtle" htmlFor="projects-directory">
              Projects folder
            </label>
            <input
              autoCapitalize="none"
              autoComplete="off"
              autoCorrect="off"
              className="mt-3 w-full rounded-lg border border-border bg-background px-3.5 py-3 font-mono text-[13px] text-foreground outline-none transition-colors focus:border-accent disabled:cursor-not-allowed disabled:opacity-60"
              disabled={settings.projectsDirectoryLocked || mutation.isPending}
              id="projects-directory"
              onChange={(event) => setProjectsDirectory(event.target.value)}
              spellCheck={false}
              type="text"
              value={projectsDirectory}
            />

            {settings.projectsDirectoryLocked && (
              <p className="mt-3 mb-0 text-xs text-muted-foreground">
                This folder is set by the <code className="font-mono">-projects-dir</code> flag.
              </p>
            )}
            {mutation.isError && (
              <p className="mt-3 mb-0 text-xs text-danger" role="alert">
                {mutation.error.message}
              </p>
            )}

            {!settings.projectsDirectoryLocked && (
              <button
                className="mt-5 min-h-11 rounded-lg border-0 bg-accent px-4 font-sans text-sm font-semibold text-[#181109] enabled:cursor-pointer disabled:opacity-45 max-[620px]:w-full"
                disabled={!changed || mutation.isPending}
                type="submit"
              >
                {mutation.isPending ? "Saving" : setup ? "Finish setup" : "Save"}
              </button>
            )}
          </form>
        </section>

        <section className="mt-8 rounded-xl bg-surface-subtle p-5 max-[620px]:p-4">
          <h2 className="m-0 text-sm font-semibold">Remote access</h2>
          <p className="mt-3 mb-0 text-sm leading-6 text-foreground-subtle">
            Tailscale is the recommended way to reach Pickle from your other devices.
          </p>
          <code className="mt-4 block overflow-x-auto rounded-lg bg-background px-3.5 py-3 font-mono text-xs text-foreground-subtle">
            tailscale serve --bg 8080
          </code>
          <a
            className="mt-2 inline-flex min-h-11 items-center text-sm text-accent no-underline hover:underline"
            href="https://tailscale.com/docs/features/tailscale-serve"
            rel="noreferrer"
            target="_blank"
          >
            Tailscale Serve guide ↗
          </a>
        </section>
      </div>
    </main>
  );
}
