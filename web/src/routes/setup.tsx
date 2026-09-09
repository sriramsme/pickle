import { createFileRoute } from "@tanstack/react-router";
import { SettingsPage } from "../features/settings/SettingsPage";

export const Route = createFileRoute("/setup")({
  component: () => <SettingsPage setup />,
});
