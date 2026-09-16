// Homechrome Service Worker — Web Push Notifications & Offline Handling

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('push', (event) => {
  let data = {
    title: 'Homechrome Handloom',
    body: 'New handloom collections & updates are live!',
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

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const targetUrl = (event.notification.data && event.notification.data.url) || '/';

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      // Focus open Homechrome window if present
      for (const client of clientList) {
        if (client.url.includes(self.location.origin) && 'focus' in client) {
          if ('navigate' in client) {
            client.navigate(targetUrl);
          }
          return client.focus();
        }
      }
      // Otherwise open a new window
      if (self.clients.openWindow) {
        return self.clients.openWindow(targetUrl);
      }
    })
  );
});
