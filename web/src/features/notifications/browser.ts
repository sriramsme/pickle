import type { NotificationSubscription } from "./api";

export function notificationSupportAvailable() {
  return (
    window.isSecureContext &&
    "Notification" in window &&
    "serviceWorker" in navigator &&
    "PushManager" in window
  );
}

export function registerNotificationWorker() {
  if (!("serviceWorker" in navigator)) {
    return Promise.resolve(undefined);
  }
  return navigator.serviceWorker.register("/service-worker.js");
}

export async function getBrowserSubscription() {
  const registration = await registerNotificationWorker();
  return registration ? registration.pushManager.getSubscription() : null;
}

export async function createBrowserSubscription(publicKey: string) {
  if (Notification.permission !== "granted") {
    throw new Error("Notifications are blocked in device settings.");
  }
  const registration = await registerNotificationWorker();
  if (!registration) {
    throw new Error("Notifications are not available in this browser.");
  }
  return registration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: decodeBase64URL(publicKey),
  });
}

export async function requestNotificationPermission() {
  const permission = await Notification.requestPermission();
  if (permission !== "granted") {
    throw new Error("Notifications are blocked in device settings.");
  }
}

export function serializeSubscription(subscription: PushSubscription): NotificationSubscription {
  const value = subscription.toJSON();
  const auth = value.keys?.auth;
  const p256dh = value.keys?.p256dh;
  if (!value.endpoint || !auth || !p256dh) {
    throw new Error("The browser returned an incomplete notification subscription.");
  }
  return {
    endpoint: value.endpoint,
    keys: { auth, p256dh },
  };
}

function decodeBase64URL(value: string) {
  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const decoded = atob((value + padding).replaceAll("-", "+").replaceAll("_", "/"));
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}
