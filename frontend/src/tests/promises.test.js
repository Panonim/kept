import { describe, it, expect, beforeEach } from 'vitest';
import { PromiseService } from '../services/promises.js';

describe('PromiseService', () => {
  let promiseService;
  let mockAuthService;

  beforeEach(() => {
    promiseService = new PromiseService();
    mockAuthService = {
      getHeaders: () => ({
        'Content-Type': 'application/json',
        'Authorization': 'Bearer test-token'
      })
    };
    promiseService.setAuthService(mockAuthService);
  });

  it('should set auth service', () => {
    expect(promiseService.authService).toBe(mockAuthService);
  });

  it('should have methods for promise operations', () => {
    expect(typeof promiseService.createPromise).toBe('function');
    expect(typeof promiseService.getPromises).toBe('function');
    expect(typeof promiseService.getPromise).toBe('function');
    expect(typeof promiseService.updatePromiseState).toBe('function');
    expect(typeof promiseService.deletePromise).toBe('function');
    expect(typeof promiseService.getTimeline).toBe('function');
  });
});
