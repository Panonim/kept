import { describe, it, expect, beforeEach } from 'vitest';
import { UIManager } from '../ui/manager.js';

class DummyAuth {
  constructor() {
    this.token = null;
    this.user = null;
  }
  login() { return Promise.resolve(); }
  register() { return Promise.resolve(); }
  logout() { this.token = null; this.user = null; }
}

class DummyPromises {
  setAuthService() {}
}

describe('UIManager logout', () => {
  beforeEach(() => {
    document.body.innerHTML = `
      <div id="auth-screen">
        <form id="login-form-element">
          <input id="login-username" value="foo" />
          <input id="login-password" value="secret" />
        </form>
        <form id="register-form-element">
          <input id="register-username" value="bar" />
          <input id="register-password" value="secret2" />
        </form>
      </div>
      <div id="main-screen"></div>
    `;
  });

  it('clears auth inputs on logout', () => {
    const auth = new DummyAuth();
    const promises = new DummyPromises();
    const manager = new UIManager(auth, promises);

    manager.handleLogout();

    expect(document.getElementById('login-username').value).toBe('');
    expect(document.getElementById('login-password').value).toBe('');
    expect(document.getElementById('register-username').value).toBe('');
    expect(document.getElementById('register-password').value).toBe('');
  });
});

describe('Due date display', () => {
  function isoDaysAgo(n) { const d = new Date(); d.setDate(d.getDate() - n); return d.toISOString(); }

  it('omits "due yesterday" on timeline but shows in detail', async () => {
    document.body.innerHTML = `
      <div id="timeline-content"></div>
      <div id="detail-content"></div>
      <div id="timeline-view" class="active"></div>
      <div id="detail-view"></div>
    `;

    const yesterdayPromise = {
      id: 1,
      recipient: 'Alice',
      description: 'Test',
      created_at: new Date().toISOString(),
      due_date: isoDaysAgo(1),
      current_state: 'active',
      events: []
    };

    class DummyPromises2 {
      setAuthService() {}
      getPromises() { return Promise.resolve([yesterdayPromise]); }
      getPromise(id) { return Promise.resolve(yesterdayPromise); }
    }

    const auth = new DummyAuth();
    const promises = new DummyPromises2();
    const manager = new UIManager(auth, promises);

    // Render timeline
    await manager.loadTimeline();
    const timelineHtml = document.getElementById('timeline-content').innerHTML;
    expect(timelineHtml).not.toContain('due yesterday');

    // Show detail
    await manager.showPromiseDetail(1);
    const detailHtml = document.getElementById('detail-content').innerHTML;
    expect(detailHtml).toContain('Due yesterday');
  });
});
