import { AuthService } from './services/auth.js';
import { PromiseService } from './services/promises.js';
import { UIManager } from './ui/manager.js';
import { NotificationService } from './services/notifications.js';

class App {
  constructor() {
    this.auth = new AuthService();
    this.promises = new PromiseService();
    this.notifications = new NotificationService(this.auth);
    this.ui = new UIManager(this.auth, this.promises, this.notifications);
  }

  async init() {
    // Initialize theme
    this.initTheme();

    // Fetch configuration from backend
    try {
      const configResponse = await fetch('/api/config');
      const config = await configResponse.json();
      this.ui.setConfig(config);
    } catch (error) {
      console.warn('Failed to fetch config:', error);
    }

    // Check if user is logged in
    const user = this.auth.getUser();
    if (user) {
      try {
        // Attempt to refresh token to see if session is still valid
        await this.auth.refreshToken();
        // If successful, show main screen
        this.ui.showMainScreen();
        await this.ui.loadTimeline();
      } catch (error) {
        console.log('Session expired or invalid, showing login');
        this.ui.showAuthScreen();
        this.ui.showLoginForm();
      }
    } else {
      // No user session, show the login/auth screen
      this.ui.showAuthScreen();
      this.ui.showLoginForm();
    }

    // Register service worker
    if ('serviceWorker' in navigator) {
      try {
        await navigator.serviceWorker.register('/sw.js');
        console.log('Service Worker registered');
        if (this.notifications && this.notifications.checkPermission) {
          try {
            await this.notifications.checkPermission();
          } catch (e) {
            console.warn('Notification check failed:', e);
          }
        }
      } catch (error) {
        console.error('Service Worker registration failed:', error);
      }
    }
  }

  initTheme() {
    // Get saved theme from localStorage or default to light
    const savedTheme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', savedTheme);

    // Add theme toggle event listener
    const themeToggle = document.getElementById('theme-toggle');
    if (themeToggle) {
      themeToggle.addEventListener('click', () => {
        const currentTheme = document.documentElement.getAttribute('data-theme');
        const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
        document.documentElement.setAttribute('data-theme', newTheme);
        localStorage.setItem('theme', newTheme);
      });
    }
  }
}

// Initialize app
const app = new App();
app.init();
