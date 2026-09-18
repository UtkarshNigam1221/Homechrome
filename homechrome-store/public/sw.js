// Homechrome Service Worker — Web Push notifications. No fetch handler: this
// worker does not cache or serve anything offline.

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
  let data = {
    title: 'Homechrome',
    body: 'Something new on the loom. Tap to take a look.',
    icon: '/icon.png',
    // Android draws the badge from alpha only, so it needs the mono glyph
    // rather than the full-colour icon, which would flatten to a filled square.
    badge: '/badge.png',
    url: '/',
    tag: 'homechrome-push',
  };

  if (event.data) {
    try {
      const parsed = event.data.json();
      data = { ...data, ...parsed };
    } catch {
      try {
        data.body = event.data.text();
      } catch {
        // Fallback default
      }
    }
  }

  const options = {
    body: data.body,
    icon: data.icon || '/icon.png',
    badge: data.badge || '/badge.png',
    // Wide banner shown when the notification is expanded. Omitted, not empty:
    // an empty string renders a broken-image slot on some Android builds.
    ...(data.image ? { image: data.image } : {}),
    data: {
      url: data.url || '/',
    },
    // A shared tag replaces the previous notification instead of stacking, so
    // fall back to a unique one rather than collapsing unrelated broadcasts.
    tag: data.tag || `homechrome-${Date.now()}`,
    renotify: true,
    vibrate: [150, 75, 150],
  };

  event.waitUntil(self.registration.showNotification(data.title, options));
});

/** Exact origin match — a substring test would accept evil-homechrome.in. */
function sameOrigin(url) {
  try {
    return new URL(url).origin === self.location.origin;
  } catch {
    return false;
  }
}

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = (event.notification.data && event.notification.data.url) || '/';

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      // Focus open Homechrome window if present
      for (const client of clientList) {
        if (sameOrigin(client.url) && 'focus' in client) {
          // Focus first: Chrome only allows navigate() inside the activation
          // window. navigate() rejects for uncontrolled clients, which
          // includeUncontrolled guarantees we can be handed.
          return client
            .focus()
            .then((focused) =>
              'navigate' in focused ? focused.navigate(targetUrl) : focused
            )
            .catch(() => self.clients.openWindow && self.clients.openWindow(targetUrl));
        }
      }
      // Otherwise open a new window
      if (self.clients.openWindow) {
        return self.clients.openWindow(targetUrl);
      }
    })
  );
});

// Browsers occasionally reissue a subscription on their own — key refresh,
// storage pressure, a push-service migration. Without this the old endpoint
// dies, the backend never hears about the new one, and the shopper silently
// stops receiving notifications. Re-register with the key we were given.
self.addEventListener('pushsubscriptionchange', (event) => {
  event.waitUntil(
    (async () => {
      const applicationServerKey =
        (event.oldSubscription && event.oldSubscription.options.applicationServerKey) ||
        (await fetch('/api/v1/store/push/vapid-key')
          .then((r) => r.json())
          .then((d) => d.public_key)
          .catch(() => null));
      if (!applicationServerKey) return;

      const subscription =
        event.newSubscription ||
        (await self.registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey,
        }));

      const { endpoint, keys } = subscription.toJSON();
      await fetch('/api/v1/store/push/subscribe', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ endpoint, keys }),
      });
    })()
  );
});
