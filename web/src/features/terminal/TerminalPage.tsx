import { Link } from "@tanstack/react-router";
import { TerminalView } from "./TerminalView";

export function TerminalPage({ session }: { session: string }) {
  return (
    <main className="relative flex h-[var(--viewport-height,100dvh)] w-screen flex-col gap-1.5 bg-background pt-[max(8px,env(safe-area-inset-top))] pr-[max(8px,env(safe-area-inset-right))] pb-[max(8px,env(safe-area-inset-bottom))] pl-[max(8px,env(safe-area-inset-left))]">
      <header className="flex min-h-6 shrink-0 items-center gap-2.5 pr-[94px] text-xs text-muted">
        <Link className="text-foreground-subtle no-underline hover:text-accent focus-visible:text-accent" to="/sessions">
          ‹ sessions
        </Link>
        <span className="overflow-hidden text-ellipsis whitespace-nowrap max-[620px]:hidden">{session}</span>
      </header>
      <TerminalView session={session} />
    </main>
  );
}
