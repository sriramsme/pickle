import { createRoot } from "react-dom/client";
import "./styles.css";
import { App } from "./App";
import { registerNotificationWorker } from "./features/notifications/browser";

createRoot(document.getElementById("root")!).render(<App />);

void registerNotificationWorker().catch(() => undefined);
