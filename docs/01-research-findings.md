# Research Findings: tabs - AI Session Capture System

**Date:** 2026-01-28
**Project:** tabs (tablatures for AI coding sessions)
**Goal:** Build an open-source system to capture, store, and share prompts/outputs from Claude Code and Cursor

---

## Executive Summary

This document synthesizes research on how to integrate with Claude Code and Cursor IDE to capture AI coding sessions, based on analysis of getspecstory and Cursor ecosystem.

**Key Findings:**
1. **Claude Code** - Has mature hook system with direct transcript access (JSONL files)
2. **Cursor** - Has newer hook system (v1.7+) but stores sessions in SQLite, requires hybrid approach
3. **Storage Strategy** - JSONL locally (append-only, greppable), PostgreSQL for central server
4. **Architecture** - Provider pattern with adapters for each tool, unified schema

---

## 1. Claude Code Integration

### Hook System

Claude Code provides a **mature hook system** via `~/.claude/config.yaml`:

**Available Hooks:**
- `SessionStart` - Fired when new session begins
- `SessionEnd` - Fired when session completes
- `PromptSubmit` - Before each prompt is sent
- `ToolUse` - When tools are invoked
- And more...

**Hook Payload Example (SessionEnd):**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "transcript_path": "/home/user/.claude/projects/abc123/550e8400.jsonl",
  "cwd": "/home/user/projects/myapp",
  "timestamp": "2026-01-28T12:00:00Z",
  "permission_mode": "ask",
  "fileContext": {
    "read": ["src/main.ts"],
    "modified": ["src/auth.ts"],
    "created": ["src/new.ts"]
  }
}
```

**Key Advantage:** `transcript_path` points directly to the JSONL file containing full session history.

### Storage Format

**Location:** `~/.claude/projects/<project-hash>/<session-id>.jsonl`

**Format:** JSONL (one JSON object per line), chronologically ordered

**Record Structure:**
```json
{
  "type": "user" | "assistant",
  "timestamp": "2026-01-28T12:00:00.000Z",
  "content": [
    { "type": "text", "text": "..." },
    { "type": "thinking", "text": "..." }
  ],
  "tool_use": {
    "id": "toolu_abc123",
    "name": "write",
    "input": { "file_path": "/path/to/file", "content": "..." }
  },
  "tool_result": {
    "tool_use_id": "toolu_abc123",
    "content": "...",
    "is_error": false
  }
}
```

**Session Boundaries:**
- Each file = one session
- Session ID is stable UUID
- `/new` command or tool restart = new session ID

### Integration Strategy for tabs

**Real-time capture via hooks:**

1. Install hook in `~/.claude/config.yaml`:
```yaml
hooks:
  SessionStart:
    - command: "tabs-cli capture-event --tool=claude-code"
  SessionEnd:
    - command: "tabs-cli capture-event --tool=claude-code"
  ToolUse:
    - command: "tabs-cli capture-event --tool=claude-code"
```

2. Hook receives JSON via stdin
3. `tabs-cli` sends event to daemon (auto-starts if needed)
4. Daemon reads JSONL from `transcript_path` and stores locally

**Advantages:**
- Direct access to full transcript via `transcript_path`
- Mature, stable hook system
- JSONL is easy to parse and append
- No database polling needed

---

## 2. Cursor Integration

### Hook System

Cursor provides a **hook system** (since v1.7, ~Oct 2024) via `.cursor/hooks.json`:

**Available Hooks:**
- `beforeSubmitPrompt` - Before user prompt is sent to model
- `afterFileEdit` - After Cursor modifies a file
- `beforeShellExecution` - Before executing shell command
- `stop` - When task/generation completes

**Hook Payload Example (beforeSubmitPrompt):**
```json
{
  "conversation_id": "668320d2-2fd8-4888-b33c-2a466fec86e7",
  "generation_id": "490b90b7-a2ce-4c2c-bb76-cb77b125df2f",
  "prompt": "implement user authentication",
  "attachments": [
    { "type": "file", "file_path": "src/auth.ts" }
  ],
  "hook_event_name": "beforeSubmitPrompt",
  "workspace_roots": ["/Users/user/projects/myapp"]
}
```

**Configuration Location:**
- Per-project: `.cursor/hooks.json`
- Global: `~/.cursor/hooks.json`

### Storage Format

**Location (macOS):** `~/Library/Application Support/Cursor/user/globalStorage/state.vscdb`
**Location (Linux):** `~/.config/Cursor/User/globalStorage/state.vscdb`

**Format:** SQLite database with JSON blobs

**Database Schema:**
```sql
-- Table: ItemTable
-- Key column contains identifiers like:
--   - 'aiService.prompts'
--   - 'aiService.generations'
--   - 'workbench.panel.aichat.view.aichat.chatdata'
-- Value column contains JSON blobs

SELECT rowid, [key], value
FROM ItemTable
WHERE [key] IN (
  'aiService.prompts',
  'workbench.panel.aichat.view.aichat.chatdata'
)
```

**Session Structure (from JSON blobs):**
```json
{
  "conversation_id": "...",
  "messages": [
    {
      "role": "user" | "assistant",
      "content": "...",
      "timestamp": "..."
    }
  ],
  "attachments": [...],
  "metadata": {...}
}
```

### Integration Strategy for tabs

**Hybrid approach: Hooks + Database polling**

1. **Real-time events via hooks:**
   - Capture `beforeSubmitPrompt` for user input
   - Capture `stop` for completion signal
   - Store conversation_id + generation_id for correlation

2. **AI responses via database polling:**
   - Watch `state.vscdb` for modifications
   - Query for new/updated conversations
   - Parse JSON blobs and extract AI responses

3. **Hook installation:**
```json
{
  "version": 1,
  "hooks": {
    "beforeSubmitPrompt": [
      { "command": "tabs-cli capture-event --tool=cursor" }
    ],
    "stop": [
      { "command": "tabs-cli capture-event --tool=cursor" }
    ]
  }
}
```

**Challenges:**
- Hooks don't provide full transcript path (unlike Claude Code)
- Need to query SQLite database separately
- JSON blobs require parsing
- Database schema may change with Cursor updates

**Future Option: Cursor Enterprise API**
- Cursor Enterprise has built-in transcript export
- Service accounts with API keys
- Could query transcripts programmatically
- Not yet publicly documented (as of Jan 2025)

---

## 3. getspecstory Architecture Analysis

### Overview

getspecstory uses a **provider pattern** with file-watching instead of hooks.

**Key Components:**
1. **SPI (Service Provider Interface)** - Abstract provider interface
2. **Provider implementations** - claudecode, cursorcli, codexcli, geminicli
3. **File watcher (fsnotify)** - Monitors JSONL changes in real-time
4. **JSONL parser** - Converts to unified SessionData schema
5. **Local storage** - Markdown files in `.specstory/history/`
6. **Cloud sync** - Optional push to SpecStory cloud

### Provider Interface

```go
type Provider interface {
    Name() string
    DetectAgent(projectPath string) bool
    GetAgentChatSession(projectPath, sessionID string) (*AgentChatSession, error)
    GetAgentChatSessions(projectPath string) ([]AgentChatSession, error)
    ExecAgentAndWatch(projectPath, customCommand, resumeSessionID string,
                      callback func(*AgentChatSession)) error
    WatchAgent(ctx context.Context, projectPath string,
               callback func(*AgentChatSession)) error
}
```

### Unified Schema

```go
type AgentChatSession struct {
    SchemaVersion string
    Provider      ProviderInfo
    SessionID     string
    CreatedAt     string // ISO 8601
    UpdatedAt     string
    Slug          string // Human-readable identifier
    WorkspaceRoot string
    Exchanges     []Exchange
    RawData       interface{} // Provider-specific raw data
}

type Exchange struct {
    ExchangeID string // "sessionId:index"
    StartTime  string
    EndTime    string
    Messages   []Message
}

type Message struct {
    ID        string
    Timestamp string
    Role      string // "user" or "agent"
    Model     string
    Content   []ContentPart // text and/or thinking
    Tool      *ToolInfo
    PathHints []string
}

type ToolInfo struct {
    Name              string // "write", "read", "bash", etc.
    Type              string // "write", "read", "search", "shell", "task"
    UseID             string
    Input             map[string]interface{}
    Output            map[string]interface{}
    Summary           *string
    FormattedMarkdown *string
}
```

### File Watching Strategy

**For Claude Code:**
```go
// Watch ~/.claude/projects/<hash>/ for JSONL file changes
watcher.Add(claudeProjectDir)

for {
    select {
    case event := <-watcher.Events:
        if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
            session := parseJSONL(event.Name)
            callback(session) // Invoke callback with updated session
        }
    }
}
```

**Callback invocation (non-blocking):**
```go
go func(s *AgentChatSession) {
    defer func() {
        if r := recover(); r != nil {
            log.Error("Callback panicked", "panic", r)
        }
    }()
    callback(s)
}(session)
```

### Key Design Patterns

1. **Deduplication** - Handle resumed sessions (same sessionID, multiple files)
2. **Warmup filtering** - Skip sidechain warmup messages
3. **Tool result merging** - Match tool uses to results by `tool_use_id`
4. **Debouncing** - 10-second interval to prevent excessive callbacks
5. **Raw data preservation** - Store provider-specific format for debugging

### Storage Strategy

**Local:**
```
<project-root>/.specstory/
├── history/
│   ├── 2026-01-28_14-30-00_fix-auth.md
│   └── 2026-01-28_15-45-23_implement-feature.md
├── debug/
│   └── <session-uuid>/
│       ├── session-data.json
│       └── raw.jsonl
└── .project.json
```

**Cloud:**
- API request with markdown + raw data + metadata
- SHA256 hash comparison to skip duplicates
- Supports project tagging and search

---

## 4. Key Differences: getspecstory vs tabs

| Aspect | getspecstory | tabs (our approach) |
|--------|-------------|-------------------|
| **Integration** | File watching (fsnotify) | Hook-based (direct events) |
| **Claude Code** | Watch `~/.claude/projects/` | Use SessionStart/SessionEnd hooks with `transcript_path` |
| **Cursor** | File watcher + SQLite parsing | Hooks + SQLite polling |
| **Storage** | Markdown + debug JSON | JSONL locally, PostgreSQL for server |
| **Output format** | Markdown-first | JSONL-first, render on demand |
| **Architecture** | CLI + cloud sync | Daemon + local web UI + remote server |
| **Real-time** | Via file watcher callbacks | Via hook callbacks |
| **Session boundary** | File-based detection | Hook lifecycle events |

**Why hooks over file watching for tabs:**
1. **Direct event notification** - No polling, immediate capture
2. **Session lifecycle signals** - Know exactly when sessions start/end
3. **Rich metadata** - Hooks provide context (cwd, permissions, file changes)
4. **Future-proof** - Claude Code's official integration method
5. **Lower resource usage** - No continuous filesystem monitoring

---

## 5. Recommended Architecture for tabs

### Component Overview

```
┌─────────────────────────────────────────────────────┐
│                  tabs Ecosystem                      │
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────────────┐      ┌──────────────┐           │
│  │  Claude Code │      │    Cursor    │           │
│  │    Hooks     │      │    Hooks     │           │
│  └──────┬───────┘      └──────┬───────┘           │
│         │                     │                     │
│         ├─────────┬───────────┤                     │
│         ▼         ▼           ▼                     │
│  ┌─────────────────────────────────┐               │
│  │       tabs-cli (Go)             │               │
│  │  - Captures hook events         │               │
│  │  - Auto-starts daemon           │               │
│  │  - Sends events to daemon       │               │
│  └──────────────┬──────────────────┘               │
│                 │                                    │
│                 ▼                                    │
│  ┌─────────────────────────────────┐               │
│  │      tabs-daemon (Go)           │               │
│  │  - Receives events from CLI     │               │
│  │  - Reads transcripts (JSONL)    │               │
│  │  - Polls Cursor SQLite DB       │               │
│  │  - Writes to local JSONL        │               │
│  │  - Provides HTTP API            │               │
│  └──────────────┬──────────────────┘               │
│                 │                                    │
│                 ▼                                    │
│  ┌─────────────────────────────────┐               │
│  │    Local Storage (JSONL)        │               │
│  │  ~/.tabs/sessions/              │               │
│  │    └─ YYYY-MM-DD/               │               │
│  │        └─ <session>-<rand>.jsonl│               │
│  └──────────────┬──────────────────┘               │
│                 │                                    │
│        ┌────────┴────────┐                          │
│        ▼                 ▼                          │
│  ┌──────────┐    ┌──────────────┐                 │
│  │ Local UI │    │   tabs-cli   │                 │
│  │(TanStack)│    │   push cmd   │                 │
│  │  - Browse│    │              │                 │
│  │  - Search│    │              │                 │
│  │  - Share │───►│              │                 │
│  └──────────┘    └──────┬───────┘                 │
│                          │                          │
└──────────────────────────┼──────────────────────────┘
                           │
                           │ HTTPS
                           ▼
                 ┌──────────────────┐
                 │  tabs-server (Go)│
                 │  - Receive pushed│
                 │    sessions      │
                 │  - Store in PG   │
                 │  - Serve web UI  │
                 │  - Search/browse │
                 │  - Tag filtering │
                 └──────────────────┘
                           │
                           ▼
                 ┌──────────────────┐
                 │   PostgreSQL     │
                 │  - Sessions      │
                 │  - Messages      │
                 │  - Tags          │
                 └──────────────────┘
```

### Data Flow

**Capture Flow (Claude Code):**
1. User submits prompt in Claude Code
2. SessionStart hook fires → calls `tabs-cli capture-event`
3. `tabs-cli` checks if daemon is running (via socket/port check)
4. If not running, `tabs-cli` spawns daemon
5. `tabs-cli` sends event to daemon via HTTP POST
6. Daemon receives session_id + transcript_path
7. Daemon reads JSONL from transcript_path
8. Daemon appends to `~/.tabs/sessions/YYYY-MM-DD/<session-id>-<rand>.jsonl`
9. On SessionEnd, mark session complete

**Capture Flow (Cursor):**
1. User submits prompt in Cursor
2. beforeSubmitPrompt hook fires → calls `tabs-cli capture-event`
3. `tabs-cli` sends event with conversation_id + prompt
4. Daemon stores event
5. Daemon polls `state.vscdb` for AI response (background goroutine)
6. Daemon matches by conversation_id + generation_id
7. Daemon appends to `~/.tabs/sessions/YYYY-MM-DD/<conversation-id>-<rand>.jsonl`
8. On stop hook, mark conversation complete

**Browse Flow:**
1. User opens `http://localhost:3787` (local UI)
2. TanStack Start app reads from `~/.tabs/sessions/` directly
3. Displays timeline view (newest first)
4. User can filter by date, cwd, tool
5. Click session → detailed view with all messages/tools

**Share Flow:**
1. User clicks "Share" on a session in local UI
2. UI POSTs to local daemon `/api/share/<session-id>`
3. Daemon reads session JSONL
4. User tags session (team, repo, etc.) in modal
5. Daemon POSTs to remote server `https://tabs.company.com/api/sessions`
6. Remote server stores in PostgreSQL
7. Remote server returns URL → UI shows success + link

---

## 6. Local Storage Schema

### Directory Structure

```
~/.tabs/
├── sessions/
│   ├── 2026-01-28/
│   │   ├── 550e8400-e29b-claude-abc123.jsonl
│   │   ├── 668320d2-2fd8-cursor-def456.jsonl
│   │   └── ...
│   ├── 2026-01-29/
│   │   └── ...
│   └── ...
├── daemon.pid         # Daemon process ID
├── daemon.log         # Daemon logs
└── config.toml        # User config (server URL, etc.)
```

### JSONL Record Format

Each line in session JSONL file:

```json
{
  "event_type": "session_start" | "message" | "tool_use" | "tool_result" | "session_end",
  "timestamp": "2026-01-28T12:00:00.000Z",
  "tool": "claude-code" | "cursor",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    // Event-specific data
  }
}
```

**Event Types:**

**session_start:**
```json
{
  "event_type": "session_start",
  "timestamp": "2026-01-28T12:00:00.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-...",
  "data": {
    "cwd": "/home/user/projects/myapp",
    "permission_mode": "ask"
  }
}
```

**message:**
```json
{
  "event_type": "message",
  "timestamp": "2026-01-28T12:00:05.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-...",
  "data": {
    "role": "user" | "assistant",
    "content": [
      { "type": "text", "text": "..." },
      { "type": "thinking", "text": "..." }
    ],
    "model": "claude-sonnet-4-5-20250929"
  }
}
```

**tool_use:**
```json
{
  "event_type": "tool_use",
  "timestamp": "2026-01-28T12:00:10.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-...",
  "data": {
    "tool_use_id": "toolu_abc123",
    "tool_name": "write",
    "input": {
      "file_path": "/path/to/file",
      "content": "..."
    }
  }
}
```

**tool_result:**
```json
{
  "event_type": "tool_result",
  "timestamp": "2026-01-28T12:00:12.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-...",
  "data": {
    "tool_use_id": "toolu_abc123",
    "content": "File written successfully",
    "is_error": false
  }
}
```

**session_end:**
```json
{
  "event_type": "session_end",
  "timestamp": "2026-01-28T12:05:00.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-...",
  "data": {
    "file_context": {
      "read": ["src/main.ts"],
      "modified": ["src/auth.ts"],
      "created": ["src/new.ts"]
    }
  }
}
```

---

## 7. Remote Server Schema (PostgreSQL)

### Tables

**sessions:**
```sql
CREATE TABLE sessions (
  id UUID PRIMARY KEY,
  tool VARCHAR(50) NOT NULL, -- 'claude-code' or 'cursor'
  session_id VARCHAR(255) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  cwd TEXT NOT NULL,
  slug TEXT, -- Human-readable identifier
  uploaded_by VARCHAR(255), -- User who uploaded
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(tool, session_id)
);

CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX idx_sessions_tool ON sessions(tool);
CREATE INDEX idx_sessions_uploaded_by ON sessions(uploaded_by);
```

**messages:**
```sql
CREATE TABLE messages (
  id UUID PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  timestamp TIMESTAMPTZ NOT NULL,
  role VARCHAR(20) NOT NULL, -- 'user' or 'assistant'
  content JSONB NOT NULL, -- Array of content parts
  model VARCHAR(100),
  seq INT NOT NULL, -- Sequence number within session
  UNIQUE(session_id, seq)
);

CREATE INDEX idx_messages_session ON messages(session_id, seq);
```

**tools:**
```sql
CREATE TABLE tools (
  id UUID PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  message_id UUID REFERENCES messages(id) ON DELETE CASCADE,
  timestamp TIMESTAMPTZ NOT NULL,
  tool_use_id VARCHAR(255) NOT NULL,
  tool_name VARCHAR(100) NOT NULL,
  input JSONB NOT NULL,
  output JSONB,
  is_error BOOLEAN,
  UNIQUE(session_id, tool_use_id)
);

CREATE INDEX idx_tools_session ON tools(session_id);
```

**tags:**
```sql
CREATE TABLE tags (
  id UUID PRIMARY KEY,
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  tag_key VARCHAR(100) NOT NULL, -- 'team', 'repo', 'category', etc.
  tag_value VARCHAR(255) NOT NULL,
  UNIQUE(session_id, tag_key, tag_value)
);

CREATE INDEX idx_tags_session ON tags(session_id);
CREATE INDEX idx_tags_key_value ON tags(tag_key, tag_value);
```

---

## 8. Implementation Priorities

### Phase 1: Claude Code + Local Storage (v0.1)
- [x] Research completed
- [ ] SPEC: System architecture
- [ ] SPEC: Data format (JSONL schema)
- [ ] SPEC: API design (daemon HTTP API)
- [ ] Implement: `tabs-cli` (hook handler + daemon starter)
- [ ] Implement: `tabs-daemon` (event receiver + JSONL writer)
- [ ] Implement: Hook installation (`tabs-cli install`)
- [ ] Test: End-to-end capture with Claude Code

### Phase 2: Local Web UI (v0.2)
- [ ] SPEC: UI flows (timeline, detail, search)
- [ ] Implement: TanStack Start app
- [ ] Implement: JSONL reader + parser
- [ ] Implement: Timeline view with filters
- [ ] Implement: Session detail view
- [ ] Implement: Search functionality

### Phase 3: Remote Server (v0.3)
- [ ] SPEC: Server API design
- [ ] SPEC: PostgreSQL schema (refined)
- [ ] Implement: `tabs-server` (Go HTTP server)
- [ ] Implement: Session upload endpoint
- [ ] Implement: Search/browse endpoints
- [ ] Implement: Tag filtering
- [ ] Implement: Docker containerization
- [ ] Implement: Push flow in local UI

### Phase 4: Cursor Integration (v0.4)
- [ ] SPEC: Cursor adapter design
- [ ] Implement: Cursor hook handler
- [ ] Implement: SQLite poller
- [ ] Implement: JSON blob parser
- [ ] Implement: Cursor session → JSONL converter
- [ ] Test: End-to-end capture with Cursor

---

## 9. Open Questions

### Technical
1. **Daemon startup:** Should daemon persist across reboots (systemd/launchd), or start-on-demand only?
2. **Cursor polling frequency:** How often to query SQLite? Balance freshness vs resource usage.
3. **Session deduplication:** If same session captured twice (e.g., resume), merge or separate?
4. **Authentication:** How should remote server authenticate uploads? API keys, OAuth, mTLS?
5. **Index files:** Should we maintain index for fast lookups, or scan JSONL on-demand?

### UX
1. **Session review before push:** Should users see a preview/diff before pushing to server?
2. **Auto-tagging:** Should we auto-extract tags from cwd (e.g., git repo name)?
3. **Redaction:** How to handle sensitive data before sharing? Manual review, auto-detect patterns?
4. **Filters:** What search/filter dimensions matter most? (date, tool, cwd, tags, file paths?)

### Security
1. **Secret detection:** Use pattern matching (regex for API keys), or trust user review?
2. **Server access control:** IAP, OAuth, mTLS, or simple API key?
3. **Data retention:** Should server auto-delete sessions after X days?
4. **Encryption:** Encrypt sessions at rest on server? In transit via HTTPS only?

---

## 10. References

### Claude Code
- [Claude Code Hooks Documentation](https://code.claude.com/docs/en/hooks)
- [Session Management Guide](https://deepwiki.com/zebbern/claude-code-guide/9.1-session-management)
- [getspecstory - Claude Code Provider](https://github.com/specstoryai/getspecstory/tree/main/specstory-cli/pkg/providers/claudecode)

### Cursor
- [Cursor Hooks Documentation](https://cursor.com/docs/agent/hooks)
- [Deep Dive: Cursor Hooks](https://blog.gitbutler.com/cursor-hooks-deep-dive)
- [cursor-chat-export Tool](https://github.com/somogyijanos/cursor-chat-export)
- [cursor-view Tool](https://github.com/saharmor/cursor-view)
- [Cursor Enterprise Features](https://cursor.com/changelog/enterprise-dec-2025)

### Architecture References
- [getspecstory Repository](https://github.com/specstoryai/getspecstory)
- [getspecstory Provider SPI](https://github.com/specstoryai/getspecstory/blob/main/specstory-cli/docs/PROVIDER-SPI.md)

---

**Next Steps:** Proceed to architecture design phase (system architecture SPEC).
