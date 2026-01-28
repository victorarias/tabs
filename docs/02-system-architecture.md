# System Architecture: tabs

**Version:** 1.0
**Date:** 2026-01-28
**Status:** SPEC

---

## Overview

**tabs** (tablatures for AI coding sessions) is a distributed system for capturing, browsing, and sharing AI coding session transcripts from Claude Code and Cursor IDE.

**Design Principles:**
1. **Local-first** - All sessions captured locally in plain JSONL files
2. **Transparent by default** - Shared sessions visible to everyone with access
3. **Hook-driven** - Real-time capture via IDE hooks (not file watching)
4. **Privacy-conscious** - Share only what you explicitly push
5. **Simple storage** - JSONL locally, PostgreSQL remotely
6. **Auto-starting** - Daemon starts on-demand, no manual management

---

## System Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        LOCAL MACHINE                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────┐              ┌──────────────┐                │
│  │ Claude Code  │              │   Cursor     │                │
│  │              │              │              │                │
│  │ ~/.claude/   │              │ .cursor/     │                │
│  │ config.yaml  │              │ hooks.json   │                │
│  └──────┬───────┘              └──────┬───────┘                │
│         │                             │                         │
│         │ Hook invokes                │ Hook invokes           │
│         │ tabs-cli                    │ tabs-cli               │
│         ▼                             ▼                         │
│  ┌────────────────────────────────────────────────────┐        │
│  │              tabs-cli (Go Binary)                  │        │
│  │                                                     │        │
│  │  - Receives hook event via stdin (JSON)            │        │
│  │  - Checks daemon status (PID file + socket)        │        │
│  │  - Auto-starts daemon if not running               │        │
│  │  - Forwards event to daemon via Unix socket        │        │
│  │  - Returns immediately (non-blocking)              │        │
│  │                                                     │        │
│  │  Commands:                                          │        │
│  │    tabs-cli install      # Install hooks           │        │
│  │    tabs-cli capture-event # Hook handler (internal)│        │
│  │    tabs-cli push <id>    # Manually push session   │        │
│  │    tabs-cli config set   # Configure API key/URL   │        │
│  └──────────────────┬─────────────────────────────────┘        │
│                     │                                            │
│                     │ Unix Socket                                │
│                     │ ~/.tabs/daemon.sock                        │
│                     ▼                                            │
│  ┌────────────────────────────────────────────────────┐        │
│  │            tabs-daemon (Go Binary)                 │        │
│  │                                                     │        │
│  │  - PID file concurrency control                    │        │
│  │  - Unix socket server (receives events from CLI)   │        │
│  │  - JSONL writer (atomic appends)                   │        │
│  │  - Claude Code handler (reads transcript JSONL)    │        │
│  │  - Cursor handler (polls SQLite DB)                │        │
│  │  - Graceful shutdown on SIGTERM/SIGINT             │        │
│  │                                                     │        │
│  │  Storage: ~/.tabs/                                 │        │
│  │    ├─ daemon.pid      # Process ID                 │        │
│  │    ├─ daemon.lock     # Single-instance lock file  │        │
│  │    ├─ daemon.sock     # Unix domain socket         │        │
│  │    ├─ daemon.log      # Daemon logs                │        │
│  │    ├─ config.toml     # User config (API key, URL) │        │
│  │    ├─ state/          # Per-session cursor state   │        │
│  │    └─ sessions/       # Captured sessions          │        │
│  │        └─ YYYY-MM-DD/ # Organized by date          │        │
│  │           └─ <session-id>-<tool>-<ts>.jsonl        │        │
│  └────────────────────┬───────────────────────────────┘        │
│                       │                                          │
│                       │ Reads JSONL directly                     │
│                       ▼                                          │
│  ┌────────────────────────────────────────────────────┐        │
│  │         tabs-ui-local (TanStack Start)             │        │
│  │                                                     │        │
│  │  - Runs on http://localhost:3787                   │        │
│  │  - Reads JSONL files directly from ~/.tabs/        │        │
│  │  - Timeline view (sessions by date, newest first)  │        │
│  │  - Search/filter (date, cwd, tool)                 │        │
│  │  - Session detail view (messages, tools, thinking) │        │
│  │  - Share button → push to remote server            │        │
│  │  - Settings page (configure API key, server URL)   │        │
│  │                                                     │        │
│  │  Communication:                                     │        │
│  │    - Reads: Direct filesystem access to JSONL      │        │
│  │    - Push: Sends to daemon via Unix socket         │        │
│  └────────────────────┬───────────────────────────────┘        │
│                       │                                          │
└───────────────────────┼──────────────────────────────────────────┘
                        │
                        │ HTTPS POST /api/sessions
                        │ Authorization: Bearer <api-key>
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                      REMOTE SERVER                               │
│             (Deployed in corporate environment)                  │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────────────────────────────────────────────┐          │
│  │           IAP / Access Control Layer              │          │
│  │  (Cloudflare Access, Google IAP, Auth0, etc.)    │          │
│  │                                                   │          │
│  │  Protects:                                        │          │
│  │    - GET  /                   # Web UI           │          │
│  │    - GET  /api/keys           # API key creation │          │
│  │    - All other UI routes                         │          │
│  │                                                   │          │
│  │  Public (API key auth only):                     │          │
│  │    - POST /api/sessions       # Upload session   │          │
│  └──────────────────┬───────────────────────────────┘          │
│                     ▼                                            │
│  ┌────────────────────────────────────────────────────┐        │
│  │          tabs-server (Go HTTP Server)              │        │
│  │                                                     │        │
│  │  API Endpoints:                                    │        │
│  │    POST /api/sessions       # Upload session      │        │
│  │    GET  /api/sessions       # List/search         │        │
│  │    GET  /api/sessions/:id   # Get session detail  │        │
│  │    POST /api/keys           # Create API key      │        │
│  │    GET  /api/tags           # List all tags       │        │
│  │                                                     │        │
│  │  Web UI Serving:                                   │        │
│  │    GET  /                    # TanStack Start app  │        │
│  │    GET  /sessions/:id        # Session view        │        │
│  │    GET  /search              # Search page         │        │
│  │    GET  /keys                # API key management  │        │
│  │                                                     │        │
│  │  Middleware:                                        │        │
│  │    - API key validation (for /api/sessions)        │        │
│  │    - CORS handling                                  │        │
│  │    - Request logging                                │        │
│  │    - Rate limiting                                  │        │
│  └────────────────────┬───────────────────────────────┘        │
│                       │                                          │
│                       │ SQL queries                              │
│                       ▼                                          │
│  ┌────────────────────────────────────────────────────┐        │
│  │              PostgreSQL Database                   │        │
│  │                                                     │        │
│  │  Tables:                                            │        │
│  │    - sessions      # Session metadata              │        │
│  │    - messages      # Individual messages           │        │
│  │    - tools         # Tool uses and results         │        │
│  │    - tags          # Session tags (team, repo)     │        │
│  │    - api_keys      # User API keys                 │        │
│  │                                                     │        │
│  │  Indexes:                                           │        │
│  │    - sessions(created_at DESC)                     │        │
│  │    - tags(tag_key, tag_value)                      │        │
│  │    - messages(session_id, seq)                     │        │
│  └────────────────────────────────────────────────────┘        │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘

    ┌────────────────────────────────────────────┐
    │         Deployment: Docker Container        │
    │                                             │
    │  FROM golang:1.23-alpine                    │
    │  COPY tabs-server /usr/local/bin/          │
    │  EXPOSE 8080                                │
    │  CMD ["tabs-server"]                        │
    │                                             │
    │  Environment Variables:                     │
    │    DATABASE_URL=postgresql://...            │
    │    PORT=8080                                │
    │    LOG_LEVEL=info                           │
    └────────────────────────────────────────────┘
```

---

## Component Details

### 1. tabs-cli (Go Binary)

**Responsibilities:**
- Receive hook events from Claude Code / Cursor (via stdin)
- Check if daemon is running
- Auto-start daemon if needed
- Forward events to daemon via Unix socket
- Provide user-facing commands (install, push, config)

**Key Features:**

#### Hook Handler (Invisible to User)
```bash
# Claude Code hook calls:
tabs-cli capture-event --tool=claude-code < hook-payload.json

# Cursor hook calls:
tabs-cli capture-event --tool=cursor < hook-payload.json
```

**Process:**
1. Read JSON from stdin
2. Check if daemon is running:
   - Read PID from `~/.tabs/daemon.pid`
   - Check if process exists: `kill -0 <pid>`
   - Check if socket exists: `~/.tabs/daemon.sock`
3. If daemon not running:
   - Start daemon: `tabs-daemon &`
   - Wait for socket to appear (max 2s)
4. Connect to Unix socket
5. Send event JSON
6. Close connection
7. Exit immediately (fast, non-blocking)

#### User Commands
```bash
# Install hooks (modifies ~/.claude/config.yaml and ~/.cursor/hooks.json)
tabs-cli install

# Uninstall hooks
tabs-cli uninstall

# Check installation status
tabs-cli status

# Configure API key and remote server
tabs-cli config set api-key <key>
tabs-cli config set server-url https://tabs.company.com

# Manually push a session
tabs-cli push <session-id> --tags team:platform,repo:myapp

# List captured sessions
tabs-cli list

# Show daemon status
tabs-cli daemon status
tabs-cli daemon stop
```

**Binary Location:** `/usr/local/bin/tabs-cli` (or `~/bin/tabs-cli`)

---

### 2. tabs-daemon (Go Binary)

**Responsibilities:**
- PID file concurrency control (only one daemon per user)
- Unix socket server (receives events from CLI)
- Event processing (parse, validate, enrich)
- JSONL writer (atomic appends to session files)
- Claude Code integration (read transcript from JSONL)
- Cursor integration (poll SQLite database)
- Graceful shutdown

**Key Features:**

#### PID File Concurrency Control

**Startup Sequence:**
1. Check if `~/.tabs/daemon.pid` exists
2. If exists:
   - Read PID from file
   - Check if process is alive: `kill -0 <pid>`
   - If alive: exit with error "Daemon already running"
   - If dead (stale PID file): remove PID file and continue
3. Acquire single-instance lock: create `~/.tabs/daemon.lock` with `O_EXCL`
   - If lock exists: exit (another daemon is running or starting)
4. Write own PID to `~/.tabs/daemon.pid`
5. Start Unix socket server
6. Register signal handlers (SIGTERM, SIGINT)

**Shutdown Sequence:**
1. Receive SIGTERM or SIGINT
2. Close Unix socket
3. Wait for in-flight events to complete (max 5s)
4. Flush any buffered JSONL writes
5. Remove PID file
6. Remove lock file
7. Remove socket file
8. Exit gracefully

**Auto-Recovery:**
If daemon crashes:
- PID file remains but process is dead
- Next CLI invocation detects stale PID (kill -0 fails)
- Removes stale PID file
- Starts new daemon

#### Unix Socket Server

**Socket Location:** `~/.tabs/daemon.sock`

**Protocol:** JSON messages over Unix domain socket

**Message Format:**
```json
{
  "type": "capture_event",
  "tool": "claude-code" | "cursor",
  "timestamp": "2026-01-28T12:00:00Z",
  "payload": {
    // Tool-specific hook payload
  }
}
```

**Flow:**
1. Listen on Unix socket
2. Accept connection from CLI
3. Read JSON message
4. Process event (see Event Processing below)
5. Send response: `{"status": "ok"}` or `{"status": "error", "message": "..."}`
6. Close connection

#### Event Processing

**Claude Code Events:**
1. Receive hook payload with `session_id` and `transcript_path`
2. Read JSONL from `transcript_path`
3. Parse all records (one JSON per line)
4. Filter out warmup messages (isSidechain)
5. Deduplicate using per-session cursor state:
   - Read `~/.tabs/state/<session-id>.json` (last byte offset + last line hash)
   - Process only new lines since offset
   - Update cursor after successful append
6. Determine event type: session_start, message, tool_use, tool_result, session_end
7. Generate filename: `<session-id>-claude-code-<timestamp>.jsonl`
8. Append event to `~/.tabs/sessions/<date>/<filename>`

**Cursor Events:**
1. Receive hook payload with `conversation_id`, `generation_id`, `prompt`
2. Store in memory: `{conversation_id: {...}}`
3. Background goroutine polls SQLite DB every 2 seconds
4. Query for AI responses matching `conversation_id` + `generation_id`
5. When found, append to session JSONL
6. On `stop` hook, mark conversation complete

#### JSONL Writer

**Atomic Append Strategy:**
1. Open file in append mode with `O_APPEND | O_CREATE`
2. Acquire file lock (flock) for write
3. Serialize event to JSON
4. Append newline-terminated JSON
5. Flush to disk (fsync)
6. Release file lock
7. Close file

**Concurrency:** Uses file locking to prevent corruption if multiple processes write (shouldn't happen, but defensive)

#### Directory Structure Management

**Initialization:**
1. Create `~/.tabs/` if not exists
2. Create `~/.tabs/sessions/` if not exists
3. Create `~/.tabs/state/` if not exists
4. Create `~/.tabs/sessions/<today>/` if not exists

**Daily Rotation:**
- At midnight (00:00), create new date directory
- No cleanup of old directories (user's responsibility or future feature)

**Log Rotation:**
- Write to `~/.tabs/daemon.log`
- Rotate daily (keep last 7 days)

---

### 3. tabs-ui-local (TanStack Start App)

**Responsibilities:**
- Browse local sessions
- Search and filter sessions
- View session details (messages, tools, thinking blocks)
- Share sessions to remote server
- Configure API key and server URL

**Server:**
- Runs on `http://localhost:3787`
- TanStack Start SSR + React frontend
- Direct filesystem access to read JSONL
- Bound to loopback only and enforces Origin/Host checks for mutating requests

**Key Features:**

#### Timeline View
- List all sessions, newest first
- Group by date (Today, Yesterday, Last 7 days, Older)
- Show: session_id, tool, cwd, timestamp, duration, message count
- Filter by: date range, tool (claude-code / cursor), cwd
- Search: full-text search in session messages (client-side)

#### Session Detail View
- Chronological message flow
- Syntax-highlighted code blocks
- Collapsible tool use sections (input/output)
- Thinking blocks (toggleable)
- File references (click to copy path)
- "Share" button in header

#### Share Workflow
1. Click "Share" on session
2. Modal appears:
   - Session summary (id, tool, cwd, duration)
   - Tag input: `team:platform`, `repo:myapp`, `category:bugfix`
   - Confirm button
3. On confirm:
   - Send request to daemon (via Unix socket or HTTP)
   - Daemon reads session JSONL
   - Daemon POSTs to remote server with API key
   - Show success message with link to remote session

#### Settings Page
- Configure remote server URL
- Configure API key (masked input)
- Test connection button
- View daemon status
- View storage usage

**Data Loading:**
- On page load, read all JSONL files from `~/.tabs/sessions/`
- Parse into memory (sessions are small, <1MB typically)
- Build index for search
- Use React state for filters

---

### 4. tabs-server (Go HTTP Server)

**Responsibilities:**
- Accept session uploads from local clients
- Store sessions in PostgreSQL
- Serve web UI for browsing shared sessions
- Manage API keys
- Search and filter sessions
- Tag-based discovery

**Deployment:**
- Docker container
- Exposed on port 8080 (or custom via env var)
- Behind IAP (Cloudflare Access, Google IAP, etc.)

**Key Features:**

#### API Endpoints

**POST /api/sessions** (Public, API key auth)
- Upload a session from local client
- Request body:
```json
{
  "session": {
    "session_id": "550e8400-...",
    "tool": "claude-code",
    "created_at": "2026-01-28T12:00:00Z",
    "ended_at": "2026-01-28T12:05:00Z",
    "cwd": "/home/user/projects/myapp",
    "events": [
      // Array of JSONL events
    ]
  },
  "tags": [
    {"key": "team", "value": "platform"},
    {"key": "repo", "value": "myapp"}
  ]
}
```
- Response: `{"id": "<uuid>", "url": "https://tabs.company.com/sessions/<uuid>"}`

**GET /api/sessions** (IAP protected)
- List sessions with pagination and filters
- Query params: `?page=1&limit=20&tool=claude-code&tag=team:platform`
- Response: Array of session metadata (no full events)

**GET /api/sessions/:id** (IAP protected)
- Get full session details including all events
- Response: Session object with events array

**POST /api/keys** (IAP protected)
- Create new API key for authenticated user
- Request: `{"name": "My Laptop"}`
- Response: `{"key": "tabs_xxxxxxxxxxxxxxxx", "created_at": "..."}`

**GET /api/tags** (IAP protected)
- List all unique tags with counts
- Response: `[{"key": "team", "value": "platform", "count": 42}, ...]`

#### Authentication

**API Key Authentication (for POST /api/sessions):**
- Header: `Authorization: Bearer tabs_xxxxxxxxxxxxxxxx`
- Middleware validates key against `api_keys` table
- If invalid: return 401 Unauthorized

**IAP Authentication (for all other routes):**
- IAP injects user identity via headers (e.g., `X-Forwarded-User`)
- Server reads identity from headers
- Associates actions with user (for audit log)

#### Web UI Routes
- `GET /` - Homepage (timeline of all sessions)
- `GET /sessions/:id` - Session detail view
- `GET /search` - Search page with filters
- `GET /keys` - API key management page

**UI Tech Stack:**
- TanStack Start (SSR)
- React (frontend)
- Tailwind CSS (styling)
- Shadcn UI (components)

#### Database Schema (See Data Format SPEC)

**Key Queries:**
- List sessions: `SELECT * FROM sessions ORDER BY created_at DESC LIMIT 20`
- Search by tags: `SELECT s.* FROM sessions s JOIN tags t ON s.id = t.session_id WHERE t.tag_key = 'team' AND t.tag_value = 'platform'`
- Full session: `SELECT * FROM messages WHERE session_id = $1 ORDER BY seq`

---

## Data Flow

### Flow 1: Capture Session (Claude Code)

```
┌─────────────┐
│ User types  │
│ prompt in   │
│ Claude Code │
└──────┬──────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Claude Code SessionStart hook fires          │
│ Executes: tabs-cli capture-event             │
│           --tool=claude-code < payload.json  │
└──────┬───────────────────────────────────────┘
       │
       │ Payload: {session_id, transcript_path, cwd, ...}
       ▼
┌──────────────────────────────────────────────┐
│ tabs-cli receives JSON via stdin             │
│ - Checks if daemon is running                │
│ - Reads PID file ~/.tabs/daemon.pid          │
│ - Checks process: kill -0 <pid>              │
└──────┬───────────────────────────────────────┘
       │
       ▼
    ┌─────┐
    │ PID │ No
    │file?├────► Start daemon: tabs-daemon &
    └──┬──┘     Wait for socket (max 2s)
       │ Yes
       ▼
┌──────────────────────────────────────────────┐
│ Check if process alive: kill -0 <pid>        │
└──────┬───────────────────────────────────────┘
       │
       ▼
    ┌──────┐
    │Alive?│ No  ► Remove stale PID file
    └──┬───┘       Start new daemon
       │ Yes
       ▼
┌──────────────────────────────────────────────┐
│ Connect to Unix socket ~/.tabs/daemon.sock   │
│ Send JSON message: {type, tool, payload}     │
│ Receive response: {status: "ok"}             │
│ Close connection                              │
│ Exit (fast, <50ms)                            │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ tabs-daemon receives event                   │
│ - Extracts session_id, transcript_path       │
│ - Reads JSONL from transcript_path           │
│ - Parses into events                          │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Determine event type:                        │
│ - First message? → session_start             │
│ - User/assistant message? → message          │
│ - Tool use? → tool_use                       │
│ - Tool result? → tool_result                 │
│ - Session complete? → session_end            │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Generate filename:                           │
│ <session-id>-claude-code-<timestamp>.jsonl   │
│ Path: ~/.tabs/sessions/2026-01-28/...        │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Append event to JSONL (atomic):              │
│ - Open file in append mode                   │
│ - Acquire file lock                          │
│ - Write JSON + newline                       │
│ - Fsync to disk                              │
│ - Release lock                               │
└──────────────────────────────────────────────┘
```

### Flow 2: Browse Sessions (Local UI)

```
┌─────────────┐
│ User opens  │
│ browser to  │
│ localhost:  │
│ 3787        │
└──────┬──────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ TanStack Start app loads                     │
│ Server-side: Read ~/.tabs/sessions/          │
│ - Scan all date directories                  │
│ - Read all *.jsonl files                     │
│ - Parse JSONL into session objects           │
│ - Sort by created_at DESC                    │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Render timeline view (React)                 │
│ - Group by date (Today, Yesterday, ...)      │
│ - Show session cards with metadata           │
│ - Filter controls (date, tool, cwd)          │
│ - Search input (client-side filter)          │
└──────┬───────────────────────────────────────┘
       │
       │ User clicks session
       ▼
┌──────────────────────────────────────────────┐
│ Navigate to /sessions/:id                    │
│ Server-side: Read specific JSONL file        │
│ - Parse all events                           │
│ - Build message thread                       │
│ - Extract tools and results                  │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Render session detail (React)                │
│ - Chronological message flow                 │
│ - Syntax-highlighted code                    │
│ - Collapsible tool sections                  │
│ - Thinking blocks (toggleable)               │
│ - Share button in header                     │
└──────────────────────────────────────────────┘
```

### Flow 3: Share Session to Remote Server

```
┌─────────────┐
│ User clicks │
│ "Share"     │
│ button      │
└──────┬──────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Modal opens with tag input                   │
│ User enters tags: team:platform, repo:myapp  │
│ Clicks "Confirm"                             │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ UI sends request to local server:            │
│ POST /api/push                               │
│ Body: {session_id, tags}                     │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Local server connects to daemon via socket   │
│ Sends: {type: "push", session_id, tags}      │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Daemon receives push request                 │
│ - Reads session JSONL file                   │
│ - Parses all events                          │
│ - Reads API key from ~/.tabs/config.toml     │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Daemon POSTs to remote server                │
│ URL: https://tabs.company.com/api/sessions   │
│ Header: Authorization: Bearer <api-key>      │
│ Body: {session, tags}                        │
└──────┬───────────────────────────────────────┘
       │
       │ HTTPS
       ▼
┌──────────────────────────────────────────────┐
│ tabs-server receives POST /api/sessions      │
│ - Validates API key (query api_keys table)   │
│ - Parses session payload                     │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Store in PostgreSQL (transaction):           │
│ - INSERT INTO sessions (...)                 │
│ - INSERT INTO messages (...) -- batch insert │
│ - INSERT INTO tools (...)    -- batch insert │
│ - INSERT INTO tags (...)     -- batch insert │
│ - COMMIT                                      │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Return response:                             │
│ {id: "<uuid>",                               │
│  url: "https://tabs.company.com/sessions/id"}│
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Local UI shows success message:              │
│ "Session shared! View at [URL]"              │
│ Copies URL to clipboard                      │
└──────────────────────────────────────────────┘
```

### Flow 4: Browse Shared Sessions (Remote UI)

```
┌─────────────┐
│ User visits │
│ https://    │
│ tabs.       │
│ company.com │
└──────┬──────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ IAP intercepts request                       │
│ - Checks authentication (SSO, Google, etc.)  │
│ - If not authenticated: redirect to login    │
│ - If authenticated: inject user identity     │
│   via header (X-Forwarded-User)              │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ tabs-server receives GET /                   │
│ - Reads user identity from header            │
│ - Queries PostgreSQL:                        │
│   SELECT * FROM sessions                     │
│   ORDER BY created_at DESC LIMIT 20          │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Render timeline view (TanStack Start)        │
│ - Session cards with metadata                │
│ - Search bar (server-side search)            │
│ - Tag filters (team, repo, etc.)             │
│ - Pagination controls                        │
└──────┬───────────────────────────────────────┘
       │
       │ User clicks session
       ▼
┌──────────────────────────────────────────────┐
│ Navigate to /sessions/:id                    │
│ Server queries:                              │
│ - SELECT * FROM sessions WHERE id = $1       │
│ - SELECT * FROM messages WHERE session_id=$1 │
│   ORDER BY seq                               │
│ - SELECT * FROM tools WHERE session_id = $1  │
└──────┬───────────────────────────────────────┘
       │
       ▼
┌──────────────────────────────────────────────┐
│ Render session detail                        │
│ - Full message thread                        │
│ - Tools and results                          │
│ - Tags displayed                             │
│ - Metadata (tool, cwd, uploaded_by)          │
│ - No "Share" button (already shared)         │
└──────────────────────────────────────────────┘
```

---

## Process Lifecycle

### Daemon Startup

```
User runs: claude (Claude Code command)
    ↓
SessionStart hook fires
    ↓
Hook executes: tabs-cli capture-event --tool=claude-code
    ↓
tabs-cli reads ~/.tabs/daemon.pid
    ↓
  ┌─────────┐
  │ PID file│ No  ──► Start daemon
  │ exists? │         Fork: tabs-daemon &
  └────┬────┘         Wait for socket (max 2s)
       │ Yes
       ▼
  ┌─────────┐
  │Process  │ No  ──► Remove stale PID file
  │ alive?  │         Start new daemon
  │(kill -0)│
  └────┬────┘
       │ Yes
       ▼
  Connect to socket
  Send event
  Exit
```

### Daemon Runtime

```
Daemon process (PID 12345)
    │
    ├─► Main goroutine: Unix socket server
    │   ├─ Listen on ~/.tabs/daemon.sock
    │   ├─ Accept connections
    │   ├─ Read JSON message
    │   ├─ Dispatch to handler
    │   └─ Send response
    │
    ├─► Event handler goroutines (pool of 10)
    │   ├─ Process Claude Code events
    │   ├─ Read transcript JSONL
    │   ├─ Parse and append to local JSONL
    │   └─ Log errors
    │
    ├─► Cursor poller goroutine
    │   ├─ Every 2 seconds
    │   ├─ Check if any pending conversations
    │   ├─ Query SQLite DB for AI responses
    │   ├─ Append to local JSONL
    │   └─ Log errors
    │
    ├─► Signal handler goroutine
    │   ├─ Wait for SIGTERM / SIGINT
    │   ├─ Set shutdown flag
    │   ├─ Close socket
    │   ├─ Wait for in-flight events (max 5s)
    │   ├─ Remove PID file
    │   └─ Exit
    │
    └─► Log rotation goroutine
        ├─ At midnight (00:00)
        ├─ Rotate ~/.tabs/daemon.log
        └─ Keep last 7 days
```

### Daemon Shutdown

**Graceful:**
```
User sends: kill <pid>  (SIGTERM)
    ↓
Signal handler catches SIGTERM
    ↓
Set shutdown flag (atomic bool)
    ↓
Close Unix socket (new connections rejected)
    ↓
Wait for in-flight events to complete (max 5s timeout)
    ↓
Flush any buffered writes
    ↓
Remove ~/.tabs/daemon.pid
    ↓
Remove ~/.tabs/daemon.sock
    ↓
Log: "Daemon shutdown gracefully"
    ↓
Exit(0)
```

**Ungraceful (crash):**
```
Daemon crashes (panic, SIGKILL, OOM, etc.)
    ↓
PID file remains: ~/.tabs/daemon.pid
Socket may remain: ~/.tabs/daemon.sock
    ↓
Next CLI invocation:
    ↓
Read PID from file
    ↓
Check if process alive: kill -0 <pid>
    ↓
Process not found (stale PID)
    ↓
Remove stale PID file
Remove stale socket file
    ↓
Start new daemon
    ↓
Write new PID
Create new socket
    ↓
Continue normally
```

---

## Authentication & Security

### API Key Flow

**Key Creation (on remote server):**
1. User visits `https://tabs.company.com/keys` (IAP protected)
2. IAP authenticates user (SSO, Google, etc.)
3. User clicks "Create API Key"
4. Server generates: `tabs_` + 32 random hex chars
5. Server stores: `INSERT INTO api_keys (key_hash, user_id, created_at)`
6. Server displays key ONCE: "tabs_abc123... (copy and save, won't show again)"
7. User copies key

**Key Storage (local):**
```bash
# Via CLI:
tabs-cli config set api-key tabs_abc123...

# Stored in ~/.tabs/config.toml:
[remote]
server_url = "https://tabs.company.com"
api_key = "tabs_abc123..."
```

**Key Usage:**
- When pushing session, daemon reads API key from config
- Includes in request: `Authorization: Bearer tabs_abc123...`
- Remote server validates by hashing and querying `api_keys` table

### Remote Server Access Control

**IAP (Identity-Aware Proxy):**
- All routes except `/api/sessions` (POST) are behind IAP
- IAP options: Cloudflare Access, Google IAP, Auth0, OAuth2 Proxy
- IAP injects user identity via headers (e.g., `X-Forwarded-User: alice@company.com`)

**API Endpoint:**
- `POST /api/sessions` is **publicly accessible** (not behind IAP)
- Protected by API key authentication
- Rate limited (e.g., 100 requests/hour per key)

### Security Considerations

1. **Secrets in sessions:**
   - User responsibility to review before sharing
   - Future: auto-detect patterns (API keys, tokens) and warn

2. **API key compromise:**
   - User can revoke keys via web UI
   - Server logs all uploads with key ID for audit

3. **Transport:**
   - Local: Unix socket (no network exposure)
   - Remote: HTTPS only (TLS 1.3+)

4. **Data at rest:**
   - Local JSONL files: user's filesystem permissions
   - Remote PostgreSQL: encryption depends on deployment (RDS, Cloud SQL, etc.)

---

## Deployment

### Local (User Machine)

**Installation (Linux/macOS):**
```bash
# Download binaries
curl -L https://github.com/yourorg/tabs/releases/latest/download/tabs-cli-$(uname -s)-$(uname -m) -o /usr/local/bin/tabs-cli
chmod +x /usr/local/bin/tabs-cli

curl -L https://github.com/yourorg/tabs/releases/latest/download/tabs-daemon-$(uname -s)-$(uname -m) -o /usr/local/bin/tabs-daemon
chmod +x /usr/local/bin/tabs-daemon

# Install hooks
tabs-cli install

# Output:
# ✓ Hooks installed for Claude Code (~/.claude/config.yaml)
# ✓ Hooks installed for Cursor (~/.cursor/hooks.json)
# ✓ tabs is ready to capture sessions!
```

**Local UI:**
```bash
# Start local web UI
tabs-cli ui

# Output:
# ✓ Local UI running at http://localhost:3787
# ✓ Press Ctrl+C to stop
```

**Directory Permissions:**
- `~/.tabs/` - 0700 (owner only)
- `~/.tabs/sessions/` - 0700
- `~/.tabs/state/` - 0700
- `~/.tabs/daemon.sock` - 0600
- `~/.tabs/config.toml` - 0600 (contains API key)

### Remote (Docker Container)

**Dockerfile:**
```dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tabs-server ./cmd/tabs-server

FROM alpine:3.19
RUN apk --no-cache add ca-certificates

COPY --from=builder /build/tabs-server /usr/local/bin/
COPY --from=builder /build/ui/dist /usr/local/share/tabs-ui

EXPOSE 8080

CMD ["tabs-server"]
```

**docker-compose.yml:**
```yaml
version: '3.8'

services:
  tabs-server:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgresql://tabs:password@postgres:5432/tabs
      PORT: 8080
      LOG_LEVEL: info
    depends_on:
      - postgres

  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: tabs
      POSTGRES_USER: tabs
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

volumes:
  postgres_data:
```

**Deployment Steps:**
1. Build Docker image: `docker build -t tabs-server:latest .`
2. Push to registry: `docker push yourorg/tabs-server:latest`
3. Deploy to cluster (Kubernetes, ECS, etc.)
4. Configure IAP to protect all routes except `/api/sessions`
5. Point DNS to load balancer

**Environment Variables:**
```bash
DATABASE_URL=postgresql://tabs:password@postgres:5432/tabs
PORT=8080
LOG_LEVEL=info  # debug, info, warn, error
CORS_ORIGINS=https://tabs.company.com
RATE_LIMIT_RPM=100  # requests per minute per API key
```

---

## Scaling Considerations

### Local System
- **Sessions per day:** 10-50 (typical user)
- **Storage per session:** 10KB-1MB (mostly text)
- **Daily storage growth:** ~10MB
- **Annual storage:** ~3.6GB
- **Performance:** JSONL read/write is fast (<1ms), no bottleneck

### Remote Server
- **Users:** 100-1000 (small to medium org)
- **Sessions per day:** 1000-10000
- **Storage per session:** 100KB (with tools, thinking blocks)
- **Daily storage growth:** 100MB-1GB
- **Annual storage:** 36GB-365GB
- **PostgreSQL:** Single instance sufficient (<100GB DB)
- **Scaling:** Add read replicas if needed (unlikely)

**Database Optimization:**
- Partition `messages` table by session_id (if >1M sessions)
- Archive old sessions (>1 year) to cold storage (S3, GCS)

---

## Error Handling

### CLI Errors
- **Daemon not starting:** Log error, suggest checking logs (`~/.tabs/daemon.log`)
- **Hook installation fails:** Check permissions, suggest manual edit
- **Push fails (no API key):** Prompt user to run `tabs-cli config set api-key`

### Daemon Errors
- **JSONL write fails:** Log error, retry once, then skip event (don't crash)
- **Cursor SQLite read fails:** Log error, continue polling (don't crash)
- **Socket bind fails:** Check if daemon already running, exit with error

### Remote Server Errors
- **Invalid API key:** Return 401 with message "Invalid or expired API key"
- **Duplicate session:** Return 409 with message "Session already uploaded"
- **Database error:** Return 500, log error, retry transaction once

---

## Monitoring & Observability

### Local
- **Daemon logs:** `~/.tabs/daemon.log` (rotation, keep 7 days)
- **CLI logs:** Stderr (user sees errors immediately)
- **Metrics:** None (not needed for local system)

### Remote
- **Structured logging:** JSON logs to stdout (captured by container runtime)
- **Metrics:** Prometheus metrics exposed on `/metrics` (IAP protected)
  - `tabs_sessions_uploaded_total` - Counter
  - `tabs_api_requests_total{endpoint, status}` - Counter
  - `tabs_db_query_duration_seconds{query}` - Histogram
- **Tracing:** Optional (OpenTelemetry) for debugging
- **Alerting:** Prometheus Alertmanager
  - Alert: Database connection failures
  - Alert: High API error rate (>5% 5xx)

---

## Future Enhancements (Out of Scope for v1)

1. **Secret detection:** Auto-detect API keys, tokens, passwords in sessions before sharing
2. **Session diffing:** Compare two sessions side-by-side
3. **Export formats:** Export session as PDF, HTML, or Markdown
4. **Team analytics:** Aggregated stats (most common prompts, tools used, etc.)
5. **Comments:** Allow team members to comment on shared sessions
6. **Session forking:** Start new session from specific point in old session (like Cursor Enterprise)
7. **AI summarization:** Auto-generate summary of session (what was accomplished)
8. **Integration with PRs:** Link sessions to GitHub PR comments
9. **Multi-user collaboration:** Multiple users working on same session (shared session ID)
10. **Retention policies:** Auto-delete old sessions after X days

---

## Conclusion

This architecture provides:
- ✅ **Local-first** - All sessions captured locally, zero dependency on remote
- ✅ **Real-time** - Hook-based capture, immediate JSONL writes
- ✅ **Resilient** - PID file concurrency control, auto-recovery from crashes
- ✅ **Secure** - Unix socket (local), API key + IAP (remote)
- ✅ **Simple** - JSONL storage, no complex indexing, direct filesystem reads
- ✅ **Transparent** - All shared sessions visible to everyone
- ✅ **Extensible** - Provider pattern supports future tools (VS Code, Windsurf, etc.)

**Next Steps:**
1. Generate Data Format SPEC (JSONL schema, PostgreSQL schema)
2. Generate API Design SPEC (Unix socket protocol, HTTP APIs)
3. Begin implementation (Phase 1: CLI + Daemon + Claude Code)

---

**Document Status:** Ready for review
**Last Updated:** 2026-01-28
