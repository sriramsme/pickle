import { Link } from "@tanstack/react-router";

export function AppHeader({ action }: { action?: "settings" | "done" }) {
  return (
    <header className="mt-1 mb-10 flex items-center justify-between max-[620px]:mb-8">
      <Link className="flex items-center gap-2.5 text-inherit no-underline" to="/">
        <img className="h-[34px] w-[34px] rounded-lg" src="/apple-touch-icon.png" alt="" />
        <span className="text-lg font-semibold">Pickle</span>
      </Link>
      {action && (
        <Link
          className="inline-flex min-h-11 items-center px-1 text-xs text-muted-foreground no-underline hover:text-accent focus-visible:text-accent focus-visible:outline-none"
          to={action === "settings" ? "/settings" : "/"}
        >
          {action === "settings" ? "Settings" : "Done"}
        </Link>
      )}
    </header>
  );
}
