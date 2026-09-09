self.addEventListener("install", (event) => {
  event.waitUntil(self.skipWaiting());
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", (event) => {
  let notification = {};
  try {
    notification = event.data?.json() ?? {};
  } catch {
    notification = {};
  }

  event.waitUntil(
    self.registration.showNotification(notification.title || "Pickle", {
      body: notification.body || "Pickle needs your attention.",
      icon: "/web-app-manifest-192x192.png",
      badge: "/favicon-96x96.png",
      tag: notification.tag,
      data: { url: notification.url || "/" },
    }),
  );
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = new URL(event.notification.data?.url || "/", self.location.origin).href;

  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then(async (windows) => {
      const existing = windows.find((client) => new URL(client.url).origin === self.location.origin);
      if (existing) {
        await existing.navigate(target);
        return existing.focus();
      }
      return self.clients.openWindow(target);
    }),
  );
});
