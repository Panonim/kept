const CACHE_NAME = 'kept-v4';
const urlsToCache = [
  '/',
  '/index.html',
  '/Static/logos/Kept Mascot Colored.svg',
];

self.addEventListener('install', (event) => {
  // Force this new service worker to activate immediately
  self.skipWaiting();
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => cache.addAll(urlsToCache))
  );
});

self.addEventListener('activate', (event) => {
  // Take control of all clients as soon as we activate
  event.waitUntil(clients.claim());
});

self.addEventListener('fetch', (event) => {
  event.respondWith(
    caches.match(event.request)
      .then((response) => {
        // Return cached version or fetch from network
        return response || fetch(event.request);
      })
  );
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (cacheName !== CACHE_NAME) {
            return caches.delete(cacheName);
          }
        })
      );
    })
  );
});

// Push notification handler
self.addEventListener('push', (event) => {
  console.log('Push received');
  try {
    if (!event.data) {
      console.error('Push event has no data');
      return;
    }
    let data = {};
    try {
      data = event.data.json();
    } catch (e) {
      console.error('Failed to parse push data');
      return;
    }
    const title = data.title || 'Kept Reminder';
    const options = {
      body: data.body || 'You have a promise to keep',
      icon: data.icon || '/Static/logos/Kept%20Mascot%20Colored.svg',
      badge: data.badge || '/Static/logos/Kept%20Mascot%20Colored.svg',
      tag: data.tag || 'promise-reminder',
      requireInteraction: false,
      data: data.data || {},
    };
    event.waitUntil(
      self.registration.showNotification(title, options)
        .then(() => console.log('Notification shown'))
        .catch(err => console.error('Failed to show notification'))
    );
  } catch (error) {
    console.error('Error in push event handler');
  }
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil(
    clients.openWindow('/')
  );
});
