const API_URL = '/api';

export class PromiseService {
  constructor() {
    this.authService = null;
  }

  setAuthService(authService) {
    this.authService = authService;
  }

  async createPromise(data) {
    const response = await fetch(`${API_URL}/promises/`, {
      method: 'POST',
      headers: this.authService.getHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to create promise');
    }

    return response.json();
  }

  async getPromises(state = null) {
    const url = state 
      ? `${API_URL}/promises/?state=${state}`
      : `${API_URL}/promises/`;

    const response = await fetch(url, {
      headers: this.authService.getHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch promises');
    }

    return response.json();
  }

  async getPromise(id) {
    const response = await fetch(`${API_URL}/promises/${id}`, {
      headers: this.authService.getHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch promise');
    }

    return response.json();
  }

  async updatePromiseState(id, data) {
    const response = await fetch(`${API_URL}/promises/${id}/state`, {
      method: 'PUT',
      headers: this.authService.getHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update promise');
    }

    return response.json();
  }

  async updatePromise(id, data) {
    const response = await fetch(`${API_URL}/promises/${id}`, {
      method: 'PUT',
      headers: this.authService.getHeaders(),
      body: JSON.stringify(data),
    });

    if (!response.ok) {
      const error = await response.json();
      throw new Error(error.error || 'Failed to update promise');
    }

    return response.json();
  }

  async deletePromise(id) {
    const response = await fetch(`${API_URL}/promises/${id}`, {
      method: 'DELETE',
      headers: this.authService.getHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to delete promise');
    }

    return response.json();
  }

  async getTimeline() {
    const response = await fetch(`${API_URL}/timeline`, {
      headers: this.authService.getHeaders(),
    });

    if (!response.ok) {
      throw new Error('Failed to fetch timeline');
    }

    return response.json();
  }

  async createReminder(promiseId, offsetMinutes) {
    const response = await fetch(`${API_URL}/reminders/promise/${promiseId}`, {
      method: 'POST',
      headers: this.authService.getHeaders(),
      body: JSON.stringify({ offset_minutes: offsetMinutes }),
    });

    if (!response.ok) {
      throw new Error('Failed to create reminder');
    }

    return response.json();
  }
}
