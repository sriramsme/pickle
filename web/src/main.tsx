import { createRoot } from "react-dom/client";
import "./styles.css";
import { App } from "./App";
import {
  listenForNotificationNavigation,
  registerNotificationWorker,
} from "./features/notifications/browser";

listenForNotificationNavigation();

createRoot(document.getElementById("root")!).render(<App />);

void registerNotificationWorker().catch(() => undefined);
