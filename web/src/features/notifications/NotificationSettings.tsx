import { useEffect, useState } from "react";
import {
  getNotificationPublicKey,
  removeNotificationSubscription,
  saveNotificationSubscription,
  sendTestNotification,
} from "./api";
import {
  createBrowserSubscription,
  getBrowserSubscription,
  notificationSupportAvailable,
  requestNotificationPermission,
  serializeSubscription,
} from "./browser";

export function NotificationSettings() {
  const supported = notificationSupportAvailable();
  const [subscription, setSubscription] = useState<PushSubscription | null>(null);
  const [loading, setLoading] = useState(supported);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    if (!supported) {
      return;
    }
    void getBrowserSubscription()
      .then(setSubscription)
      .catch((reason: unknown) => setError(errorMessage(reason)))
      .finally(() => setLoading(false));
  }, [supported]);

  const enable = async () => {
    setBusy(true);
    setError("");
    setMessage("");
    let created: PushSubscription | null = null;
    try {
      await requestNotificationPermission();
      const publicKey = await getNotificationPublicKey();
      created = await createBrowserSubscription(publicKey);
      await saveNotificationSubscription(serializeSubscription(created));
      setSubscription(created);
    } catch (reason) {
      if (created) {
        await created.unsubscribe();
      }
      setError(errorMessage(reason));
    } finally {
      setBusy(false);
    }
  };

  const disable = async () => {
    if (!subscription) {
      return;
    }
    setBusy(true);
    setError("");
    setMessage("");
    try {
      await removeNotificationSubscription(subscription.endpoint);
      await subscription.unsubscribe();
      setSubscription(null);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setBusy(false);
    }
  };

  const sendTest = async () => {
    if (!subscription) {
      return;
    }
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const result = await sendTestNotification();
      setMessage(`Test sent to ${result.sent} ${result.sent === 1 ? "device" : "devices"}`);
    } catch (reason) {
      setError(errorMessage(reason));
    } finally {
      setBusy(false);
    }
  };

  const denied = supported && Notification.permission === "denied";
  const status = loading
    ? "Checking this device"
    : !supported
      ? "Unavailable in this browser"
      : denied
        ? "Blocked in device settings"
        : subscription
          ? "On for this device"
          : "Off on this device";

  return (
    <section className="mt-8 rounded-xl bg-surface-subtle p-5 max-[620px]:p-4">
      <div className="flex min-h-11 items-center justify-between gap-4">
        <div>
          <h2 className="m-0 text-sm font-semibold">Notifications</h2>
          <p className="mt-1.5 mb-0 text-xs text-foreground-subtle">{status}</p>
        </div>
        {!loading && supported && !denied && !subscription && (
          <button
            className="min-h-11 shrink-0 cursor-pointer rounded-lg border-0 bg-accent px-4 text-sm font-semibold text-[#181109] disabled:cursor-default disabled:opacity-45"
            disabled={busy}
            onClick={() => void enable()}
            type="button"
          >
            Enable
          </button>
        )}
      </div>

      {!supported && (
        <p className="mt-3 mb-0 text-xs leading-5 text-muted-foreground">
          On iPhone and iPad, open Pickle from its Home Screen icon.
        </p>
      )}
      {subscription && (
        <div className="mt-4 flex flex-wrap gap-2">
          <button
            className="min-h-11 cursor-pointer rounded-lg border-0 bg-accent px-4 text-sm font-semibold text-[#181109] disabled:cursor-default disabled:opacity-45"
            disabled={busy}
            onClick={() => void sendTest()}
            type="button"
          >
            Send test
          </button>
          <button
            className="min-h-11 cursor-pointer rounded-lg border-0 bg-surface px-4 text-sm text-foreground-subtle disabled:cursor-default disabled:opacity-45"
            disabled={busy}
            onClick={() => void disable()}
            type="button"
          >
            Turn off
          </button>
        </div>
      )}
      <div aria-live="polite">
        {message && <p className="mt-3 mb-0 text-xs text-foreground-subtle">{message}</p>}
        {error && (
          <p className="mt-3 mb-0 text-xs text-danger" role="alert">
            {error}
          </p>
        )}
      </div>
    </section>
  );
}

function errorMessage(reason: unknown) {
  return reason instanceof Error ? reason.message : "Notification setup failed.";
}
