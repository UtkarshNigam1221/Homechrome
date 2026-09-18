// Homechrome Service Worker — Web Push Notifications & Offline Handling

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
    data: {
      url: data.url || '/',
    },
    tag: data.tag || 'homechrome-push',
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
