const API_URL = '/api';

export class NotificationService {
  // Accept optional authService to access token and authenticatedFetch
  constructor(authService) {
    this.authService = authService;
    this.vapidPublicKey = null;
    this.init();
  }

  async init() {
    if (!('Notification' in window)) {
      console.error('Notifications not supported in this browser');
      return;
    }

    if (!('serviceWorker' in navigator)) {
      console.error('Service Workers are not supported in this browser');
      return;
    }

    // Register Service Worker first
    try {
      await this.registerServiceWorker();
    } catch (error) {
      console.warn('Failed to register Service Worker:', error);
    }

    // Fetch VAPID public key from server
    try {
      await this.fetchVapidPublicKey();
    } catch (error) {
      console.error('Error fetching VAPID key');
    }
  }

  async registerServiceWorker() {
    try {
      const registration = await navigator.serviceWorker.register('/sw.js', {
        scope: '/',
      });
      // Wait for it to be ready
      await navigator.serviceWorker.ready;
      return registration;
    } catch (error) {
      console.error('Failed to register Service Worker');
      throw error;
    }
  }

  async fetchVapidPublicKey() {
    try {
      const response = await fetch(`${API_URL}/push/vapid-public-key`);
      if (!response.ok) {
        throw new Error('VAPID key not available');
      }
      const data = await response.json();
      this.vapidPublicKey = data.publicKey;
      return this.vapidPublicKey;
    } catch (error) {
      console.error('Push notifications not configured on server');
      return null;
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
      // Always unsubscribe from any existing subscription to avoid mismatches
      let oldSubscription = await registration.pushManager.getSubscription();
      if (oldSubscription) {
        await oldSubscription.unsubscribe();
      }
      // Ensure we have the VAPID key
      if (!this.vapidPublicKey) {
        await this.fetchVapidPublicKey();
      }
      if (!this.vapidPublicKey) {
        console.error('Cannot subscribe: VAPID public key not available');
        return null;
      }
      // Create new subscription
        const subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: this.urlBase64ToUint8Array(this.vapidPublicKey),
        });
      // Send subscription to server. Prefer authService.authenticatedFetch when available
      const p256dh = this.arrayBufferToUrlBase64(subscription.getKey('p256dh'));
      const auth = this.arrayBufferToUrlBase64(subscription.getKey('auth'));
      
      const body = JSON.stringify({
        endpoint: subscription.endpoint,
        p256dh,
        auth,
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
      
      console.log('Subscribed to push notifications');
      return subscription;
    } catch (error) {
      console.error('Failed to subscribe to push');
      return null;
    }
  }

  async getVapidPublicKey() {
    if (this.vapidPublicKey) {
      return this.vapidPublicKey;
    }
    return await this.fetchVapidPublicKey();
  }

  arrayBufferToUrlBase64(buffer) {
    const binary = String.fromCharCode(...new Uint8Array(buffer));
    return window.btoa(binary)
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '');
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

  // Request the server to send a test push notification
  // The server will send an actual push notification to this device
  async sendTestNotification() {
    try {
      // First ensure we're subscribed to push
      const subscription = await this.subscribeToPush();
      if (!subscription) {
        // Fall back to local notification if push subscription failed
        const registration = await navigator.serviceWorker.ready;
        await registration.showNotification('Kept — Test', {
          body: 'Push subscription not available. Please check your notification settings.',
          icon: '/Static/logos/Kept Mascot Colored.svg',
          badge: '/Static/logos/Kept Mascot Colored.svg',
          tag: 'kept-test-local',
        });
        return;
      }

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
        const errorData = await res.json().catch(() => ({}));
        // If 500, might be VAPID mismatch - retry by unsubscribing and re-subscribing
        if (res.status === 500 && errorData.error && errorData.error.includes('push notifications')) {
          console.error('Push failed; clearing subscription to retry');
          const registration = await navigator.serviceWorker.ready;
          const existingSub = await registration.pushManager.getSubscription();
          if (existingSub) {
            await existingSub.unsubscribe();
          }
        }
        throw new Error(errorData.error || 'Failed to send test notification');
      }

      // Server sends the push notification directly
      const result = await res.json();

      // Also show a local notification to indicate success
      // (the server-sent notification will arrive asynchronously via Service Worker)
      const registration = await navigator.serviceWorker.ready;
      await registration.showNotification('Kept — Test', {
        body: 'This is a test notification',
        icon: '/Static/logos/Kept Mascot Colored.svg',
        badge: '/Static/logos/Kept Mascot Colored.svg',
        tag: 'kept-test-local',
      });
    } catch (error) {
      console.error('Failed to send test notification');
      // Show error as local notification
      try {
        const registration = await navigator.serviceWorker.ready;
        await registration.showNotification('Kept — Notification Error', {
          body: error.message || 'Failed to send test notification',
          icon: '/Static/logos/Kept Mascot Colored.svg',
          badge: '/Static/logos/Kept Mascot Colored.svg',
          tag: 'kept-test-error',
        });
      } catch (e) {
        console.error('Failed to show error notification');
      }
      throw error;
    }
  }
}

// No default singleton exported to avoid early initialization without AuthService
