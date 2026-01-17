const API_URL = '/api';

export class NotificationService {
  // Accept optional authService to access token and authenticatedFetch
  constructor(authService) {
    this.authService = authService;
    this.init();
  }

  async init() {
    if (!('Notification' in window)) {
      console.log('This browser does not support notifications');
      return;
    }

    if (!('serviceWorker' in navigator)) {
      console.log('Service Workers are not supported');
      return;
    }
  }

  async checkPermission() {
    if (Notification.permission === 'default') {
      // Don't auto-request, wait for user action
      return;
    }

    if (Notification.permission === 'granted') {
      await this.subscribeToPush();
    }
  }

  async requestPermission() {
    const permission = await Notification.requestPermission();
    if (permission === 'granted') {
      await this.subscribeToPush();
    }
    return permission;
  }

  async subscribeToPush() {
    try {
      const registration = await navigator.serviceWorker.ready;
      
      // Check if already subscribed
      let subscription = await registration.pushManager.getSubscription();
      
      if (!subscription) {
        // Create new subscription
        const publicKey = await this.getVapidPublicKey();
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: this.urlBase64ToUint8Array(publicKey),
        });
      }

      // Send subscription to server. Prefer authService.authenticatedFetch when available
      const body = JSON.stringify({
        endpoint: subscription.endpoint,
        p256dh: btoa(String.fromCharCode(...new Uint8Array(subscription.getKey('p256dh')))),
        auth: btoa(String.fromCharCode(...new Uint8Array(subscription.getKey('auth')))),
      });

      if (this.authService && this.authService.authenticatedFetch) {
        await this.authService.authenticatedFetch(`${API_URL}/push/subscribe`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body,
        });
      } else {
        const token = (this.authService && this.authService.getToken && this.authService.getToken()) || localStorage.getItem('token');
        if (token) {
          await fetch(`${API_URL}/push/subscribe`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              'Authorization': `Bearer ${token}`,
            },
            body,
          });
        }
      }
    } catch (error) {
      console.error('Failed to subscribe to push:', error);
    }
  }

  async getVapidPublicKey() {
    // In production, this should be fetched from the server
    // For now, return a placeholder (you'll need to generate real VAPID keys)
    return 'BEl62iUYgUivxIkv69yViEuiBIa-Ib37J8-fZHX_5CzNqpmqvxT5O5y7L6rPKZp_gQZAYy6Y4g7a3YN8-8X-Y=';
  }

  urlBase64ToUint8Array(base64String) {
    const padding = '='.repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding)
      .replace(/\-/g, '+')
      .replace(/_/g, '/');

    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
      outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
  }

  // Request the server to generate a test push payload and display it via the service worker
  async sendTestNotification() {
    try {
      console.debug('sendTestNotification');

      let res;
      if (this.authService && this.authService.authenticatedFetch) {
        res = await this.authService.authenticatedFetch(`${API_URL}/push/test`, {
          method: 'POST',
        });
      } else {
        const token = (this.authService && this.authService.getToken && this.authService.getToken()) || localStorage.getItem('token');
        res = await fetch(`${API_URL}/push/test`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': token ? `Bearer ${token}` : '',
          },
        });
      }

      if (!res.ok) {
        throw new Error('Failed to request test notification');
      }

      const payload = await res.json();
      const registration = await navigator.serviceWorker.ready;

      const title = payload.title || 'Kept — Test Reminder';
      const options = {
        body: payload.body || 'This is a test reminder about one of your promises.',
        tag: 'kept-test',
        icon: '/Static/logos/Kept Mascot Colored.svg',
        badge: '/Static/logos/Kept Mascot Colored.svg',
        data: payload.data || {},
      };

      await registration.showNotification(title, options);
    } catch (error) {
      console.error('sendTestNotification failed:', error);
      throw error;
    }
  }
}

// No default singleton exported to avoid early initialization without AuthService
