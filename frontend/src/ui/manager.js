import Chart from 'chart.js/auto';

export class UIManager {
  constructor(authService, promiseService, notificationService) {
    this.auth = authService;
    this.promises = promiseService;
    this.notifications = notificationService;
    this.promises.setAuthService(authService);
    
    this.currentFilter = 'all';
    this.currentPromiseId = null;
    this.selectedState = null;
    this.chartAnimated = false; // Track if chart has been animated
    this.config = { disableRegistration: false }; // Default config

    this.initEventListeners();
    this.initCustomSelects();
  }

  setConfig(config) {
    this.config = config;
    this.applyConfigToUI();
  }

  applyConfigToUI() {
    // Hide registration link if registration is disabled
    const authSwitch = document.querySelector('.auth-switch');
    if (authSwitch && this.config.disableRegistration) {
      authSwitch.style.display = 'none';
      // Also disable the register form toggle
      document.getElementById('show-register')?.removeEventListener('click', this.showRegisterForm);
    }
  }

  initCustomSelects() {
    document.querySelectorAll('.custom-select').forEach(select => {
      const trigger = select.querySelector('.custom-select-trigger');
      const options = select.querySelector('.custom-select-options');
      const optionItems = select.querySelectorAll('.custom-option');
      const hiddenInput = select.parentElement.querySelector('input[type="hidden"]');
      const valueSpan = select.querySelector('.custom-select-value');

      // Toggle open
      trigger.addEventListener('click', (e) => {
        e.stopPropagation(); // prevent window click from closing immediately
        // Close others
        document.querySelectorAll('.custom-select.open').forEach(s => {
          if (s !== select) s.classList.remove('open');
        });
        select.classList.toggle('open');
      });

      // Handle selection
      optionItems.forEach(option => {
        option.addEventListener('click', (e) => {
          e.stopPropagation();
          const value = option.dataset.value;
          const text = option.textContent;

          // Update state
          hiddenInput.value = value;
          valueSpan.textContent = text;
          
          // Update visual selected state
          optionItems.forEach(opt => opt.classList.remove('selected'));
          option.classList.add('selected');

          // Close dropdown
          select.classList.remove('open');
        });
      });
    });

    // Close on click outside
    window.addEventListener('click', (e) => {
      document.querySelectorAll('.custom-select.open').forEach(select => {
          select.classList.remove('open');
      });
    });
  }

  initEventListeners() {
    // Auth form toggles
    document.getElementById('show-register')?.addEventListener('click', (e) => {
      e.preventDefault();
      this.showRegisterForm();
    });

    document.getElementById('show-login')?.addEventListener('click', (e) => {
      e.preventDefault();
      this.showLoginForm();
    });

    // Auth form submissions
    document.getElementById('login-form-element')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.handleLogin();
    });

    document.getElementById('register-form-element')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.handleRegister();
    });

    // Logout
    document.getElementById('logout-btn')?.addEventListener('click', () => {
      this.handleLogout();
    });

    // Notification bell
    document.getElementById('notification-bell')?.addEventListener('click', () => {
      this.showNotificationSettings();
    });

    // Notification settings form
    document.getElementById('notification-settings-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.handleSaveNotificationSettings();
    });

    // Enable push notifications button
    document.getElementById('enable-push-btn')?.addEventListener('click', async () => {
      await this.handleEnablePushNotifications();
    });

    // Filter tabs
    document.querySelectorAll('.filter-tab').forEach(tab => {
      tab.addEventListener('click', () => {
        this.handleFilterChange(tab.dataset.filter);
      });
    });

    // Create promise button
    document.getElementById('create-promise-btn')?.addEventListener('click', () => {
      this.showPromiseModal();
    });

    // Promise form
    document.getElementById('promise-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.handleCreatePromise();
    });

    // Resolve form
    document.getElementById('resolve-form')?.addEventListener('submit', (e) => {
      e.preventDefault();
      this.handleResolvePromise();
    });

    // State buttons in resolve modal
    document.querySelectorAll('.state-btn').forEach(btn => {
      btn.addEventListener('click', () => {
        this.handleStateSelection(btn.dataset.state);
      });
    });

    // Back to timeline
    document.getElementById('back-to-timeline')?.addEventListener('click', () => {
      this.showTimelineView();
    });

    // Modal close buttons
    document.querySelectorAll('.modal-close, .modal-overlay').forEach(el => {
      el.addEventListener('click', () => {
        this.closeModals();
      });
    });
  }

  showAuthScreen() {
    document.getElementById('auth-screen')?.classList.remove('hidden');
    document.getElementById('main-screen')?.classList.add('hidden');

    // Reset auth forms so no credentials remain visible
    document.getElementById('login-form-element')?.reset();
    document.getElementById('register-form-element')?.reset();
    this.clearAuthError();
  }

  showMainScreen() {
    document.getElementById('auth-screen')?.classList.add('hidden');
    document.getElementById('main-screen')?.classList.remove('hidden');
  }

  showLoginForm() {
    document.getElementById('login-form')?.classList.add('active');
    document.getElementById('register-form')?.classList.remove('active');
    this.clearAuthError();
  }

  showRegisterForm() {
    document.getElementById('register-form')?.classList.add('active');
    document.getElementById('login-form')?.classList.remove('active');
    this.clearAuthError();
  }

  showAuthError(message) {
    const errorEl = document.getElementById('auth-error');
    if (errorEl) {
      errorEl.textContent = message;
      errorEl.classList.add('show');
    }
  }

  clearAuthError() {
    const errorEl = document.getElementById('auth-error');
    if (errorEl) {
      errorEl.textContent = '';
      errorEl.classList.remove('show');
    }
  }

  showNotification(message, type = 'info', duration = 3000) {
    const notifEl = document.getElementById('notification-message');
    if (notifEl) {
      notifEl.textContent = message;
      notifEl.className = `notification-message show ${type}`;
      
      // Auto-hide after duration
      setTimeout(() => {
        notifEl.classList.remove('show');
      }, duration);
    }
  }

  async handleNotificationPermission() {
    // This method is deprecated - now using showNotificationSettings
    this.showNotificationSettings();
  }

  showNotificationSettings() {
    const modal = document.getElementById('notification-settings-modal');
    const enablePushBtn = document.getElementById('enable-push-btn');
    const emailInput = document.getElementById('user-email');

    // Update push notification status
    if ('Notification' in window && Notification.permission === 'granted') {
      enablePushBtn.textContent = '✓ Enabled';
      enablePushBtn.classList.add('enabled');
      enablePushBtn.disabled = true;
    } else {
      enablePushBtn.textContent = 'Enable Notifications';
      enablePushBtn.classList.remove('enabled');
      enablePushBtn.disabled = false;
    }

    // Load user's current email
    const user = this.auth.getUser();
    if (user && user.email) {
      emailInput.value = user.email;
    }

    modal.classList.remove('hidden');
  }

  async handleEnablePushNotifications() {
    if (!this.notifications) {
      this.showNotification('Notifications Error: Service not available', 'error', 4000);
      return;
    }

    try {
      const permission = await this.notifications.requestPermission();
      
      if (permission === 'granted') {
        await this.notifications.sendTestNotification();
        this.showNotification('✓ Push Notifications Enabled', 'success', 3000);
        
        // Update UI
        const enablePushBtn = document.getElementById('enable-push-btn');
        enablePushBtn.textContent = '✓ Enabled';
        enablePushBtn.classList.add('enabled');
        enablePushBtn.disabled = true;
        enablePushBtn.disabled = true;
      } else if (permission === 'denied') {
        this.showNotification('Notifications Error: Permission Denied', 'error', 4000);
      }
    } catch (error) {
      console.error('Error enabling push notifications:', error);
      this.showNotification(`Notifications Error: ${error.message}`, 'error', 4000);
    }
  }

  async handleSaveNotificationSettings() {
    const emailInput = document.getElementById('user-email');
    const email = emailInput.value.trim();

    try {
      // Update email via API
      const response = await fetch('/api/user/email', {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${this.auth.getToken()}`
        },
        body: JSON.stringify({ email: email || null })
      });

      if (!response.ok) {
        throw new Error('Failed to update email');
      }

      const data = await response.json();
      
      // Update local user data
      const user = this.auth.getUser();
      if (user) {
        user.email = email;
        localStorage.setItem('user', JSON.stringify(user));
      }

      this.showNotification('✓ Settings saved successfully', 'success', 3000);
      this.closeModals();
    } catch (error) {
      console.error('Error saving notification settings:', error);
      this.showNotification(`Error: ${error.message}`, 'error', 4000);
    }
  }

  async handleLogin() {
    const username = document.getElementById('login-username').value;
    const password = document.getElementById('login-password').value;

    try {
      await this.auth.login(username, password);
      this.showMainScreen();
      await this.loadTimeline();
    } catch (error) {
      this.showAuthError(error.message);
    }
  }

  async handleRegister() {
    const username = document.getElementById('register-username').value;
    const password = document.getElementById('register-password').value;

    try {
      await this.auth.register(username, password);
      this.showMainScreen();
      await this.loadTimeline();
    } catch (error) {
      this.showAuthError(error.message);
    }
  }

  handleLogout() {
    this.auth.logout();

    // Ensure all auth inputs are cleared when logging out
    document.getElementById('login-form-element')?.reset();
    document.getElementById('register-form-element')?.reset();

    this.showAuthScreen();
    // Always show the login form after logout (avoid leaving register visible)
    this.showLoginForm();
    this.clearAuthError();
  }

  async handleFilterChange(filter) {
    this.currentFilter = filter;
    
    // Update active tab
    document.querySelectorAll('.filter-tab').forEach(tab => {
      tab.classList.toggle('active', tab.dataset.filter === filter);
    });

    await this.loadTimeline();
  }

  async loadTimeline() {
    try {
      const state = this.currentFilter === 'all' ? null : this.currentFilter;
      const promises = await this.promises.getPromises(state);
      this.renderStats(promises);
      this.renderTimeline(promises);
    } catch (error) {
      console.error('Failed to load timeline:', error);
    }
  }

  renderStats(promises) {
    const container = document.getElementById('stats-graph');
    if (!container) return;

    // Get all promises (not filtered) to calculate overall stats
    this.promises.getPromises(null).then(allPromises => {
      const keptCount = allPromises.filter(p => p.current_state === 'kept').length;
      const brokenCount = allPromises.filter(p => p.current_state === 'broken').length;
      const activeCount = allPromises.filter(p => p.current_state === 'active').length;
      const total = allPromises.length;

      // Calculate percentages
      const keptPercent = total > 0 ? (keptCount / total) * 100 : 0;
      const brokenPercent = total > 0 ? (brokenCount / total) * 100 : 0;
      const activePercent = total > 0 ? (activeCount / total) * 100 : 0;

      // Calculate success rate
      const resolvedTotal = keptCount + brokenCount;
      const successRate = resolvedTotal > 0 ? Math.round((keptCount / resolvedTotal) * 100) : 0;

      container.innerHTML = `
        <div class="stats-container">
          <div class="stat-box stat-kept">
            <div class="stat-number">${keptCount}</div>
            <div class="stat-label">Kept</div>
            <div class="stat-bar">
              <div class="stat-fill" style="width: ${keptPercent}%"></div>
            </div>
          </div>
          <div class="stat-box stat-active">
            <div class="stat-number">${activeCount}</div>
            <div class="stat-label">Active</div>
            <div class="stat-bar">
              <div class="stat-fill" style="width: ${activePercent}%"></div>
            </div>
          </div>
          <div class="stat-box stat-broken">
            <div class="stat-number">${brokenCount}</div>
            <div class="stat-label">Broken</div>
            <div class="stat-bar">
              <div class="stat-fill" style="width: ${brokenPercent}%"></div>
            </div>
          </div>
          <div class="stat-box stat-success">
            <div class="stat-number">${successRate}%</div>
            <div class="stat-label">Success Rate</div>
            <div class="stat-motivation">${this.getMotivationalMessage(successRate, keptCount, brokenCount)}</div>
          </div>
        </div>
      `;

      // Add a small line chart showing kept vs broken trend (last 8 weeks)
      const chartHtml = `<div class="chart-wrap"><canvas id="stats-line-chart" height="120"></canvas></div>`;
      container.insertAdjacentHTML('beforeend', chartHtml);

      // Destroy previous chart if present
      try {
        if (this.statsChart) this.statsChart.destroy();

        const ctx = container.querySelector('#stats-line-chart')?.getContext('2d');
        if (ctx) {
          const weeks = 8;
          const oneWeek = 7 * 24 * 60 * 60 * 1000;
          const now = new Date();
          const labels = [];
          for (let i = weeks - 1; i >= 0; i--) {
            const d = new Date(now.getTime() - i * oneWeek);
            labels.push(d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' }));
          }

          const keptByWeek = new Array(weeks).fill(0);
          const brokenByWeek = new Array(weeks).fill(0);

          allPromises.forEach(p => {
            if (!p.created_at) return;
            const created = new Date(p.created_at);
            const diff = now - created;
            const idx = Math.floor(diff / oneWeek);
            if (idx >= 0 && idx < weeks) {
              const arrayIndex = weeks - 1 - idx; // newest at end of labels
              if (p.current_state === 'kept') keptByWeek[arrayIndex] += 1;
              if (p.current_state === 'broken') brokenByWeek[arrayIndex] += 1;
            }
          });

          this.statsChart = new Chart(ctx, {
            type: 'line',
            data: {
              labels,
              datasets: [
                {
                  label: 'Kept',
                  data: keptByWeek,
                  borderColor: '#426142',
                  tension: 0.4,
                  fill: false,
                  pointRadius: 0,
                  borderWidth: 3,
                  pointHoverRadius: 0,
                },
                {
                  label: 'Broken',
                  data: brokenByWeek,
                  borderColor: '#d9534f',
                  tension: 0.4,
                  fill: false,
                  pointRadius: 0,
                  borderWidth: 3,
                  pointHoverRadius: 0,
                }
              ]
            },
            options: {
              responsive: true,
              maintainAspectRatio: false,
              // Disable Chart.js built-in animation - we'll use a CSS mask instead
              animation: false,
              plugins: {
                legend: { display: false },
                tooltip: { enabled: false }
              },
              scales: {
                x: {
                  display: false,
                  grid: { display: false, drawBorder: false },
                  ticks: { display: false }
                },
                y: {
                  display: false,
                  beginAtZero: true,
                  ticks: { display: false },
                  grid: { display: false, drawBorder: false }
                }
              }
            }
          });

        // Reveal animation: wrap the canvas so mask aligns exactly with drawn area
        try {
          const canvas = container.querySelector('#stats-line-chart');
          if (canvas && !this.chartAnimated) {
            // Only animate on first render, not on filter changes
            this.chartAnimated = true;
            
            // create a wrapper that matches canvas size and position
            const canvasWrap = document.createElement('div');
            canvasWrap.className = 'chart-canvas-wrap';
            canvasWrap.style.position = 'relative';
            canvasWrap.style.width = '100%';
            canvasWrap.style.height = '100%';

            canvas.parentNode.insertBefore(canvasWrap, canvas);
            canvasWrap.appendChild(canvas);

            // create mask on top of canvas anchored to the right so it reveals left->right
            const mask = document.createElement('div');
            mask.className = 'chart-mask';
            mask.style.width = '100%';
            mask.style.transition = 'width 600ms ease-out';
            // anchor mask to the right so shrinking width uncovers from left to right
            mask.style.right = '0';
            mask.style.left = 'auto';
            canvasWrap.appendChild(mask);

            // start reveal on next frame
            requestAnimationFrame(() => requestAnimationFrame(() => {
              mask.style.width = '0%';
            }));

            mask.addEventListener('transitionend', () => {
              try { mask.remove(); } catch (e) { /* ignore */ }
            }, { once: true });
          }
        } catch (err) {
          console.error('Failed to run reveal animation:', err);
        }
        }
      } catch (err) {
        console.error('Failed to render stats chart:', err);
      }
    }).catch(error => {
      console.error('Failed to load stats:', error);
    });
  }

  getMotivationalMessage(successRate, keptCount, brokenCount) {
    const total = keptCount + brokenCount;
    
    if (total === 0) {
      return "Start making promises!";
    }
    
    if (successRate === 100) {
      return "Perfect! Keep it up! 🎉";
    }
    
    if (successRate >= 80) {
      return "Excellent work! 🌟";
    }
    
    if (successRate >= 60) {
      return "Good progress! 💪";
    }
    
    if (successRate >= 40) {
      return "Keep improving! 📈";
    }
    
    return "You can do better! 💫";
  }

  renderTimeline(promises) {
    const container = document.getElementById('timeline-content');
    if (!container) return;

    // If empty, show empty state (only update if needed)
    if (promises.length === 0) {
      if (!container.querySelector('.empty-state')) {
        container.innerHTML = `
          <div class="empty-state">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 0 0 2.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 0 0-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 0 0 .75-.75 2.25 2.25 0 0 0-.1-.664m-5.8 0A2.251 2.251 0 0 1 13.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25ZM6.75 12h.008v.008H6.75V12Zm0 3h.008v.008H6.75V15Zm0 3h.008v.008H6.75V18Z" />
            </svg>
            <h3>No promises yet</h3>
            <p>${this.currentFilter === 'all' ? 'Create your first promise' : `No ${this.currentFilter} promises`}</p>
          </div>
        `;
      }
      return;
    }

    // Use a keyed diff for cards
    const existingCards = Array.from(container.querySelectorAll('.promise-card'));
    const existingIds = new Set(existingCards.map(card => card.dataset.id));
    const newIds = new Set(promises.map(p => String(p.id)));

    // Remove cards that are no longer present
    existingCards.forEach(card => {
      if (!newIds.has(card.dataset.id)) {
        card.remove();
      }
    });

    // Add or update cards in order
    let prev = null;
    promises.forEach(promise => {
      const id = String(promise.id);
      let card = container.querySelector(`.promise-card[data-id="${id}"]`);
      const cardHtml = this.renderPromiseCard(promise);
      if (!card) {
        // Create new card
        const temp = document.createElement('div');
        temp.innerHTML = cardHtml;
        card = temp.firstElementChild;
        card.classList.add('card-appear');
        // Insert in correct order
        if (prev) {
          prev.after(card);
        } else {
          container.prepend(card);
        }
        // Remove appear class after animation
        setTimeout(() => card.classList.remove('card-appear'), 300);
      } else {
        // Update card if changed
        if (card.outerHTML !== cardHtml) {
          const temp = document.createElement('div');
          temp.innerHTML = cardHtml;
          const newCard = temp.firstElementChild;
          card.replaceWith(newCard);
          card = newCard;
        }
      }
      // Add click listener
      card.onclick = () => this.showPromiseDetail(parseInt(card.dataset.id));
      prev = card;
    });

    // Remove empty state if present
    const empty = container.querySelector('.empty-state');
    if (empty) empty.remove();
  }

  renderPromiseCard(promise) {
    const dueDate = promise.due_date ? new Date(promise.due_date) : null;
    const createdDate = new Date(promise.created_at);
    const dueLabel = dueDate ? this.formatDate(dueDate) : null;
    const dueSpan = (dueLabel && dueLabel !== 'yesterday') ? `<span class="promise-date">due ${dueLabel}</span>` : '';

    return `
      <div class="promise-card" data-id="${promise.id}">
        <div class="promise-card-header">
          <span class="promise-recipient">to ${this.escapeHtml(promise.recipient)}</span>
          <span class="promise-state ${promise.current_state}">${this.stateLabel(promise.current_state)}</span>
        </div>
        <div class="promise-description">${this.escapeHtml(promise.description)}</div>
        <div class="promise-card-footer">
          <span class="promise-date">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6.75 3v2.25M17.25 3v2.25M3 18.75V7.5a2.25 2.25 0 0 1 2.25-2.25h13.5A2.25 2.25 0 0 1 21 7.5v11.25m-18 0A2.25 2.25 0 0 0 5.25 21h13.5A2.25 2.25 0 0 0 21 18.75m-18 0v-7.5A2.25 2.25 0 0 1 5.25 9h13.5A2.25 2.25 0 0 1 21 11.25v7.5" />
            </svg>
            ${this.formatDate(createdDate)}
          </span>
          ${dueSpan}
        </div>
      </div>
    `;
  }

  async showPromiseDetail(id) {
    try {
      const promise = await this.promises.getPromise(id);
      this.currentPromiseId = id;
      this.renderPromiseDetail(promise);
      this.showDetailView();
    } catch (error) {
      console.error('Failed to load promise:', error);
    }
  }

  renderPromiseDetail(promise) {
    const container = document.getElementById('detail-content');
    if (!container) return;

    const dueDate = promise.due_date ? new Date(promise.due_date) : null;
    const createdDate = new Date(promise.created_at);
    const dueLabel = dueDate ? this.formatDate(dueDate) : null;
    const hideDueForKeptYesterday = promise.current_state === 'kept' && dueLabel === 'yesterday';
    const dueHtml = (dueLabel && !hideDueForKeptYesterday) ? `<span>Due ${dueLabel}</span>` : '';

    container.innerHTML = `
      <div class="promise-detail">
        <div class="promise-detail-header">
          <div class="promise-detail-recipient">to ${this.escapeHtml(promise.recipient)}</div>
          <div class="promise-detail-description">${this.escapeHtml(promise.description)}</div>
          <div class="promise-detail-meta">
            <span class="promise-state ${promise.current_state}">${this.stateLabel(promise.current_state)}</span>
            <span>Created ${this.formatDate(createdDate)}</span>
            ${dueHtml}
          </div>
        </div>

        ${promise.events && promise.events.length > 0 ? `
          <div class="promise-detail-section">
            <h3>History</h3>
            <div class="event-timeline">
              ${promise.events.slice().reverse().map(event => this.renderEvent(event)).join('')}
            </div>
          </div>
        ` : ''}

        <div class="promise-actions">
          <button class="btn-secondary" id="resolve-promise-btn" title="Change promise state">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
              <path stroke-linecap="round" stroke-linejoin="round" d="m12.75 15 3-3m0 0-3-3m3 3h-7.5M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
            </svg>
            <span class="btn-label">Change State</span>
          </button>
          <button class="btn-secondary" id="edit-reminder-btn" title="Edit reminder settings">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
              <path stroke-linecap="round" stroke-linejoin="round" d="M3 3v1.5M3 21v-6m0 0 2.77-.693a9 9 0 0 1 6.208.682l.108.054a9 9 0 0 0 6.086.71l3.114-.732a48.524 48.524 0 0 1-.005-10.499l-3.11.732a9 9 0 0 1-6.085-.711l-.108-.054a9 9 0 0 0-6.208-.682L3 4.5M3 15V4.5" />
            </svg>
            <span class="btn-label">Edit Reminder</span>
          </button>
          <button class="btn-danger" id="delete-promise-btn" title="Delete promise">
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
              <path stroke-linecap="round" stroke-linejoin="round" d="m14.74 9-.346 9m-4.788 0L9.26 9m9.968-3.21c.342.052.682.107 1.022.166m-1.022-.165L18.16 19.673a2.25 2.25 0 0 1-2.244 2.077H8.084a2.25 2.25 0 0 1-2.244-2.077L4.772 5.79m14.456 0a48.108 48.108 0 0 0-3.478-.397m-12 .562c.34-.059.68-.114 1.022-.165m0 0a48.11 48.11 0 0 1 3.478-.397m7.5 0v-.916c0-1.18-.91-2.164-2.09-2.201a51.964 51.964 0 0 0-3.32 0c-1.18.037-2.09 1.022-2.09 2.201v.916m7.5 0a48.667 48.667 0 0 0-7.5 0" />
            </svg>
            <span class="btn-label">Delete</span>
          </button>
        </div>
      </div>
    `;

    // Add event listeners
    document.getElementById('resolve-promise-btn')?.addEventListener('click', () => {
      this.showResolveModal();
    });

    document.getElementById('edit-reminder-btn')?.addEventListener('click', () => {
      this.showEditReminderModal(promise);
    });

    document.getElementById('delete-promise-btn')?.addEventListener('click', () => {
      this.showDeleteConfirmModal();
    });
  }

  renderEvent(event) {
    const eventDate = new Date(event.created_at);
    return `
      <div class="event-item">
        <div class="event-marker ${event.state}"></div>
        <div class="event-content">
          <div class="event-state">${event.state}</div>
          ${event.reflection_note ? `<div class="event-note">${this.escapeHtml(event.reflection_note)}</div>` : ''}
          <div class="event-time">${this.formatDateTime(eventDate)}</div>
        </div>
      </div>
    `;
  }

  showTimelineView() {
    document.getElementById('timeline-view')?.classList.add('active');
    document.getElementById('detail-view')?.classList.remove('active');
  }

  showDetailView() {
    document.getElementById('timeline-view')?.classList.remove('active');
    document.getElementById('detail-view')?.classList.add('active');
  }

  showPromiseModal() {
    document.getElementById('promise-modal')?.classList.remove('hidden');
    // Reset form
    document.getElementById('promise-form')?.reset();
    
    // Reset custom select
    const freqSelect = document.getElementById('frequency-select');
    if (freqSelect) {
      const defaultOption = freqSelect.querySelector('[data-value=""]');
      if (defaultOption) defaultOption.click();
    }
  }

  // Return a human-friendly label for a state for display on pills
  stateLabel(state) {
    if (state === 'active') return 'Active';
    return state;
  }

  async showResolveModal() {
    const modal = document.getElementById('resolve-modal');
    if (!modal) return;

    modal.classList.remove('hidden');
    // Reset form
    document.getElementById('resolve-form')?.reset();
    this.selectedState = null;
    document.querySelectorAll('.state-btn').forEach(btn => {
      btn.classList.remove('active');
    });
    document.getElementById('postponed-date-group')?.classList.add('hidden');

    // If current promise is kept, offer "Active" as the first option instead of "Kept"
    try {
      if (!this.currentPromiseId) return;
      const promise = await this.promises.getPromise(this.currentPromiseId);
      const stateBtns = modal.querySelectorAll('.state-btn');
      if (stateBtns && stateBtns.length >= 2) {
        const firstBtn = stateBtns[0];
        const secondBtn = stateBtns[1];

        if (promise.current_state === 'active') {
          // default: left = Kept, right = Broken
          firstBtn.dataset.state = 'kept';
          firstBtn.innerHTML = `
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
            </svg>
            Kept
          `;

          secondBtn.dataset.state = 'broken';
          secondBtn.innerHTML = `
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 0 0 5.636 5.636m12.728 12.728A9 9 0 0 1 5.636 5.636m12.728 12.728L5.636 5.636" />
            </svg>
            Broken
          `;
        } else {
          // When resolved (kept or broken), place Active on the left, and the other state on the right
          firstBtn.dataset.state = 'active';
          firstBtn.innerHTML = `
            <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor" class="size-6">
              <path stroke-linecap="round" stroke-linejoin="round" d="M8.625 12a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H8.25m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0H12m4.125 0a.375.375 0 1 1-.75 0 .375.375 0 0 1 .75 0Zm0 0h-.375M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
            </svg>
            Active
          `;

          // right button becomes the opposite resolved state
          if (promise.current_state === 'kept') {
            secondBtn.dataset.state = 'broken';
            secondBtn.innerHTML = `
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M18.364 18.364A9 9 0 0 0 5.636 5.636m12.728 12.728A9 9 0 0 1 5.636 5.636m12.728 12.728L5.636 5.636" />
              </svg>
              Broken
            `;
          } else {
            secondBtn.dataset.state = 'kept';
            secondBtn.innerHTML = `
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M9 12.75 11.25 15 15 9.75M21 12a9 9 0 1 1-18 0 9 9 0 0 1 18 0Z" />
              </svg>
              Kept
            `;
          }
        }
      }
    } catch (err) {
      console.error('Failed to adjust resolve options:', err);
    }
  }

  showChangeStateModal() {
    // change-state modal removed; use resolve modal for state changes
  }

  async updatePromiseState(newState) {
    if (!this.currentPromiseId) return;

    try {
      await this.promises.updatePromiseState(this.currentPromiseId, {
        state: newState,
      });
      await this.loadTimeline();
      this.showTimelineView();
    } catch (error) {
      alert('Failed to change promise state: ' + error.message);
    }
  }

  showDeleteConfirmModal() {
    const modalId = 'delete-confirm-modal';
    let modal = document.getElementById(modalId);

    if (!modal) {
      modal = document.createElement('div');
      modal.id = modalId;
      modal.className = 'modal';
      modal.innerHTML = `
        <div class="modal-overlay"></div>
        <div class="modal-content">
          <div class="modal-header">
            <h2>Delete Promise</h2>
            <button class="modal-close" aria-label="Close">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <p>Are you sure you want to delete this promise? This action cannot be undone.</p>

          <div class="modal-actions" style="display:flex;gap:var(--space-md);justify-content:flex-end;margin-top:var(--space-lg);">
            <button class="btn-danger" id="confirm-delete-btn">Delete</button>
            <button class="btn-secondary" id="cancel-delete-btn">Cancel</button>
          </div>
        </div>
      `;

      document.body.appendChild(modal);

      // Wire up buttons
      modal.querySelector('#confirm-delete-btn')?.addEventListener('click', async () => {
        await this.handleDeletePromise();
        this.closeModals();
      });

      modal.querySelector('#cancel-delete-btn')?.addEventListener('click', () => {
        this.closeModals();
      });

      modal.querySelectorAll('.modal-close, .modal-overlay').forEach(el => {
        el.addEventListener('click', () => this.closeModals());
      });
    } else {
      modal.classList.remove('hidden');
    }
  }

  async handleDeletePromise() {
    if (!this.currentPromiseId) return;

    try {
      await this.promises.deletePromise(this.currentPromiseId);
      this.closeModals();
      await this.loadTimeline();
      this.showTimelineView();
    } catch (error) {
      alert('Failed to delete promise: ' + error.message);
    }
  }

  showEditReminderModal(promise) {
    const modalId = 'edit-reminder-modal';
    let modal = document.getElementById(modalId);

    if (!modal) {
      modal = document.createElement('div');
      modal.id = modalId;
      modal.className = 'modal';
      modal.innerHTML = `
        <div class="modal-overlay"></div>
        <div class="modal-content">
          <div class="modal-header">
            <h2>Edit Reminder Settings</h2>
            <button class="modal-close" aria-label="Close">
              <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <form id="edit-reminder-form">
            <div class="form-group">
              <label for="edit-due-date">Due date (optional)</label>
              <input type="datetime-local" id="edit-due-date">
            </div>
            <div class="form-group custom-select-container">
              <label>Remind me</label>
              <div class="custom-select" id="edit-frequency-select">
                <div class="custom-select-trigger">
                  <span class="custom-select-value">Never</span>
                  <svg xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" d="m19.5 8.25-7.5 7.5-7.5-7.5" />
                  </svg>
                </div>
                <div class="custom-select-options">
                  <div class="custom-option" data-value="">Never</div>
                  <div class="custom-option" data-value="daily">Daily</div>
                  <div class="custom-option" data-value="weekly">Weekly</div>
                  <div class="custom-option" data-value="monthly">Monthly</div>
                </div>
              </div>
              <input type="hidden" id="edit-reminder-frequency" value="">
            </div>
            <div class="form-actions">
              <button type="button" class="btn-secondary" id="cancel-edit-reminder-btn">Cancel</button>
              <button type="submit" class="btn-primary">Save</button>
            </div>
          </form>
        </div>
      `;

      document.body.appendChild(modal);

      // Wire up form submission
      modal.querySelector('#edit-reminder-form')?.addEventListener('submit', async (e) => {
        e.preventDefault();
        await this.handleEditReminder();
      });

      modal.querySelector('#cancel-edit-reminder-btn')?.addEventListener('click', () => {
        this.closeModals();
      });

      modal.querySelectorAll('.modal-close, .modal-overlay').forEach(el => {
        el.addEventListener('click', () => this.closeModals());
      });

      // Initialize custom select for this modal
      this.initCustomSelectForModal('edit-frequency-select', 'edit-reminder-frequency');
    }

    // Populate form with current values
    const dueDateInput = modal.querySelector('#edit-due-date');
    if (promise.due_date && dueDateInput) {
      const dueDate = new Date(promise.due_date);
      // Format date for datetime-local input
      const year = dueDate.getFullYear();
      const month = String(dueDate.getMonth() + 1).padStart(2, '0');
      const day = String(dueDate.getDate()).padStart(2, '0');
      const hours = String(dueDate.getHours()).padStart(2, '0');
      const minutes = String(dueDate.getMinutes()).padStart(2, '0');
      dueDateInput.value = `${year}-${month}-${day}T${hours}:${minutes}`;
    } else if (dueDateInput) {
      dueDateInput.value = '';
    }

    // Set reminder frequency
    const freqInput = modal.querySelector('#edit-reminder-frequency');
    const freqSelect = modal.querySelector('#edit-frequency-select');
    if (freqInput && freqSelect) {
      const frequency = promise.reminder_frequency || '';
      freqInput.value = frequency;
      
      // Update UI
      const option = freqSelect.querySelector(`[data-value="${frequency}"]`);
      if (option) {
        freqSelect.querySelectorAll('.custom-option').forEach(opt => opt.classList.remove('selected'));
        option.classList.add('selected');
        const valueDisplay = freqSelect.querySelector('.custom-select-value');
        if (valueDisplay) valueDisplay.textContent = option.textContent;
      }
    }

    modal.classList.remove('hidden');
  }

  initCustomSelectForModal(selectId, inputId) {
    const select = document.getElementById(selectId);
    const input = document.getElementById(inputId);
    if (!select || !input) return;

    const trigger = select.querySelector('.custom-select-trigger');
    const valueDisplay = select.querySelector('.custom-select-value');
    const options = select.querySelectorAll('.custom-option');

    trigger?.addEventListener('click', (e) => {
      e.stopPropagation();
      // Close others
      document.querySelectorAll('.custom-select.open').forEach(s => {
        if (s !== select) s.classList.remove('open');
      });
      select.classList.toggle('open');
    });

    options.forEach(option => {
      option.addEventListener('click', (e) => {
        e.stopPropagation();
        const value = option.dataset.value;
        input.value = value;
        valueDisplay.textContent = option.textContent;
        
        options.forEach(opt => opt.classList.remove('selected'));
        option.classList.add('selected');
        
        select.classList.remove('open');
      });
    });

    // Close on outside click
    const closeHandler = (e) => {
      if (!select.contains(e.target)) {
        select.classList.remove('open');
      }
    };
    document.addEventListener('click', closeHandler);
  }

  async handleEditReminder() {
    if (!this.currentPromiseId) return;

    const dueDateInput = document.getElementById('edit-due-date').value;
    const reminderFrequency = document.getElementById('edit-reminder-frequency').value;

    const data = {
      reminder_frequency: reminderFrequency || '',
    };

    if (dueDateInput) {
      data.due_date = new Date(dueDateInput).toISOString();
    } else {
      data.due_date = null;
    }

    try {
      await this.promises.updatePromise(this.currentPromiseId, data);
      this.closeModals();
      await this.showPromiseDetail(this.currentPromiseId);
      await this.loadTimeline();
    } catch (error) {
      alert('Failed to update reminder settings: ' + error.message);
    }
  }

  closeModals() {
    document.querySelectorAll('.modal').forEach(modal => {
      modal.classList.add('hidden');
    });
  }

  handleStateSelection(state) {
    this.selectedState = state;
    
    // Update button states
    document.querySelectorAll('.state-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.state === state);
    });

  }

  async handleCreatePromise() {
    const recipient = document.getElementById('promise-recipient').value;
    const description = document.getElementById('promise-description').value;
    const dueDateInput = document.getElementById('promise-due-date').value;
    const reminderFrequency = document.getElementById('promise-reminder-frequency').value;

    const data = {
      recipient,
      description,
      reminder_frequency: reminderFrequency || '',
    };

    if (dueDateInput) {
      data.due_date = new Date(dueDateInput).toISOString();
    }

    try {
      await this.promises.createPromise(data);
      this.closeModals();
      await this.loadTimeline();
    } catch (error) {
      alert('Failed to create promise: ' + error.message);
    }
  }

  async handleResolvePromise() {
    if (!this.selectedState) {
      alert('Please select how the promise went');
      return;
    }

    const reflectionNote = document.getElementById('reflection-note').value;
    const data = {
      state: this.selectedState,
    };

    if (reflectionNote) {
      data.reflection_note = reflectionNote;
    }


    try {
      await this.promises.updatePromiseState(this.currentPromiseId, data);
      this.closeModals();
      await this.showPromiseDetail(this.currentPromiseId);
      await this.loadTimeline();
    } catch (error) {
      alert('Failed to resolve promise: ' + error.message);
    }
  }

  formatDate(date) {
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const thatDay = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    const diffDays = Math.round((thatDay - today) / (1000 * 60 * 60 * 24));

    if (diffDays === 0) return 'today';
    if (diffDays === 1) return 'tomorrow';
    if (diffDays === -1) return 'yesterday';
    if (diffDays > 1 && diffDays < 7) return `in ${diffDays} days`;
    if (diffDays < -1 && diffDays > -7) return `${-diffDays} days ago`;

    return date.toLocaleDateString('en-US', { 
      month: 'short', 
      day: 'numeric',
      year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined
    });
  }

  formatDateTime(date) {
    const now = new Date();
    const diffMs = now - date;
    const diffMins = Math.max(0, Math.floor(diffMs / (1000 * 60)));
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
    
    // For days, use calendar day difference to be consistent with formatDate
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const thatDay = new Date(date.getFullYear(), date.getMonth(), date.getDate());
    const diffDays = Math.round((today - thatDay) / (1000 * 60 * 60 * 24));

    if (diffMins < 1) return 'just now';
    if (diffMins < 60) return `${diffMins} minutes ago`;
    if (diffHours < 24 && diffDays === 0) return `${diffHours} hours ago`;
    if (diffDays === 1) return 'yesterday';
    if (diffDays < 7) return `${diffDays} days ago`;

    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: date.getFullYear() !== now.getFullYear() ? 'numeric' : undefined,
      hour: 'numeric',
      minute: '2-digit'
    });
  }

  escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }
}
