import { requestJSON } from "../../lib/http";

export type NotificationSubscription = {
  endpoint: string;
  keys: {
    auth: string;
    p256dh: string;
  };
};

export async function getNotificationPublicKey() {
  const response = await requestJSON<{ publicKey: string }>("/api/notifications", {
    cache: "no-store",
  });
  return response.publicKey;
}

export function saveNotificationSubscription(subscription: NotificationSubscription) {
  return requestJSON<{ ok: boolean }>("/api/notifications", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(subscription),
  });
}

export function removeNotificationSubscription(endpoint: string) {
  return requestJSON<{ ok: boolean }>("/api/notifications", {
    method: "DELETE",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ endpoint }),
  });
}

export function sendTestNotification() {
  return requestJSON<{ sent: number }>("/api/notifications", {
    method: "POST",
  });
}
