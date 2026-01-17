const API_URL = '/api';

export class AuthService {
  constructor() {
    // Access token stored in memory only (not localStorage for security)
    this.token = null;
    this.user = JSON.parse(localStorage.getItem('user') || 'null');
    this.refreshPromise = null;
  }

  getToken() {
    return this.token;
  }

  getUser() {
    return this.user;
  }

  async register(username, password) {
    const response = await fetch(`${API_URL}/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include', // Include cookies
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Registration failed');
    }

    const data = await response.json();
    this.token = data.token;
    this.user = data.user;
    localStorage.setItem('user', JSON.stringify(data.user));
    return data;
  }

  async login(username, password) {
    const response = await fetch(`${API_URL}/auth/login`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      credentials: 'include', // Include cookies
      body: JSON.stringify({ username, password }),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Login failed');
    }

    const data = await response.json();
    this.token = data.token;
    this.user = data.user;
    localStorage.setItem('user', JSON.stringify(data.user));
    return data;
  }

  async logout() {
    try {
      await fetch(`${API_URL}/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      });
    } catch (error) {
      console.error('Logout request failed:', error);
    }
    
    this.token = null;
    this.user = null;
    localStorage.removeItem('user');
  }

  async refreshToken() {
    // Prevent multiple simultaneous refresh requests
    if (this.refreshPromise) {
      return this.refreshPromise;
    }

    this.refreshPromise = fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      credentials: 'include', // Send refresh token cookie
    })
      .then(async (response) => {
        if (!response.ok) {
          throw new Error('Failed to refresh token');
        }
        const data = await response.json();
        this.token = data.token;
        return data.token;
      })
      .finally(() => {
        this.refreshPromise = null;
      });

    return this.refreshPromise;
  }

  getHeaders() {
    return {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${this.token}`,
    };
  }

  // Helper method to make authenticated requests with automatic token refresh
  async authenticatedFetch(url, options = {}) {
    // Ensure we have a token
    if (!this.token) {
      try {
        await this.refreshToken();
      } catch (error) {
        throw new Error('Authentication required');
      }
    }

    // Make the request
    const response = await fetch(url, {
      ...options,
      headers: {
        ...options.headers,
        'Authorization': `Bearer ${this.token}`,
      },
      credentials: 'include',
    });

    // If unauthorized, try to refresh token once and retry
    if (response.status === 401) {
      try {
        await this.refreshToken();
        // Retry the original request with new token
        return fetch(url, {
          ...options,
          headers: {
            ...options.headers,
            'Authorization': `Bearer ${this.token}`,
          },
          credentials: 'include',
        });
      } catch (error) {
        // Refresh failed, user needs to log in again
        this.token = null;
        this.user = null;
        localStorage.removeItem('user');
        throw new Error('Session expired. Please log in again.');
      }
    }

    return response;
  }
}
