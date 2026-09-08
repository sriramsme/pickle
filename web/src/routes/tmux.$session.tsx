import { createFileRoute } from "@tanstack/react-router";
import { TerminalPage } from "../features/terminal/TerminalPage";

export const Route = createFileRoute("/tmux/$session")({
  component: TerminalRoute,
});

function TerminalRoute() {
  const { session } = Route.useParams();
  return <TerminalPage session={session} />;
}
