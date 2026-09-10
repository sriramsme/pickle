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
  const requestedTarget = new URL(event.notification.data?.url || "/", self.location.origin);
  const target =
    requestedTarget.origin === self.location.origin
      ? requestedTarget
      : new URL("/", self.location.origin);
  const path = `${target.pathname}${target.search}${target.hash}`;

  event.waitUntil(
    self.clients.matchAll({ type: "window", includeUncontrolled: true }).then(async (windows) => {
      const existing = windows.find((client) => new URL(client.url).origin === self.location.origin);
      if (existing) {
        try {
          await existing.focus();
        } catch {
          // The navigation message below can still wake the app.
        }
        try {
          await existing.navigate(target.href);
        } catch {
          // Some iOS versions wake a Home Screen app but reject navigate().
        }
        await sendNavigation(existing, path);
        return;
      }

      const opened = await self.clients.openWindow(target.href);
      if (opened) {
        await sendNavigation(opened, path);
      }
    }),
  );
});

async function sendNavigation(client, url) {
  const message = { type: "pickle:navigate", url };
  postNavigation(client, message);
  await delay(500);
  postNavigation(client, message);
  await delay(1000);
  postNavigation(client, message);
}

function postNavigation(client, message) {
  try {
    client.postMessage(message);
  } catch {
    // A newly opened iOS Home Screen client can be inert briefly.
  }
}

function delay(milliseconds) {
  return new Promise((resolve) => setTimeout(resolve, milliseconds));
}
