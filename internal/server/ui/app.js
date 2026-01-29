(() => {
  const state = {
    sessions: [],
    tags: [],
    activeTags: [],
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
  const tagCloud = document.getElementById('tag-cloud');
  const activeTags = document.getElementById('active-tags');
  const toast = document.getElementById('toast');

  const placeholders = [
    'Search across all shared sessions...',
    'Try: npm install',
    'Try: authentication bug',
    'Try: team:platform',
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

  const formatRelativeTime = (value) => {
    if (!value) return '';
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return '';
    const now = new Date();
    const diff = now - date;
    const mins = Math.floor(diff / 60000);
    if (mins < 1) return 'just now';
    if (mins < 60) return `${mins}m ago`;
    const hours = Math.floor(mins / 60);
    if (hours < 24) return `${hours}h ago`;
    const days = Math.floor(hours / 24);
    if (days < 7) return `${days}d ago`;
    return formatDate(value);
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
    state.activeTags.forEach((tag) => {
      params.append('tag', `${tag.key}:${tag.value}`);
    });
    const url = `/api/sessions${params.toString() ? `?${params.toString()}` : ''}`;
    const resp = await fetch(url);
    if (!resp.ok) throw new Error('Failed to load sessions');
    const data = await resp.json();
    return data.sessions || [];
  };

  const fetchTags = async () => {
    const resp = await fetch('/api/tags?limit=20');
    if (!resp.ok) return [];
    const data = await resp.json();
    return data.tags || [];
  };

  const loadTimeline = async () => {
    state.view = 'timeline';
    setLoading('Loading sessions...');
    try {
      const [sessions, tags] = await Promise.all([fetchSessions(), fetchTags()]);
      state.sessions = sessions;
      state.tags = tags;
      sessionCount.textContent = `${sessions.length} session${sessions.length === 1 ? '' : 's'}`;
      updateToolFilter(sessions);
      renderTagCloud(tags);
      renderTimeline(sessions);
    } catch (err) {
      app.innerHTML = `<div class="empty-state"><div class="icon">Alert</div>Failed to load sessions.</div>`;
    }
  };

  const renderTagCloud = (tags) => {
    tagCloud.innerHTML = '';
    tags.slice(0, 10).forEach((tag) => {
      const pill = document.createElement('button');
      pill.className = 'tag-pill';
      const isActive = state.activeTags.some(
        (t) => t.key === tag.key && t.value === tag.value
      );
      if (isActive) pill.classList.add('active');
      pill.innerHTML = `${escapeHTML(tag.key)}:${escapeHTML(tag.value)}<span class="count">(${tag.count})</span>`;
      pill.addEventListener('click', () => toggleTag(tag));
      tagCloud.appendChild(pill);
    });
  };

  const renderActiveTags = () => {
    activeTags.innerHTML = '';
    state.activeTags.forEach((tag) => {
      const el = document.createElement('span');
      el.className = 'active-tag';
      el.innerHTML = `${escapeHTML(tag.key)}:${escapeHTML(tag.value)}<span class="remove">&times;</span>`;
      el.querySelector('.remove').addEventListener('click', () => {
        state.activeTags = state.activeTags.filter(
          (t) => !(t.key === tag.key && t.value === tag.value)
        );
        renderActiveTags();
        loadTimeline();
      });
      activeTags.appendChild(el);
    });
  };

  const toggleTag = (tag) => {
    const idx = state.activeTags.findIndex(
      (t) => t.key === tag.key && t.value === tag.value
    );
    if (idx >= 0) {
      state.activeTags.splice(idx, 1);
    } else {
      state.activeTags.push({ key: tag.key, value: tag.value });
    }
    renderActiveTags();
    loadTimeline();
  };

  const groupSessions = (sessions) => {
    const groups = new Map();
    sessions.forEach((session) => {
      const key = dateKey(session.created_at || session.uploaded_at);
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
          <h2>No sessions shared yet</h2>
          <p>Be the first to share a session! Create an API key and upload from your local tabs.</p>
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
        card.dataset.sessionId = session.id;

        const meta = document.createElement('div');
        meta.className = 'session-meta';
        const icon = toolIcons[session.tool] || toolIcons.default;
        const time = formatTime(session.created_at);
        meta.appendChild(document.createTextNode(`${icon} ${time} `));

        const badge = document.createElement('span');
        badge.className = 'tool-badge';
        badge.textContent = session.tool || 'unknown';
        meta.appendChild(badge);

        const duration = document.createElement('span');
        duration.textContent = formatDuration(session.duration_seconds);
        meta.appendChild(duration);

        card.appendChild(meta);

        if (session.cwd) {
          const path = document.createElement('div');
          path.className = 'session-path';
          path.title = session.cwd;
          path.textContent = shortenPath(session.cwd);
          card.appendChild(path);
        }

        const title = document.createElement('div');
        title.className = 'session-title';
        title.textContent = session.summary || `Session ${session.session_id}`;
        card.appendChild(title);

        if (session.uploaded_by) {
          const uploader = document.createElement('div');
          uploader.className = 'session-uploader';
          uploader.innerHTML = `<span class="icon">&#9650;</span> ${escapeHTML(session.uploaded_by)} - ${escapeHTML(formatRelativeTime(session.uploaded_at))}`;
          card.appendChild(uploader);
        }

        if (session.tags && session.tags.length > 0) {
          const tagsEl = document.createElement('div');
          tagsEl.className = 'session-tags';
          session.tags.slice(0, 5).forEach((tag) => {
            const tagEl = document.createElement('span');
            tagEl.className = 'session-tag';
            tagEl.textContent = `${tag.key}:${tag.value}`;
            tagsEl.appendChild(tagEl);
          });
          card.appendChild(tagsEl);
        }

        const stats = document.createElement('div');
        stats.className = 'session-stats';
        stats.textContent = `${session.message_count} messages - ${session.tool_use_count} tools`;
        card.appendChild(stats);

        card.addEventListener('click', () => navigate(`/sessions/${session.id}`));
        card.addEventListener('keydown', (event) => {
          if (event.key === 'Enter') navigate(`/sessions/${session.id}`);
        });

        group.appendChild(card);
      });

      fragment.appendChild(group);
    });

    app.innerHTML = '';
    app.appendChild(fragment);
  };

  const renderSessionDetail = (detail) => {
    if (!detail) return;
    const messages = detail.messages || [];
    const tools = detail.tools || [];

    const header = document.createElement('section');
    header.className = 'session-header';

    let tagsHtml = '';
    if (detail.tags && detail.tags.length > 0) {
      tagsHtml = `<div class="session-tags">${detail.tags.map(t => `<span class="session-tag">${escapeHTML(t.key)}:${escapeHTML(t.value)}</span>`).join('')}</div>`;
    }

    header.innerHTML = `
      <a class="back-link" href="/" data-nav>&larr; Back</a>
      <div class="session-id">Session: ${escapeHTML(detail.session_id)}</div>
      <div class="session-meta-line">${escapeHTML(detail.tool || 'unknown')} - ${escapeHTML(formatDate(detail.created_at))} ${escapeHTML(formatTime(detail.created_at))} - ${escapeHTML(formatDuration(detail.duration_seconds))}</div>
      <div class="session-meta-line">${escapeHTML(detail.cwd || '')}</div>
      <div class="session-uploader"><span class="icon">&#9650;</span> Shared by ${escapeHTML(detail.uploaded_by || 'unknown')} on ${escapeHTML(formatDate(detail.uploaded_at))}</div>
      ${tagsHtml}
      <div class="session-meta-line">${detail.message_count} messages - ${detail.tool_use_count} tools</div>
    `;

    const list = document.createElement('div');
    list.className = 'detail-list';

    const items = mergeMessagesAndTools(messages, tools);
    items.forEach((item) => {
      if (item.type === 'message') {
        const content = extractText(item.content);
        const card = document.createElement('div');
        card.className = 'message-card';
        card.innerHTML = `
          <div class="role">${escapeHTML(item.role === 'user' ? 'User' : 'Assistant')} - ${escapeHTML(formatTime(item.timestamp))}</div>
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
  };

  const mergeMessagesAndTools = (messages, tools) => {
    const items = [];
    messages.forEach((msg) => {
      items.push({
        type: 'message',
        timestamp: msg.timestamp,
        role: msg.role,
        content: msg.content,
      });
    });
    tools.forEach((tool) => {
      items.push({
        type: 'tool',
        timestamp: tool.timestamp,
        tool_name: tool.tool_name,
        input: tool.input,
        output: tool.output,
        is_error: tool.is_error,
      });
    });
    items.sort((a, b) => new Date(a.timestamp) - new Date(b.timestamp));
    return items;
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

  const loadKeys = async () => {
    state.view = 'keys';
    app.innerHTML = `
      <section class="session-header">
        <h2>API Keys</h2>
        <p>Create keys to upload sessions from your local machine.</p>
        <p class="session-meta-line">Note: API key management requires authentication via IAP.</p>
      </section>
      <div class="empty-state">
        <div class="icon">Key</div>
        <p>API key management coming soon.</p>
        <p>For now, contact your admin to create an API key.</p>
      </div>
    `;
  };

  const route = () => {
    const path = window.location.pathname;
    if (path.startsWith('/sessions/')) {
      const id = path.split('/').pop();
      if (id) loadSessionDetail(id);
      return;
    }
    if (path === '/keys') {
      loadKeys();
      return;
    }
    if (path === '/search') {
      loadTimeline();
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
