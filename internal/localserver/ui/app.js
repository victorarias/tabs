(() => {
  const state = {
    sessions: [],
    query: '',
    tool: '',
    view: 'timeline',
    detail: null,
  };

  const toolIcons = {
    'claude-code': 'Claude',
    cursor: 'Cursor',
    default: 'Tabs',
  };

  const app = document.getElementById('app');
  const searchInput = document.getElementById('search-input');
  const clearBtn = document.getElementById('clear-search');
  const toolFilter = document.getElementById('tool-filter');
  const sessionCount = document.getElementById('session-count');
  const themeToggle = document.getElementById('theme-toggle');
  const toast = document.getElementById('toast');
  let cachedConfig = null;

  const placeholders = [
    'Search sessions, messages, files...',
    'Try: npm install',
    'Try: authentication bug',
    'Try: /home/user/projects/myapp',
  ];
  let placeholderIndex = 0;

  const debounce = (fn, wait) => {
    let timer;
    return (...args) => {
      clearTimeout(timer);
      timer = setTimeout(() => fn(...args), wait);
    };
  };

  const escapeHTML = (value) =>
    String(value)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');

  const formatDuration = (seconds) => {
    if (!seconds || seconds <= 0) return '--';
    const mins = Math.round(seconds / 60);
    if (mins < 60) return `${mins}m`;
    const hours = Math.floor(mins / 60);
    const rem = mins % 60;
    return rem ? `${hours}h ${rem}m` : `${hours}h`;
  };

  const formatTime = (value) => {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
  };

  const formatDate = (value) => {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toLocaleDateString([], { year: 'numeric', month: 'short', day: 'numeric' });
  };

  const dateKey = (value) => {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    return date.toISOString().slice(0, 10);
  };

  const dayLabel = (key) => {
    if (!key) return 'Unknown';
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(today.getDate() - 1);
    const todayKey = today.toISOString().slice(0, 10);
    const yesterdayKey = yesterday.toISOString().slice(0, 10);
    if (key === todayKey) return 'Today';
    if (key === yesterdayKey) return 'Yesterday';
    return formatDate(key);
  };

  const shortenPath = (value) => {
    if (!value) return '';
    const parts = value.split('/').filter(Boolean);
    if (parts.length <= 3) return value;
    return `.../${parts.slice(-3).join('/')}`;
  };

  const extractText = (content) => {
    if (!content) return '';
    if (typeof content === 'string') return content;
    if (Array.isArray(content)) {
      return content
        .map((part) => {
          if (typeof part === 'string') return part;
          if (part && typeof part === 'object') return part.text || '';
          return '';
        })
        .filter(Boolean)
        .join('\n');
    }
    return '';
  };

  const setLoading = (message) => {
    app.innerHTML = `<div class="loading">${escapeHTML(message)}</div>`;
  };

  const showToast = (message) => {
    toast.textContent = message;
    toast.classList.add('visible');
    setTimeout(() => toast.classList.remove('visible'), 2000);
  };

  const updateToolFilter = (sessions) => {
    const tools = new Set(sessions.map((s) => s.tool).filter(Boolean));
    const current = toolFilter.value;
    toolFilter.innerHTML = '<option value="">All</option>';
    Array.from(tools)
      .sort()
      .forEach((tool) => {
        const option = document.createElement('option');
        option.value = tool;
        option.textContent = tool;
        toolFilter.appendChild(option);
      });
    if (current) {
      toolFilter.value = current;
    }
  };

  const fetchSessions = async () => {
    const params = new URLSearchParams();
    if (state.query) params.set('q', state.query);
    if (state.tool) params.set('tool', state.tool);
    const url = `/api/sessions${params.toString() ? `?${params.toString()}` : ''}`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to load sessions');
    const data = await resp.json();
    return data.sessions || [];
  };

  const loadTimeline = async () => {
    state.view = 'timeline';
    setLoading('Loading sessions...');
    try {
      const sessions = await fetchSessions();
      state.sessions = sessions;
      sessionCount.textContent = `${sessions.length} session${sessions.length === 1 ? '' : 's'}`;
      updateToolFilter(sessions);
      renderTimeline(sessions);
    } catch (err) {
      app.innerHTML = `<div class="empty-state"><div class="icon">Alert</div>Failed to load sessions.</div>`;
    }
  };

  const groupSessions = (sessions) => {
    const groups = new Map();
    sessions.forEach((session) => {
      const key = dateKey(session.created_at || session.ended_at);
      if (!groups.has(key)) groups.set(key, []);
      groups.get(key).push(session);
    });
    return Array.from(groups.entries());
  };

  const renderTimeline = (sessions) => {
    if (!sessions.length) {
      app.innerHTML = `
        <div class="empty-state">
          <div class="icon">Tabs</div>
          <h2>No sessions yet</h2>
          <p>Start using Claude Code or Cursor and your sessions will appear here.</p>
        </div>`;
      return;
    }

    const groups = groupSessions(sessions);
    const fragment = document.createDocumentFragment();

    groups.forEach(([key, items]) => {
      const group = document.createElement('section');
      group.className = 'timeline-group';

      const heading = document.createElement('div');
      heading.className = 'timeline-date';
      heading.textContent = dayLabel(key);
      group.appendChild(heading);

      items.forEach((session, index) => {
        const card = document.createElement('article');
        card.className = 'session-card note-in';
        card.style.animationDelay = `${index * 40}ms`;
        card.setAttribute('role', 'button');
        card.tabIndex = 0;
        card.dataset.sessionId = session.session_id;

        const meta = document.createElement('div');
        meta.className = 'session-meta';
        const icon = toolIcons[session.tool] || toolIcons.default;
        const time = formatTime(session.created_at || session.ended_at);
        meta.appendChild(document.createTextNode(`${icon} ${time} `));

        const badge = document.createElement('span');
        badge.className = 'tool-badge';
        badge.textContent = session.tool || 'unknown';
        meta.appendChild(badge);

        const duration = document.createElement('span');
        duration.textContent = formatDuration(session.duration_seconds);
        meta.appendChild(duration);

        const title = document.createElement('div');
        title.className = 'session-title';
        title.textContent = session.summary || `Session ${session.session_id}`;

        const path = document.createElement('div');
        path.className = 'session-path';
        path.title = session.cwd || '';
        path.textContent = shortenPath(session.cwd || '');

        const stats = document.createElement('div');
        stats.className = 'session-stats';
        stats.textContent = `${session.message_count} messages - ${session.tool_use_count} tools`;

        card.appendChild(meta);
        if (session.cwd) card.appendChild(path);
        card.appendChild(title);
        card.appendChild(stats);

        card.addEventListener('click', () => navigate(`/sessions/${session.session_id}`));
        card.addEventListener('keydown', (event) => {
          if (event.key === 'Enter') navigate(`/sessions/${session.session_id}`);
        });

        group.appendChild(card);
      });

      fragment.appendChild(group);
    });

    app.innerHTML = '';
    app.appendChild(fragment);
  };

  const buildDetailItems = (events) => {
    const items = [];
    const toolMap = new Map();

    events.forEach((event) => {
      const type = event.event_type;
      const data = event.data || {};
      if (type === 'message') {
        items.push({ type: 'message', event });
      } else if (type === 'tool_use') {
        const item = {
          type: 'tool',
          tool_use_id: data.tool_use_id,
          tool_name: data.tool_name || 'tool',
          input: data.input,
          output: null,
          is_error: false,
          timestamp: event.timestamp,
        };
        items.push(item);
        if (data.tool_use_id) toolMap.set(data.tool_use_id, item);
      } else if (type === 'tool_result') {
        const existing = toolMap.get(data.tool_use_id);
        if (existing) {
          existing.output = data.content;
          existing.is_error = Boolean(data.is_error);
        } else {
          items.push({
            type: 'tool',
            tool_use_id: data.tool_use_id,
            tool_name: 'tool',
            input: null,
            output: data.content,
            is_error: Boolean(data.is_error),
            timestamp: event.timestamp,
          });
        }
      }
    });

    return items;
  };

  const renderSessionDetail = (detail) => {
    if (!detail) return;
    const events = detail.events || [];
    const messageCount = events.filter((e) => e.event_type === 'message').length;
    const toolCount = events.filter((e) => e.event_type === 'tool_use').length;

    const header = document.createElement('section');
    header.className = 'session-header';
    header.innerHTML = `
      <a class="back-link" href="/" data-nav>&larr; Back</a>
      <div class="session-id">Session: ${escapeHTML(detail.session_id)}</div>
      <div class="session-actions">
        <button class="primary-btn" id="share-session" type="button">Share</button>
      </div>
      <div class="session-meta-line">${escapeHTML(detail.tool || 'unknown')} - ${escapeHTML(formatDate(detail.created_at))} ${escapeHTML(formatTime(detail.created_at))} - ${escapeHTML(formatDuration(detail.duration_seconds))}</div>
      <div class="session-meta-line">${escapeHTML(detail.cwd || '')}</div>
      <div class="session-meta-line">${messageCount} messages - ${toolCount} tools</div>
    `;

    const list = document.createElement('div');
    list.className = 'detail-list';

    const items = buildDetailItems(events);
    items.forEach((item) => {
      if (item.type === 'message') {
        const data = item.event.data || {};
        const role = data.role || 'unknown';
        const content = extractText(data.content || data.text || '');
        const card = document.createElement('div');
        card.className = 'message-card';
        card.innerHTML = `
          <div class="role">${escapeHTML(role === 'user' ? 'User' : 'Assistant')} - ${escapeHTML(formatTime(item.event.timestamp))}</div>
          <div class="message-content">${escapeHTML(content)}</div>
        `;
        list.appendChild(card);
        return;
      }

      if (item.type === 'tool') {
        const card = document.createElement('div');
        card.className = `tool-card${item.is_error ? ' error' : ''}`;
        const inputText = item.input ? JSON.stringify(item.input, null, 2) : '';
        const outputText = item.output ? JSON.stringify(item.output, null, 2) : '';
        const inputOpen = inputText.length <= 500 ? 'open' : '';
        const outputOpen = outputText.length <= 500 ? 'open' : '';
        card.innerHTML = `
          <div class="tool-title">
            <span>Tool ${escapeHTML(item.tool_name)}</span>
            <span>${escapeHTML(formatTime(item.timestamp))}</span>
          </div>
          ${inputText ? `<details ${inputOpen}><summary>Input</summary><pre>${escapeHTML(inputText)}</pre></details>` : ''}
          ${outputText ? `<details ${outputOpen}><summary>Output</summary><pre>${escapeHTML(outputText)}</pre></details>` : ''}
        `;
        list.appendChild(card);
      }
    });

    app.innerHTML = '';
    app.appendChild(header);
    app.appendChild(list);

    const shareBtn = document.getElementById('share-session');
    if (shareBtn) {
      shareBtn.addEventListener('click', () => openShareModal(detail));
    }
  };

  const parseTagInput = (value) => {
    const tags = [];
    const tokens = value
      .split(/[,\\n]/)
      .map((part) => part.trim())
      .filter(Boolean);
    tokens.forEach((token) => {
      const split = token.includes(':') ? token.split(':') : token.split('=');
      if (split.length < 2) return;
      const key = split[0].trim();
      const rest = split.slice(1).join(':').trim();
      if (!key || !rest) return;
      tags.push({ key, value: rest });
    });
    return tags;
  };

  const openShareModal = (detail) => {
    const overlay = document.createElement('div');
    overlay.className = 'modal-overlay';
    overlay.innerHTML = `
      <div class="modal">
        <div class="modal-header">
          <h3>Share Session</h3>
          <button class="ghost-btn" type="button" id="share-close">Close</button>
        </div>
        <p class="modal-subtext">Add tags like <strong>team:platform</strong> or <strong>repo:myapp</strong>.</p>
        <label class="filter-label">Tags (optional)
          <input id="share-tags" class="search-input" type="text" placeholder="team:platform, repo:myapp" />
        </label>
        <div class="session-stats" id="share-status"></div>
        <div class="modal-actions">
          <button class="ghost-btn" type="button" id="share-cancel">Cancel</button>
          <button class="primary-btn" type="button" id="share-submit">Share →</button>
        </div>
      </div>
    `;

    const close = () => {
      overlay.remove();
      document.removeEventListener('keydown', onKeydown);
    };
    const onKeydown = (event) => {
      if (event.key === 'Escape') close();
    };
    document.addEventListener('keydown', onKeydown);

    overlay.addEventListener('click', (event) => {
      if (event.target === overlay) close();
    });

    const closeBtn = overlay.querySelector('#share-close');
    const cancelBtn = overlay.querySelector('#share-cancel');
    const submitBtn = overlay.querySelector('#share-submit');
    const tagsInput = overlay.querySelector('#share-tags');
    const statusEl = overlay.querySelector('#share-status');

    const ensureDefaults = async () => {
      if (!cachedConfig) {
        const resp = await fetch('/api/config');
        if (resp.ok) {
          cachedConfig = await resp.json();
        }
      }
      const defaults =
        cachedConfig &&
        cachedConfig.remote &&
        Array.isArray(cachedConfig.remote.default_tags)
          ? cachedConfig.remote.default_tags
          : [];
      if (tagsInput && defaults.length && !tagsInput.value.trim()) {
        tagsInput.value = defaults.join(', ');
      }
    };

    const share = async () => {
      statusEl.textContent = 'Uploading...';
      const tags = parseTagInput(tagsInput.value || '');
      const payload = {
        session_id: detail.session_id,
        tool: detail.tool,
        tags,
      };
      const resp = await fetch('/api/sessions/push', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (resp.ok) {
        const data = await resp.json().catch(() => null);
        statusEl.textContent = data && data.url ? `Shared: ${data.url}` : 'Shared.';
        showToast('Session shared');
        setTimeout(close, 1200);
      } else {
        const data = await resp.json().catch(() => null);
        statusEl.textContent = data && data.error ? data.error.message : 'Failed to share.';
      }
    };

    if (closeBtn) closeBtn.addEventListener('click', close);
    if (cancelBtn) cancelBtn.addEventListener('click', close);
    if (submitBtn) submitBtn.addEventListener('click', share);

    document.body.appendChild(overlay);
    ensureDefaults().catch(() => {});
    if (tagsInput) tagsInput.focus();
  };

  const loadSessionDetail = async (id) => {
    state.view = 'detail';
    setLoading('Loading session...');
    try {
      const resp = await fetch(`/api/sessions/${id}`);
      if (!resp.ok) throw new Error('Failed to load session');
      const data = await resp.json();
      state.detail = data.session;
      renderSessionDetail(state.detail);
    } catch (err) {
      app.innerHTML = `<div class="empty-state"><div class="icon">Alert</div>Session not found.</div>`;
    }
  };

  const loadSettings = async () => {
    state.view = 'settings';
    setLoading('Loading settings...');
    try {
      const [configResp, daemonResp] = await Promise.all([
        fetch('/api/config'),
        fetch('/api/daemon/status'),
      ]);
      const config = configResp.ok ? await configResp.json() : null;
      const daemon = daemonResp.ok ? await daemonResp.json() : null;
      renderSettings(config, daemon);
    } catch (err) {
      app.innerHTML = `<div class="empty-state"><div class="icon">Alert</div>Settings unavailable.</div>`;
    }
  };

  const renderSettings = (config, daemon) => {
    const remote = (config && config.remote) || {};
    const daemonStatus = (daemon && daemon.status) || {};
    app.innerHTML = `
      <section class="session-header">
        <h2>Settings</h2>
        <p>Manage your local tabs configuration.</p>
      </section>
      <section class="message-card">
        <div class="role">Remote Server</div>
        <label class="filter-label">Server URL
          <input id="server-url" class="search-input" type="text" value="${escapeHTML(remote.server_url || '')}" />
        </label>
        <label class="filter-label">API Key
          <input id="api-key" class="search-input" type="password" value="" placeholder="tabs_..." />
        </label>
        <button class="ghost-btn" id="save-settings" type="button">Save Changes</button>
        <div class="session-stats" id="settings-status"></div>
      </section>
      <section class="message-card">
        <div class="role">Daemon Status</div>
        <div class="session-stats">Status: ${daemonStatus.running ? 'Running' : 'Stopped'}</div>
        <div class="session-stats">PID: ${daemonStatus.pid || '--'}</div>
        <div class="session-stats">Uptime: ${daemonStatus.uptime_seconds || 0}s</div>
      </section>
    `;

    const saveBtn = document.getElementById('save-settings');
    const serverInput = document.getElementById('server-url');
    const apiInput = document.getElementById('api-key');
    const statusEl = document.getElementById('settings-status');

    saveBtn.addEventListener('click', async () => {
      statusEl.textContent = 'Saving...';
      const payload = {
        remote: {
          server_url: serverInput.value.trim(),
          api_key: apiInput.value.trim(),
        },
      };
      const resp = await fetch('/api/config', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (resp.ok) {
        statusEl.textContent = 'Saved.';
        apiInput.value = '';
        showToast('Settings updated');
      } else {
        const data = await resp.json().catch(() => null);
        statusEl.textContent = data && data.error ? data.error.message : 'Failed to save.';
      }
    });
  };

  const route = () => {
    const path = window.location.pathname;
    if (path.startsWith('/sessions/')) {
      const id = path.split('/').pop();
      if (id) loadSessionDetail(id);
      return;
    }
    if (path === '/settings') {
      loadSettings();
      return;
    }
    loadTimeline();
  };

  const navigate = (path) => {
    history.pushState({}, '', path);
    route();
  };

  document.addEventListener('click', (event) => {
    const target = event.target;
    if (!(target instanceof HTMLElement)) return;
    const nav = target.closest('[data-nav]');
    if (nav) {
      event.preventDefault();
      const href = nav.getAttribute('href') || nav.dataset.nav;
      if (href) navigate(href);
    }
  });

  window.addEventListener('popstate', route);

  const applyTheme = (theme) => {
    const isDark =
      theme === 'dark' ||
      (theme === 'system' && window.matchMedia('(prefers-color-scheme: dark)').matches);
    document.body.classList.toggle('dark', isDark);
  };

  const initTheme = () => {
    const stored = localStorage.getItem('tabs-theme') || 'system';
    applyTheme(stored);
  };

  themeToggle.addEventListener('click', () => {
    const current = localStorage.getItem('tabs-theme') || 'system';
    const next = current === 'light' ? 'dark' : current === 'dark' ? 'system' : 'light';
    localStorage.setItem('tabs-theme', next);
    applyTheme(next);
    showToast(`Theme: ${next}`);
  });

  searchInput.addEventListener(
    'input',
    debounce(() => {
      state.query = searchInput.value.trim();
      clearBtn.classList.toggle('visible', Boolean(state.query));
      if (state.view === 'timeline') loadTimeline();
    }, 300),
  );

  clearBtn.addEventListener('click', () => {
    searchInput.value = '';
    state.query = '';
    clearBtn.classList.remove('visible');
    if (state.view === 'timeline') loadTimeline();
  });

  toolFilter.addEventListener('change', () => {
    state.tool = toolFilter.value;
    if (state.view === 'timeline') loadTimeline();
  });

  document.addEventListener('keydown', (event) => {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      searchInput.focus();
    }
  });

  setInterval(() => {
    if (searchInput.value) return;
    placeholderIndex = (placeholderIndex + 1) % placeholders.length;
    searchInput.placeholder = placeholders[placeholderIndex];
  }, 4000);

  initTheme();
  route();
})();
