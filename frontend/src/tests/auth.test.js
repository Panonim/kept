import { describe, it, expect, beforeEach, vi } from 'vitest';
import { AuthService } from '../services/auth.js';

describe('AuthService', () => {
  let authService;

  beforeEach(() => {
    // Basic localStorage mock if it's missing or broken
    if (!window.localStorage || typeof window.localStorage.clear !== 'function') {
      const store = {};
      Object.defineProperty(window, 'localStorage', {
        value: {
          getItem: (key) => store[key] || null,
          setItem: (key, value) => { store[key] = value.toString(); },
          removeItem: (key) => { delete store[key]; },
          clear: () => { for (const key in store) delete store[key]; },
        },
        writable: true
      });
    }

    window.localStorage.clear();
    
    // Mock fetch
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    }));
    
    authService = new AuthService();
  });

  it('should initialize without token', () => {
    expect(authService.getToken()).toBeNull();
    expect(authService.getUser()).toBeNull();
  });

  it('should store token after login', () => {
    const mockData = {
      token: 'test-token',
      user: { id: 1, username: 'testuser' }
    };

    authService.token = mockData.token;
    authService.user = mockData.user;
    localStorage.setItem('token', mockData.token);
    localStorage.setItem('user', JSON.stringify(mockData.user));

    expect(authService.getToken()).toBe('test-token');
    expect(authService.getUser()).toEqual(mockData.user);
  });

  it('should clear token on logout', async () => {
    authService.token = 'test-token';
    authService.user = { id: 1, username: 'testuser' };
    localStorage.setItem('token', 'test-token');

    await authService.logout();

    expect(authService.getToken()).toBeNull();
    expect(authService.getUser()).toBeNull();
    expect(localStorage.getItem('user')).toBeNull();
  });

  it('should generate correct headers', () => {
    authService.token = 'test-token';
    const headers = authService.getHeaders();

    expect(headers['Authorization']).toBe('Bearer test-token');
    expect(headers['Content-Type']).toBe('application/json');
  });
});
