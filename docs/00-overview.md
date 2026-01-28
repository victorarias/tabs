# tabs - Project Overview

**Tablatures for AI Coding Sessions**

---

## What is tabs?

**tabs** is an open-source system for capturing, browsing, and sharing AI coding session transcripts from Claude Code and Cursor IDE. Just as guitar tabs share the technique behind music, tabs shares the prompts and reasoning behind code.

**Core Philosophy:**
- **Prompts matter** - Sometimes the conversation is more valuable than the code
- **Learn by example** - See how others prompt, debug, and architect
- **Local-first** - Capture everything locally, share only what you choose
- **Transparency** - Shared sessions visible to everyone (team knowledge base)

---

## Architecture

```
Local Machine                          Remote Server
┌────────────────────────┐            ┌──────────────────────┐
│                        │            │                      │
│  Claude Code / Cursor  │            │   tabs-server (Go)   │
│         ↓              │            │                      │
│    Hook fires          │            │   - PostgreSQL       │
│         ↓              │   HTTPS    │   - Web UI           │
│     tabs-cli ─────────────────────► │   - Search/Browse    │
│         ↓              │    POST    │   - API keys         │
│    tabs-daemon         │   /api/    │                      │
│         ↓              │  sessions  │   Protected by IAP   │
│  JSONL files in        │            │                      │
│  ~/.tabs/sessions/     │            │                      │
│         ↓              │            │                      │
│  tabs-ui-local         │            │                      │
│  (TanStack Start)      │            │                      │
│                        │            │                      │
└────────────────────────┘            └──────────────────────┘
```

---

## Components

### Local System

1. **tabs-cli** (Go binary)
   - Receives hook events from Claude Code/Cursor
   - Auto-starts daemon if not running
   - Forwards events to daemon via Unix socket
   - User commands: install, push, config

2. **tabs-daemon** (Go binary)
   - PID file concurrency control (one daemon at a time)
   - Unix socket server
   - Claude Code integration (reads JSONL transcripts)
   - Cursor integration (polls SQLite database)
   - Writes events to `~/.tabs/sessions/YYYY-MM-DD/<session-id>-<tool>-<timestamp>.jsonl`

3. **tabs-ui-local** (TanStack Start app)
   - Runs on `http://localhost:3787`
   - Reads JSONL files directly from filesystem
   - Timeline view, search, filters
   - Share workflow (push to remote server)
   - Settings (configure API key, server URL)

### Remote Server

4. **tabs-server** (Go HTTP server)
   - Receives session uploads (API key auth)
   - Stores in PostgreSQL
   - Serves web UI (TanStack Start)
   - Search, browse, tag filtering
   - API key management
   - Protected by IAP (Identity-Aware Proxy)

---

## Data Flow

### Capture (Claude Code)

```
User submits prompt
  ↓
SessionStart hook fires
  ↓
Hook executes: tabs-cli capture-event < hook-payload
  ↓
tabs-cli checks daemon status (PID file)
  ↓
If not running: start daemon
  ↓
tabs-cli connects to Unix socket (~/.tabs/daemon.sock)
  ↓
Daemon receives event
  ↓
Daemon reads transcript from ~/.claude/projects/.../session.jsonl
  ↓
Daemon appends to ~/.tabs/sessions/YYYY-MM-DD/<session-id>-claude-code-<ts>.jsonl
```

### Share

```
User clicks "Share" in local UI
  ↓
UI sends request to local TanStack server
  ↓
Server forwards to daemon via Unix socket
  ↓
Daemon reads session JSONL file
  ↓
Daemon posts to https://tabs.company.com/api/sessions
  Header: Authorization: Bearer <api-key>
  ↓
Remote server validates API key
  ↓
Store session in PostgreSQL (sessions, messages, tools, tags)
  ↓
Return session URL
  ↓
Local UI shows success + URL
```

---

## Storage

### Local

```
~/.tabs/
├── daemon.pid                    # Process ID
├── daemon.sock                   # Unix socket
├── daemon.log                    # Logs
├── config.toml                   # API key, server URL
└── sessions/
    ├── 2026-01-28/
    │   ├── 550e8400-claude-code-1738065600.jsonl
    │   ├── 668320d2-cursor-1738067400.jsonl
    │   └── ...
    ├── 2026-01-29/
    └── ...
```

**JSONL Event Format:**
```json
{"event_type":"session_start","timestamp":"2026-01-28T12:00:00.000Z","tool":"claude-code","session_id":"550e8400-...","data":{...}}
{"event_type":"message","timestamp":"2026-01-28T12:00:05.123Z","tool":"claude-code","session_id":"550e8400-...","data":{...}}
{"event_type":"tool_use","timestamp":"2026-01-28T12:00:15.789Z","tool":"claude-code","session_id":"550e8400-...","data":{...}}
{"event_type":"tool_result","timestamp":"2026-01-28T12:00:16.012Z","tool":"claude-code","session_id":"550e8400-...","data":{...}}
{"event_type":"session_end","timestamp":"2026-01-28T12:05:00.789Z","tool":"claude-code","session_id":"550e8400-...","data":{...}}
```

### Remote (PostgreSQL)

**Tables:**
- `sessions` - Session metadata (session_id, tool, cwd, timestamps, uploaded_by)
- `messages` - User and assistant messages (role, content, model)
- `tools` - Tool invocations and results (tool_name, input, output, is_error)
- `tags` - Session tags (key-value pairs: team:platform, repo:myapp)
- `api_keys` - API keys for authentication (hashed)

---

## Tech Stack

### Backend
- **Language:** Go 1.23+
- **HTTP Server:** net/http (standard library)
- **Database:** PostgreSQL 16
- **Migrations:** golang-migrate or similar
- **Logging:** slog (structured logging)

### Frontend
- **Framework:** TanStack Start (SSR + React)
- **Styling:** Tailwind CSS
- **Components:** Shadcn UI
- **Icons:** Lucide React
- **Animations:** Framer Motion
- **Code Highlighting:** Shiki or Prism

### Infrastructure
- **Deployment:** Docker container
- **IAP:** Cloudflare Access, Google IAP, or Auth0
- **Database:** PostgreSQL (RDS, Cloud SQL, or self-hosted)

---

## Installation & Usage

### Local Setup

```bash
# Download binaries
curl -L https://github.com/yourorg/tabs/releases/latest/download/tabs-cli-$(uname -s)-$(uname -m) -o /usr/local/bin/tabs-cli
chmod +x /usr/local/bin/tabs-cli

curl -L https://github.com/yourorg/tabs/releases/latest/download/tabs-daemon-$(uname -s)-$(uname -m) -o /usr/local/bin/tabs-daemon
chmod +x /usr/local/bin/tabs-daemon

# Install hooks (modifies ~/.claude/config.yaml and ~/.cursor/hooks.json)
tabs-cli install

# Start local UI
tabs-cli ui
# Opens http://localhost:3787
```

### Configuration

```bash
# Set API key (from remote server)
tabs-cli config set api-key tabs_abc123def456...

# Set remote server URL
tabs-cli config set server-url https://tabs.company.com

# Check status
tabs-cli status
```

### Sharing a Session

**Option 1: From local UI**
1. Open http://localhost:3787
2. Click session
3. Click "Share" button
4. Add tags (optional): team:platform, repo:myapp
5. Click "Share →"
6. Copy URL and share with team

**Option 2: From CLI**
```bash
tabs-cli push <session-id> --tags team:platform,repo:myapp
```

### Remote Server (Team Admin)

```bash
# Deploy via Docker
docker-compose up -d

# Environment variables
DATABASE_URL=postgresql://tabs:password@postgres:5432/tabs
PORT=8080
LOG_LEVEL=info
```

**Create API Key:**
1. Visit https://tabs.company.com/keys (IAP protected)
2. Click "Create New Key"
3. Give it a name (e.g., "My Laptop")
4. Copy key (shown once!)
5. Configure locally: `tabs-cli config set api-key <key>`

---

## Document Index

All design documents are in `/docs`:

1. **[01-research-findings.md](./01-research-findings.md)** - Claude Code & Cursor integration research
2. **[02-system-architecture.md](./02-system-architecture.md)** - Component architecture, data flow, deployment
3. **[03-data-format.md](./03-data-format.md)** - JSONL schema, PostgreSQL schema, config format
4. **[04-api-design.md](./04-api-design.md)** - Unix socket protocol, HTTP APIs, error handling
5. **[05-local-ui-flows.md](./05-local-ui-flows.md)** - Local web UI design, components, interactions
6. **[06-remote-server-ux.md](./06-remote-server-ux.md)** - Remote server UI design, search, API keys
7. **[07-daemon-implementation.md](./07-daemon-implementation.md)** - Daemon Go implementation details
8. **[08-cli-implementation.md](./08-cli-implementation.md)** - CLI Go implementation details
9. **[09-server-implementation.md](./09-server-implementation.md)** - Remote server Go implementation details

---

## Implementation Phases

### Phase 1: Claude Code + Local Storage (v0.1)
**Goal:** Capture Claude Code sessions locally

- [ ] Implement tabs-cli (hook handler, daemon starter)
- [ ] Implement tabs-daemon (PID control, Unix socket, JSONL writer)
- [ ] Hook installation (modify ~/.claude/config.yaml)
- [ ] Test end-to-end capture

**Deliverable:** Can capture Claude Code sessions to ~/.tabs/sessions/

---

### Phase 2: Local Web UI (v0.2)
**Goal:** Browse and search local sessions

- [ ] Implement TanStack Start app
- [ ] Timeline view with filters
- [ ] Session detail view
- [ ] Search functionality

**Deliverable:** Can browse local sessions in web UI

---

### Phase 3: Remote Server (v0.3)
**Goal:** Share sessions with team

- [ ] Implement tabs-server (Go HTTP server)
- [ ] PostgreSQL schema + migrations
- [ ] Session upload endpoint
- [ ] API key management
- [ ] Docker containerization
- [ ] Push workflow in local UI

**Deliverable:** Can share sessions to remote server

---

### Phase 4: Cursor Integration (v0.4)
**Goal:** Capture Cursor sessions

- [ ] Cursor hook handler in tabs-cli
- [ ] SQLite poller in tabs-daemon
- [ ] Cursor session → JSONL converter
- [ ] Test end-to-end capture

**Deliverable:** Can capture both Claude Code and Cursor sessions

---

### Phase 5: Polish & Launch (v1.0)
**Goal:** Production-ready

- [ ] Error handling & logging
- [ ] Documentation (README, user guide)
- [ ] CI/CD pipeline
- [ ] Release binaries for Linux/macOS
- [ ] Deployment guide
- [ ] Announcement blog post

**Deliverable:** v1.0 release, ready for teams to adopt

---

## Future Enhancements (v2+)

**Secret Detection:**
- Auto-detect API keys, tokens, passwords before sharing
- Warn user with highlighted matches

**Session Analytics:**
- Most common prompts
- Tool usage statistics
- Team activity dashboard

**Comments & Collaboration:**
- Comment on specific messages in sessions
- "Helpful" reactions (like GitHub reactions)

**Session Forking:**
- Start new session from point in old session
- Like Cursor Enterprise feature

**Export Formats:**
- PDF, HTML, Markdown exports
- Embeddable session widgets for documentation

**Integration:**
- Link sessions to GitHub PRs
- Slack bot for sharing sessions

---

## Security Considerations

### Local
- Unix socket permissions: 0600 (owner only)
- Config file permissions: 0600 (contains API key)
- PID file auto-cleanup on crash

### Remote
- HTTPS only (TLS 1.3+)
- API key hashed with SHA256 (never stored plain text)
- IAP protection on all web UI routes
- Rate limiting (100 req/hr per API key)
- Input validation & parameterized SQL queries

### Privacy
- User reviews sessions before sharing (manual push)
- Future: auto-detect secrets with pattern matching
- Audit log of all uploads (who, when, from which key)

---

## Community & Contribution

**GitHub Repository:** `https://github.com/yourorg/tabs`

**License:** MIT (open source)

**Contributing:**
- Issues: Bug reports, feature requests
- Pull requests: Welcome!
- Discussions: Architecture, design questions

**Support:**
- GitHub Discussions
- Slack community (optional)
- Documentation site

---

## Credits

**Inspired by:**
- [SpecStory](https://specstory.com/) - Original inspiration
- [getspecstory](https://github.com/specstoryai/getspecstory) - Provider pattern architecture
- Guitar tabs - Sharing technique, not just the output

**Built with:**
- Claude Sonnet 4.5 (this very tool!)
- Claude Code CLI
- Love for transparent knowledge sharing

---

## Quick Links

- [Research Findings](./01-research-findings.md) - How Claude Code & Cursor work
- [System Architecture](./02-system-architecture.md) - High-level design
- [Get Started with Implementation](./07-daemon-implementation.md) - Start coding!

---

**Ready to build? Start with the daemon implementation SPEC!**

**Document Status:** Complete
**Last Updated:** 2026-01-28
