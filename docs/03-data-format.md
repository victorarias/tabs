# Data Format Specification: tabs

**Version:** 1.0
**Date:** 2026-01-28
**Status:** SPEC

---

## Overview

This document specifies all data formats used in the tabs system:
1. Local JSONL format (session capture)
2. Directory structure and file naming
3. Configuration file format
4. PID file format
5. Remote PostgreSQL schema

**Design Principles:**
- **Human-readable** - All formats can be inspected with standard tools
- **Append-only** - JSONL supports streaming writes without corruption
- **Tool-agnostic** - Schema accommodates Claude Code, Cursor, and future tools
- **Timestamped** - All events have ISO 8601 timestamps for ordering

---

## 1. Local Storage Structure

### Directory Layout

```
~/.tabs/
├── daemon.pid                           # Daemon process ID
├── daemon.lock                          # Single-instance lock file
├── daemon.sock                          # Unix domain socket
├── daemon.log                           # Daemon log file
├── config.toml                          # User configuration
├── state/                               # Per-session cursor state
└── sessions/                            # Captured sessions
    ├── 2026-01-28/                      # Date-based folders
    │   ├── 550e8400-claude-code-1738065600.jsonl
    │   ├── 668320d2-cursor-1738067400.jsonl
    │   └── ...
    ├── 2026-01-29/
    │   └── ...
    └── ...
```

### File Naming Convention

**Session Files:**
```
<session-id>-<tool>-<timestamp>.jsonl

Components:
  session-id:  UUID from Claude Code/Cursor (36 chars)
  tool:        "claude-code" or "cursor"
  timestamp:   Unix timestamp (seconds since epoch)

Examples:
  550e8400-e29b-41d4-a716-446655440000-claude-code-1738065600.jsonl
  668320d2-2fd8-4888-b33c-2a466fec86e7-cursor-1738067400.jsonl
```

**Why this format:**
- `session-id` - Unique identifier from source tool
- `tool` - Allows filtering by tool without opening file
- `timestamp` - Prevents collisions if same session captured twice (resume scenario)
- Sortable by name (timestamp is numeric)

### Directory Permissions

```
~/.tabs/                 0700 (drwx------)
~/.tabs/daemon.pid       0600 (-rw-------)
~/.tabs/daemon.lock      0600 (-rw-------)
~/.tabs/daemon.sock      0600 (srw-------)
~/.tabs/daemon.log       0600 (-rw-------)
~/.tabs/config.toml      0600 (-rw-------)
~/.tabs/state/           0700 (drwx------)
~/.tabs/sessions/        0700 (drwx------)
~/.tabs/sessions/*/*.jsonl  0600 (-rw-------)
```

**Rationale:** Sessions may contain sensitive information (code, prompts, file paths). Only owner should read/write.

### Session Cursor State (Deduplication)

To prevent duplicate events when re-reading transcripts, the daemon stores a per-session cursor.

**Location:**
```
~/.tabs/state/<session-id>.json
```

**Schema:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "transcript_path": "/home/user/.claude/projects/abc123/550e8400.jsonl",
  "last_offset": 123456,
  "last_line_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "updated_at": "2026-01-28T12:00:16.012Z"
}
```

**Behavior:**
- On each hook event, read the transcript from `last_offset` forward
- Skip lines whose hash matches `last_line_hash` (defensive against partial writes)
- Append only new events to the session JSONL
- Update cursor after a successful append batch

---

## 2. JSONL Event Format

### Structure

Each line in a session JSONL file is a single JSON object:

```json
{"event_type": "...", "timestamp": "...", "tool": "...", "session_id": "...", "data": {...}}
```

**Common Fields (All Events):**
- `event_type` (string, required) - Type of event (see below)
- `timestamp` (string, required) - ISO 8601 timestamp with milliseconds
- `tool` (string, required) - "claude-code" or "cursor"
- `session_id` (string, required) - UUID from source tool
- `data` (object, required) - Event-specific payload

### Event Types

**Event Type Summary:**
1. `session_start` - Session begins
2. `message` - User or assistant message
3. `tool_use` - Agent invokes a tool
4. `tool_result` - Tool execution result
5. `session_end` - Session completes

---

### 2.1 session_start

**When:** Session begins (SessionStart hook in Claude Code, beforeSubmitPrompt in Cursor)

**Schema:**
```json
{
  "event_type": "session_start",
  "timestamp": "2026-01-28T12:00:00.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "cwd": "/home/user/projects/myapp",
    "permission_mode": "ask",
    "model": "claude-sonnet-4-5-20250929",
    "metadata": {
      // Optional tool-specific metadata
    }
  }
}
```

**Fields:**
- `data.cwd` (string, required) - Working directory
- `data.permission_mode` (string, optional) - "ask", "auto", etc. (Claude Code only)
- `data.model` (string, optional) - Model identifier
- `data.metadata` (object, optional) - Tool-specific metadata

**Example (Claude Code):**
```json
{
  "event_type": "session_start",
  "timestamp": "2026-01-28T12:00:00.123Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "cwd": "/home/user/projects/myapp",
    "permission_mode": "ask",
    "model": "claude-sonnet-4-5-20250929"
  }
}
```

**Example (Cursor):**
```json
{
  "event_type": "session_start",
  "timestamp": "2026-01-28T12:00:00.456Z",
  "tool": "cursor",
  "session_id": "668320d2-2fd8-4888-b33c-2a466fec86e7",
  "data": {
    "cwd": "/Users/user/projects/webapp",
    "workspace_roots": ["/Users/user/projects/webapp"]
  }
}
```

---

### 2.2 message

**When:** User submits prompt, or assistant responds

**Schema:**
```json
{
  "event_type": "message",
  "timestamp": "2026-01-28T12:00:05.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "role": "user" | "assistant",
    "content": [
      {
        "type": "text" | "thinking",
        "text": "Content here..."
      }
    ],
    "model": "claude-sonnet-4-5-20250929"
  }
}
```

**Fields:**
- `data.role` (string, required) - "user" or "assistant"
- `data.content` (array, required) - Array of content parts
  - `type` (string, required) - "text" or "thinking"
  - `text` (string, required) - Content text
- `data.model` (string, optional) - Model used (assistant messages only)

**Example (User Message):**
```json
{
  "event_type": "message",
  "timestamp": "2026-01-28T12:00:05.123Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "role": "user",
    "content": [
      {
        "type": "text",
        "text": "Please implement a function to check if a number is prime"
      }
    ]
  }
}
```

**Example (Assistant Message with Thinking):**
```json
{
  "event_type": "message",
  "timestamp": "2026-01-28T12:00:10.456Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "role": "assistant",
    "content": [
      {
        "type": "thinking",
        "text": "I need to implement a prime checking function. The efficient approach is to check divisibility up to sqrt(n)."
      },
      {
        "type": "text",
        "text": "I'll create a function that checks if a number is prime using trial division up to the square root."
      }
    ],
    "model": "claude-sonnet-4-5-20250929"
  }
}
```

---

### 2.3 tool_use

**When:** Assistant invokes a tool (write, read, bash, etc.)

**Schema:**
```json
{
  "event_type": "tool_use",
  "timestamp": "2026-01-28T12:00:15.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_abc123",
    "tool_name": "write",
    "input": {
      // Tool-specific input parameters
    }
  }
}
```

**Fields:**
- `data.tool_use_id` (string, required) - Unique ID for this tool invocation
- `data.tool_name` (string, required) - Name of tool (write, read, bash, grep, etc.)
- `data.input` (object, required) - Tool-specific parameters

**Example (Write Tool):**
```json
{
  "event_type": "tool_use",
  "timestamp": "2026-01-28T12:00:15.789Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_01ABC123XYZ",
    "tool_name": "write",
    "input": {
      "file_path": "/home/user/projects/myapp/src/prime.ts",
      "content": "export function isPrime(n: number): boolean {\n  if (n <= 1) return false;\n  if (n <= 3) return true;\n  if (n % 2 === 0 || n % 3 === 0) return false;\n  \n  for (let i = 5; i * i <= n; i += 6) {\n    if (n % i === 0 || n % (i + 2) === 0) return false;\n  }\n  return true;\n}\n"
    }
  }
}
```

**Example (Bash Tool):**
```json
{
  "event_type": "tool_use",
  "timestamp": "2026-01-28T12:00:20.123Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_02DEF456ABC",
    "tool_name": "bash",
    "input": {
      "command": "npm test -- prime.test.ts",
      "description": "Run tests for prime function"
    }
  }
}
```

**Example (Read Tool):**
```json
{
  "event_type": "tool_use",
  "timestamp": "2026-01-28T12:00:25.456Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_03GHI789DEF",
    "tool_name": "read",
    "input": {
      "file_path": "/home/user/projects/myapp/src/prime.ts"
    }
  }
}
```

---

### 2.4 tool_result

**When:** Tool execution completes (success or error)

**Schema:**
```json
{
  "event_type": "tool_result",
  "timestamp": "2026-01-28T12:00:16.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_abc123",
    "content": "Result text or error message",
    "is_error": false
  }
}
```

**Fields:**
- `data.tool_use_id` (string, required) - Matches `tool_use_id` from tool_use event
- `data.content` (string, required) - Result text (stdout, file content, error message)
- `data.is_error` (boolean, required) - true if tool execution failed

**Example (Success):**
```json
{
  "event_type": "tool_result",
  "timestamp": "2026-01-28T12:00:16.012Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_01ABC123XYZ",
    "content": "File written successfully",
    "is_error": false
  }
}
```

**Example (Error):**
```json
{
  "event_type": "tool_result",
  "timestamp": "2026-01-28T12:00:21.234Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "tool_use_id": "toolu_02DEF456ABC",
    "content": "Error: Command failed with exit code 1\nnpm ERR! Test failed. See above for more details.",
    "is_error": true
  }
}
```

---

### 2.5 session_end

**When:** Session completes (user exits tool, SessionEnd hook)

**Schema:**
```json
{
  "event_type": "session_end",
  "timestamp": "2026-01-28T12:05:00.000Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "file_context": {
      "read": ["src/main.ts", "src/auth.ts"],
      "modified": ["src/prime.ts"],
      "created": ["test/prime.test.ts"]
    },
    "duration_seconds": 300,
    "message_count": 12,
    "tool_use_count": 8
  }
}
```

**Fields:**
- `data.file_context` (object, optional) - Files accessed during session
  - `read` (array of strings) - Files read
  - `modified` (array of strings) - Files modified
  - `created` (array of strings) - Files created
- `data.duration_seconds` (number, optional) - Session duration
- `data.message_count` (number, optional) - Total messages
- `data.tool_use_count` (number, optional) - Total tool invocations

**Example:**
```json
{
  "event_type": "session_end",
  "timestamp": "2026-01-28T12:05:00.789Z",
  "tool": "claude-code",
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "data": {
    "file_context": {
      "read": ["src/main.ts", "src/utils.ts", "package.json"],
      "modified": ["src/prime.ts", "test/prime.test.ts"],
      "created": []
    },
    "duration_seconds": 300,
    "message_count": 12,
    "tool_use_count": 8
  }
}
```

---

## 3. Configuration File Format

### File Location
`~/.tabs/config.toml`

### Schema

```toml
# tabs configuration file
# Generated by: tabs-cli config

[local]
# Local UI server port
ui_port = 3787

# Daemon log level (debug, info, warn, error)
log_level = "info"

[remote]
# Remote server URL (where sessions are pushed)
server_url = "https://tabs.company.com"

# API key for authentication (keep secret!)
api_key = "tabs_abc123def456..."

# Auto-push sessions (default: false, user must explicitly share)
auto_push = false

# Tags to apply to all pushed sessions
default_tags = ["team:platform", "user:alice"]

[cursor]
# Cursor SQLite database path (auto-detected, can override)
db_path = "~/Library/Application Support/Cursor/user/globalStorage/state.vscdb"

# Polling interval in seconds (default: 2)
poll_interval = 2

[claude_code]
# Claude Code project directory (auto-detected, can override)
projects_dir = "~/.claude/projects"
```

### Example

```toml
[local]
ui_port = 3787
log_level = "info"

[remote]
server_url = "https://tabs.company.com"
api_key = "tabs_1234567890abcdef1234567890abcdef"
auto_push = false
default_tags = ["team:platform", "user:alice"]

[cursor]
db_path = "/Users/alice/Library/Application Support/Cursor/user/globalStorage/state.vscdb"
poll_interval = 2

[claude_code]
projects_dir = "/home/alice/.claude/projects"
```

### Validation Rules
- `local.ui_port` - 1024-65535
- `local.log_level` - One of: debug, info, warn, error
- `remote.server_url` - Valid HTTPS URL
- `remote.api_key` - Starts with "tabs_", 36+ chars
- `cursor.poll_interval` - 1-60 seconds
- All paths - Valid filesystem paths, expand `~` to home directory

---

## 4. PID File Format

### File Location
`~/.tabs/daemon.pid`

### Format
Single line containing decimal process ID:
```
12345
```

### Usage

**Write (daemon startup):**
```go
pid := os.Getpid()
err := os.WriteFile("~/.tabs/daemon.pid", []byte(fmt.Sprintf("%d\n", pid)), 0600)
```

**Read (CLI checking if daemon is running):**
```go
data, err := os.ReadFile("~/.tabs/daemon.pid")
if err != nil {
    // PID file doesn't exist, daemon not running
    return false
}

pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
if err != nil {
    // Invalid PID file, remove and start new daemon
    os.Remove("~/.tabs/daemon.pid")
    return false
}

// Check if process is alive
process, err := os.FindProcess(pid)
if err != nil {
    // Process not found, remove stale PID file
    os.Remove("~/.tabs/daemon.pid")
    return false
}

// Send signal 0 (doesn't kill, just checks existence)
err = process.Signal(syscall.Signal(0))
if err != nil {
    // Process dead, remove stale PID file
    os.Remove("~/.tabs/daemon.pid")
    return false
}

// Process is alive
return true
```

---

## 5. Remote PostgreSQL Schema

### Database Name
`tabs`

### Tables

#### 5.1 sessions

**Purpose:** Store session metadata

```sql
CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Source tool information
  tool VARCHAR(50) NOT NULL CHECK (tool IN ('claude-code', 'cursor')),
  session_id VARCHAR(255) NOT NULL,

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL,
  ended_at TIMESTAMPTZ,
  uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

  -- Session context
  cwd TEXT NOT NULL,
  slug TEXT,  -- Human-readable identifier (e.g., "fix-auth-bug")

  -- Ownership and access
  uploaded_by VARCHAR(255) NOT NULL,  -- User who uploaded (IAP user for UI actions; API key owner for uploads)
  api_key_id UUID NOT NULL,  -- API key used for upload

  -- Stats
  duration_seconds INTEGER,
  message_count INTEGER DEFAULT 0,
  tool_use_count INTEGER DEFAULT 0,

  -- Indexes
  UNIQUE(tool, session_id)
);

CREATE INDEX idx_sessions_created_at ON sessions(created_at DESC);
CREATE INDEX idx_sessions_tool ON sessions(tool);
CREATE INDEX idx_sessions_uploaded_by ON sessions(uploaded_by);
CREATE INDEX idx_sessions_uploaded_at ON sessions(uploaded_at DESC);
CREATE INDEX idx_sessions_cwd ON sessions USING gin(to_tsvector('english', cwd));
```

**Upload Mapping:**
- For `POST /api/sessions` uploads, set `uploaded_by` from `api_keys.user_id` (derived from the API key).
- Ignore any client-provided `uploaded_by` field.

**Example Row:**
```sql
INSERT INTO sessions (id, tool, session_id, created_at, ended_at, cwd, uploaded_by, api_key_id, duration_seconds, message_count, tool_use_count)
VALUES (
  '123e4567-e89b-12d3-a456-426614174000',
  'claude-code',
  '550e8400-e29b-41d4-a716-446655440000',
  '2026-01-28T12:00:00Z',
  '2026-01-28T12:05:00Z',
  '/home/user/projects/myapp',
  'alice@company.com',
  '789e0123-e89b-12d3-a456-426614174111',
  300,
  12,
  8
);
```

---

#### 5.2 messages

**Purpose:** Store individual messages in session

```sql
CREATE TABLE messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Session reference
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,

  -- Message metadata
  timestamp TIMESTAMPTZ NOT NULL,
  seq INTEGER NOT NULL,  -- Sequence number within session (for ordering)
  role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant')),
  model VARCHAR(100),

  -- Content
  content JSONB NOT NULL,  -- Array of {type, text} objects

  -- Indexes
  UNIQUE(session_id, seq)
);

CREATE INDEX idx_messages_session ON messages(session_id, seq);
CREATE INDEX idx_messages_role ON messages(role);
CREATE INDEX idx_messages_content ON messages USING gin(content);
```

**Example Row:**
```sql
INSERT INTO messages (id, session_id, timestamp, seq, role, model, content)
VALUES (
  'abc12345-e89b-12d3-a456-426614174222',
  '123e4567-e89b-12d3-a456-426614174000',
  '2026-01-28T12:00:05Z',
  1,
  'user',
  NULL,
  '[{"type": "text", "text": "Implement a prime checking function"}]'::jsonb
);

INSERT INTO messages (id, session_id, timestamp, seq, role, model, content)
VALUES (
  'def67890-e89b-12d3-a456-426614174333',
  '123e4567-e89b-12d3-a456-426614174000',
  '2026-01-28T12:00:10Z',
  2,
  'assistant',
  'claude-sonnet-4-5-20250929',
  '[
    {"type": "thinking", "text": "I need to implement a prime checking function..."},
    {"type": "text", "text": "I will create a function..."}
  ]'::jsonb
);
```

---

#### 5.3 tools

**Purpose:** Store tool invocations and results

```sql
CREATE TABLE tools (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Session and message references
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  message_id UUID REFERENCES messages(id) ON DELETE SET NULL,

  -- Tool metadata
  timestamp TIMESTAMPTZ NOT NULL,
  tool_use_id VARCHAR(255) NOT NULL,
  tool_name VARCHAR(100) NOT NULL,

  -- Input and output
  input JSONB NOT NULL,
  output JSONB,
  is_error BOOLEAN DEFAULT FALSE,

  -- Indexes
  UNIQUE(session_id, tool_use_id)
);

CREATE INDEX idx_tools_session ON tools(session_id);
CREATE INDEX idx_tools_name ON tools(tool_name);
CREATE INDEX idx_tools_input_file_path ON tools((input->>'file_path'));
```

**Example Row (Write Tool):**
```sql
INSERT INTO tools (id, session_id, message_id, timestamp, tool_use_id, tool_name, input, output, is_error)
VALUES (
  'ghi12345-e89b-12d3-a456-426614174444',
  '123e4567-e89b-12d3-a456-426614174000',
  'def67890-e89b-12d3-a456-426614174333',
  '2026-01-28T12:00:15Z',
  'toolu_01ABC123XYZ',
  'write',
  '{"file_path": "/home/user/projects/myapp/src/prime.ts", "content": "export function isPrime..."}'::jsonb,
  '{"content": "File written successfully"}'::jsonb,
  false
);
```

**Example Row (Bash Tool with Error):**
```sql
INSERT INTO tools (id, session_id, message_id, timestamp, tool_use_id, tool_name, input, output, is_error)
VALUES (
  'jkl67890-e89b-12d3-a456-426614174555',
  '123e4567-e89b-12d3-a456-426614174000',
  'def67890-e89b-12d3-a456-426614174333',
  '2026-01-28T12:00:20Z',
  'toolu_02DEF456ABC',
  'bash',
  '{"command": "npm test", "description": "Run tests"}'::jsonb,
  '{"content": "Error: Command failed with exit code 1\\nnpm ERR! Test failed"}'::jsonb,
  true
);
```

---

#### 5.4 tags

**Purpose:** Store session tags for filtering and discovery

```sql
CREATE TABLE tags (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Session reference
  session_id UUID NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,

  -- Tag key-value pair
  tag_key VARCHAR(100) NOT NULL,
  tag_value VARCHAR(255) NOT NULL,

  -- Indexes
  UNIQUE(session_id, tag_key, tag_value)
);

CREATE INDEX idx_tags_session ON tags(session_id);
CREATE INDEX idx_tags_key_value ON tags(tag_key, tag_value);
CREATE INDEX idx_tags_key ON tags(tag_key);
```

**Example Rows:**
```sql
INSERT INTO tags (session_id, tag_key, tag_value)
VALUES
  ('123e4567-e89b-12d3-a456-426614174000', 'team', 'platform'),
  ('123e4567-e89b-12d3-a456-426614174000', 'repo', 'myapp'),
  ('123e4567-e89b-12d3-a456-426614174000', 'category', 'bugfix');
```

**Common Tag Keys:**
- `team` - Team name (e.g., "platform", "frontend", "infra")
- `repo` - Repository name (e.g., "myapp", "api-server")
- `category` - Session type (e.g., "bugfix", "feature", "refactor")
- `user` - User identifier (e.g., "alice", "bob")
- `pr` - Pull request number (e.g., "123")
- `issue` - Issue number (e.g., "456")

---

#### 5.5 api_keys

**Purpose:** Store API keys for authentication

```sql
CREATE TABLE api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Key information
  key_hash VARCHAR(64) NOT NULL UNIQUE,  -- SHA256 hash of key
  key_prefix VARCHAR(20) NOT NULL,  -- First 8 chars for display (tabs_abc1234...)

  -- Ownership
  user_id VARCHAR(255) NOT NULL,  -- User who created key (from IAP headers)
  name VARCHAR(255) NOT NULL,  -- User-provided name (e.g., "My Laptop")

  -- Timestamps
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  last_used_at TIMESTAMPTZ,
  expires_at TIMESTAMPTZ,  -- NULL = never expires

  -- Status
  is_active BOOLEAN NOT NULL DEFAULT TRUE,

  -- Usage stats
  usage_count BIGINT DEFAULT 0
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX idx_api_keys_created_at ON api_keys(created_at DESC);
```

**Example Row:**
```sql
INSERT INTO api_keys (id, key_hash, key_prefix, user_id, name, created_at, is_active)
VALUES (
  '789e0123-e89b-12d3-a456-426614174111',
  'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855',  -- SHA256 of actual key
  'tabs_abc1234',
  'alice@company.com',
  'My Laptop',
  '2026-01-28T10:00:00Z',
  true
);
```

**Key Generation:**
```go
// Generate new API key
func GenerateAPIKey() string {
    randomBytes := make([]byte, 32)
    rand.Read(randomBytes)
    return "tabs_" + hex.EncodeToString(randomBytes)
}

// Hash key for storage
func HashAPIKey(key string) string {
    hash := sha256.Sum256([]byte(key))
    return hex.EncodeToString(hash[:])
}

// Extract prefix for display
func GetKeyPrefix(key string) string {
    if len(key) < 13 {
        return key
    }
    return key[:13]  // "tabs_" + first 8 hex chars
}
```

---

### Database Initialization

**Create Database:**
```sql
CREATE DATABASE tabs
  WITH OWNER = tabs
  ENCODING = 'UTF8'
  LC_COLLATE = 'en_US.UTF-8'
  LC_CTYPE = 'en_US.UTF-8'
  TEMPLATE = template0;
```

**Create User:**
```sql
CREATE USER tabs WITH PASSWORD 'secure-password';
GRANT ALL PRIVILEGES ON DATABASE tabs TO tabs;
```

**Run Migrations:**
```bash
# Using golang-migrate or similar
migrate -path ./migrations -database "postgresql://tabs:password@localhost:5432/tabs?sslmode=disable" up
```

**Migration Files:**
```
migrations/
├── 000001_create_sessions.up.sql
├── 000001_create_sessions.down.sql
├── 000002_create_messages.up.sql
├── 000002_create_messages.down.sql
├── 000003_create_tools.up.sql
├── 000003_create_tools.down.sql
├── 000004_create_tags.up.sql
├── 000004_create_tags.down.sql
├── 000005_create_api_keys.up.sql
└── 000005_create_api_keys.down.sql
```

---

## 6. Example Session (Full JSONL)

**File:** `~/.tabs/sessions/2026-01-28/550e8400-claude-code-1738065600.jsonl`

```jsonl
{"event_type":"session_start","timestamp":"2026-01-28T12:00:00.000Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"cwd":"/home/user/projects/myapp","permission_mode":"ask","model":"claude-sonnet-4-5-20250929"}}
{"event_type":"message","timestamp":"2026-01-28T12:00:05.123Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"role":"user","content":[{"type":"text","text":"Please implement a function to check if a number is prime"}]}}
{"event_type":"message","timestamp":"2026-01-28T12:00:10.456Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"role":"assistant","content":[{"type":"thinking","text":"I need to implement a prime checking function. The efficient approach is to check divisibility up to sqrt(n)."},{"type":"text","text":"I'll create a function that checks if a number is prime using trial division up to the square root."}],"model":"claude-sonnet-4-5-20250929"}}
{"event_type":"tool_use","timestamp":"2026-01-28T12:00:15.789Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"tool_use_id":"toolu_01ABC123XYZ","tool_name":"write","input":{"file_path":"/home/user/projects/myapp/src/prime.ts","content":"export function isPrime(n: number): boolean {\n  if (n <= 1) return false;\n  if (n <= 3) return true;\n  if (n % 2 === 0 || n % 3 === 0) return false;\n  \n  for (let i = 5; i * i <= n; i += 6) {\n    if (n % i === 0 || n % (i + 2) === 0) return false;\n  }\n  return true;\n}\n"}}}
{"event_type":"tool_result","timestamp":"2026-01-28T12:00:16.012Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"tool_use_id":"toolu_01ABC123XYZ","content":"File written successfully","is_error":false}}
{"event_type":"message","timestamp":"2026-01-28T12:00:18.345Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"role":"assistant","content":[{"type":"text","text":"I've created the isPrime function in src/prime.ts. Let me run the tests to verify it works correctly."}],"model":"claude-sonnet-4-5-20250929"}}
{"event_type":"tool_use","timestamp":"2026-01-28T12:00:20.123Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"tool_use_id":"toolu_02DEF456ABC","tool_name":"bash","input":{"command":"npm test -- prime.test.ts","description":"Run tests for prime function"}}}
{"event_type":"tool_result","timestamp":"2026-01-28T12:00:21.234Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"tool_use_id":"toolu_02DEF456ABC","content":"PASS  src/prime.test.ts\n  isPrime\n    ✓ returns false for numbers <= 1 (3 ms)\n    ✓ returns true for 2 and 3 (1 ms)\n    ✓ returns false for even numbers (1 ms)\n    ✓ returns true for prime numbers (2 ms)\n    ✓ returns false for composite numbers (1 ms)\n\nTest Suites: 1 passed, 1 total\nTests:       5 passed, 5 total","is_error":false}}
{"event_type":"message","timestamp":"2026-01-28T12:00:23.567Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"role":"assistant","content":[{"type":"text","text":"Perfect! All tests passed. The isPrime function is working correctly."}],"model":"claude-sonnet-4-5-20250929"}}
{"event_type":"session_end","timestamp":"2026-01-28T12:05:00.789Z","tool":"claude-code","session_id":"550e8400-e29b-41d4-a716-446655440000","data":{"file_context":{"read":["package.json"],"modified":["src/prime.ts"],"created":["test/prime.test.ts"]},"duration_seconds":300,"message_count":5,"tool_use_count":2}}
```

---

## 7. Data Validation

### JSONL Validation Rules

**All Events:**
- Must be valid JSON (single line, no embedded newlines)
- Must have all required fields: `event_type`, `timestamp`, `tool`, `session_id`, `data`
- `timestamp` must be valid ISO 8601 with timezone
- `tool` must be "claude-code" or "cursor"
- `session_id` must be valid UUID format

**Event-Specific:**
- `message`: `role` must be "user" or "assistant"
- `tool_use`: `tool_use_id` must be unique within session
- `tool_result`: `tool_use_id` must match a prior `tool_use` event
- `session_end`: must be last event in file

**Parser Behavior:**
- Skip invalid JSON lines (log warning)
- Skip events with missing required fields (log warning)
- Continue processing valid events
- Don't crash on malformed data

### PostgreSQL Constraints

**Type Constraints:**
- All UUIDs validated by PostgreSQL
- Timestamps validated by PostgreSQL
- Check constraints on enums (tool, role)

**Foreign Key Constraints:**
- Messages reference sessions (cascade delete)
- Tools reference sessions (cascade delete)
- Tags reference sessions (cascade delete)
- Tools reference messages (set null on delete)

**Unique Constraints:**
- `(tool, session_id)` in sessions - prevent duplicate uploads
- `(session_id, seq)` in messages - enforce ordering
- `(session_id, tool_use_id)` in tools - prevent duplicate tool records
- `(session_id, tag_key, tag_value)` in tags - prevent duplicate tags

---

## 8. Data Retention

### Local (User Machine)

**No automatic cleanup:**
- User owns all data in `~/.tabs/sessions/`
- User can manually delete old sessions
- Future: `tabs-cli cleanup --older-than 90d` command

**Disk Space Estimates:**
- Small session (5 messages, 2 tools): ~10 KB
- Medium session (20 messages, 10 tools): ~50 KB
- Large session (100 messages, 50 tools): ~500 KB
- Daily usage (10 sessions): ~500 KB/day = ~180 MB/year

### Remote (Server)

**Retention Policy (Configurable):**
- Default: keep all sessions forever
- Optional: auto-archive sessions >1 year old to cold storage
- Optional: delete sessions >2 years old (compliance)

**Archive Strategy:**
- Move old sessions to S3/GCS
- Keep metadata in PostgreSQL (for search)
- Lazy-load session content on view

---

## 9. Migration and Compatibility

### Schema Version

**JSONL Schema Version:**
- Current: `1.0`
- Stored in first line of each JSONL file (optional):
```json
{"event_type":"schema_version","version":"1.0"}
```

**PostgreSQL Schema Version:**
- Managed by migrations (golang-migrate, Flyway, etc.)
- Current: `000005` (5 tables created)

### Breaking Changes

**If JSONL schema changes:**
1. Increment schema version: `1.0` → `2.0`
2. Parser supports both versions
3. Legacy events converted on read

**If PostgreSQL schema changes:**
1. Write migration: `000006_add_column.up.sql`
2. Write rollback: `000006_add_column.down.sql`
3. Deploy migration before app update

---

## Conclusion

This data format specification provides:
- ✅ **Human-readable** - JSONL, TOML, plain text PID files
- ✅ **Append-only** - JSONL supports streaming writes
- ✅ **Tool-agnostic** - Schema works for Claude Code, Cursor, future tools
- ✅ **Structured** - PostgreSQL schema for efficient querying
- ✅ **Validated** - Constraints and checks prevent bad data
- ✅ **Extensible** - Schema versioning for future changes

**Next Steps:**
1. Generate API Design SPEC (Unix socket protocol, HTTP APIs)
2. Generate Local UI Flows SPEC
3. Generate Remote Server UX SPEC

---

**Document Status:** Ready for review
**Last Updated:** 2026-01-28
