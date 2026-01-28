# API Design Specification: tabs

**Version:** 1.0
**Date:** 2026-01-28
**Status:** SPEC

---

## Overview

This document specifies all APIs in the tabs system:
1. Unix Socket Protocol (CLI ↔ daemon)
2. Local Web Server API (UI ↔ daemon/filesystem)
3. Remote Server HTTP API (client ↔ server)

**Design Principles:**
- **Simple protocols** - JSON everywhere, no custom binary formats
- **Idempotent** - Safe to retry operations
- **Stateless** - Each request contains all needed context
- **Versioned** - API version in requests for future compatibility

---

## 1. Unix Socket Protocol (CLI ↔ Daemon)

### Overview

**Transport:** Unix domain socket
**Location:** `~/.tabs/daemon.sock`
**Format:** Line-delimited JSON (JSON-LD)
**Connection:** One request per connection (connect, send, receive, close)

### Message Format

**Request:**
```json
{
  "version": "1.0",
  "type": "capture_event" | "push_session" | "daemon_status",
  "payload": {
    // Request-specific data
  }
}
```

**Response:**
```json
{
  "version": "1.0",
  "status": "ok" | "error",
  "data": {
    // Response data (if status is "ok")
  },
  "error": {
    // Error details (if status is "error")
    "code": "error_code",
    "message": "Human-readable error message"
  }
}
```

### Protocol Flow

```
1. CLI connects to ~/.tabs/daemon.sock
2. CLI sends JSON request + newline
3. Daemon processes request
4. Daemon sends JSON response + newline
5. CLI reads response
6. CLI closes connection
```

**Timing:**
- Total round-trip: <50ms (typically <10ms)
- Connection timeout: 2 seconds
- Read timeout: 5 seconds

---

### 1.1 capture_event (Hook Event)

**Purpose:** Forward hook event from Claude Code/Cursor to daemon

**Request:**
```json
{
  "version": "1.0",
  "type": "capture_event",
  "payload": {
    "tool": "claude-code" | "cursor",
    "timestamp": "2026-01-28T12:00:00.000Z",
    "event": {
      // Raw hook payload from stdin
    }
  }
}
```

**Response (Success):**
```json
{
  "version": "1.0",
  "status": "ok",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "events_written": 3
  }
}
```

**Response (Error):**
```json
{
  "version": "1.0",
  "status": "error",
  "error": {
    "code": "invalid_payload",
    "message": "Missing required field: session_id"
  }
}
```

**Error Codes:**
- `invalid_payload` - Malformed event data
- `write_failed` - Could not write to JSONL file
- `unknown_tool` - Tool not supported (not claude-code or cursor)

**Example (Claude Code SessionStart):**
```json
{
  "version": "1.0",
  "type": "capture_event",
  "payload": {
    "tool": "claude-code",
    "timestamp": "2026-01-28T12:00:00.123Z",
    "event": {
      "session_id": "550e8400-e29b-41d4-a716-446655440000",
      "transcript_path": "/home/user/.claude/projects/abc123/550e8400.jsonl",
      "cwd": "/home/user/projects/myapp",
      "permission_mode": "ask"
    }
  }
}
```

---

### 1.2 push_session (Share to Remote)

**Purpose:** Push a local session to remote server

**Request:**
```json
{
  "version": "1.0",
  "type": "push_session",
  "payload": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "tool": "claude-code",
    "tags": [
      {"key": "team", "value": "platform"},
      {"key": "repo", "value": "myapp"}
    ]
  }
}
```

**Response (Success):**
```json
{
  "version": "1.0",
  "status": "ok",
  "data": {
    "remote_id": "123e4567-e89b-12d3-a456-426614174000",
    "url": "https://tabs.company.com/sessions/123e4567-e89b-12d3-a456-426614174000"
  }
}
```

**Response (Error):**
```json
{
  "version": "1.0",
  "status": "error",
  "error": {
    "code": "session_not_found",
    "message": "Session 550e8400-e29b-41d4-a716-446655440000 not found locally"
  }
}
```

**Error Codes:**
- `session_not_found` - Session file doesn't exist locally
- `no_api_key` - API key not configured
- `invalid_api_key` - API key rejected by remote server
- `network_error` - Could not reach remote server
- `duplicate_session` - Session already uploaded to server

---

### 1.3 daemon_status (Daemon Info)

**Purpose:** Get daemon status and statistics

**Request:**
```json
{
  "version": "1.0",
  "type": "daemon_status",
  "payload": {}
}
```

**Response:**
```json
{
  "version": "1.0",
  "status": "ok",
  "data": {
    "pid": 12345,
    "uptime_seconds": 3600,
    "sessions_captured": 42,
    "events_processed": 1337,
    "cursor_polling": true,
    "last_event_at": "2026-01-28T12:05:00Z"
  }
}
```

**Error Codes:** (None, always succeeds if daemon is running)

---

## 2. Local Web Server API (TanStack Start)

### Overview

**Server:** TanStack Start (Node.js)
**Port:** 3787 (configurable via `~/.tabs/config.toml`)
**Base URL:** `http://localhost:3787`
**Access:** Localhost only (bind to `127.0.0.1`, not `0.0.0.0`)
**CSRF Protection:** Enforce Origin/Host checks for all mutating routes (POST/PUT/DELETE).
Only allow `http://localhost:<port>` and `http://127.0.0.1:<port>`. Reject all others.

**Architecture:**
- SSR (Server-Side Rendering) for initial page load
- React for client-side interactivity
- Direct filesystem access to read JSONL files
- Unix socket communication to daemon for push operations

---

### 2.1 Page Routes (SSR)

#### GET / (Homepage)

**Purpose:** Timeline view of all sessions

**Response:** HTML page with React app

**Server-side data loading:**
1. Scan `~/.tabs/sessions/` for all JSONL files
2. Parse session metadata (session_id, tool, created_at, cwd)
3. Sort by created_at DESC
4. Pass to React as initial props

**URL Params:**
- `?tool=claude-code` - Filter by tool
- `?date=2026-01-28` - Filter by date
- `?cwd=/home/user/projects/myapp` - Filter by cwd

---

#### GET /sessions/:id (Session Detail)

**Purpose:** View full session details

**URL Params:**
- `:id` - Session ID (UUID)

**Response:** HTML page with React app

**Server-side data loading:**
1. Find JSONL file matching session_id: `~/.tabs/sessions/**/*<session-id>-*.jsonl`
2. Read and parse all events
3. Build message thread with tools
4. Pass to React as initial props

**Error Handling:**
- Session not found: Return 404 page

---

#### GET /settings (Settings Page)

**Purpose:** Configure API key, server URL, preferences

**Response:** HTML page with React app

**Server-side data loading:**
1. Read `~/.tabs/config.toml`
2. Pass config (with API key masked) to React

---

### 2.2 API Routes (JSON)

#### POST /api/sessions/push

**Purpose:** Push a session to remote server (called from UI)

**Request Body:**
```json
{
  "session_id": "550e8400-e29b-41d4-a716-446655440000",
  "tool": "claude-code",
  "tags": [
    {"key": "team", "value": "platform"},
    {"key": "repo", "value": "myapp"}
  ]
}
```

**Response (Success):**
```json
{
  "status": "ok",
  "remote_id": "123e4567-e89b-12d3-a456-426614174000",
  "url": "https://tabs.company.com/sessions/123e4567-e89b-12d3-a456-426614174000"
}
```

**Response (Error):**
```json
{
  "status": "error",
  "error": {
    "code": "no_api_key",
    "message": "API key not configured. Please set it in Settings."
  }
}
```

**Implementation:**
1. Validate request body
2. Connect to daemon via Unix socket
3. Send `push_session` request to daemon
4. Return daemon's response

---

#### GET /api/sessions

**Purpose:** List all sessions (for search/filter)

**Query Params:**
- `?tool=claude-code` - Filter by tool
- `?date=2026-01-28` - Filter by date
- `?cwd=/home/user/projects` - Filter by cwd (prefix match)
- `?q=search term` - Full-text search in messages

**Response:**
```json
{
  "sessions": [
    {
      "session_id": "550e8400-e29b-41d4-a716-446655440000",
      "tool": "claude-code",
      "created_at": "2026-01-28T12:00:00Z",
      "ended_at": "2026-01-28T12:05:00Z",
      "cwd": "/home/user/projects/myapp",
      "duration_seconds": 300,
      "message_count": 12,
      "tool_use_count": 8,
      "file_path": "/home/user/.tabs/sessions/2026-01-28/550e8400-claude-code-1738065600.jsonl"
    }
  ],
  "total": 42
}
```

**Implementation:**
1. Scan `~/.tabs/sessions/` for matching JSONL files
2. Parse metadata from each file (first and last events)
3. Apply filters
4. Return results

---

#### GET /api/sessions/:id

**Purpose:** Get full session details (for detail view)

**URL Params:**
- `:id` - Session ID (UUID)

**Response:**
```json
{
  "session": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "tool": "claude-code",
    "created_at": "2026-01-28T12:00:00Z",
    "ended_at": "2026-01-28T12:05:00Z",
    "cwd": "/home/user/projects/myapp",
    "events": [
      {
        "event_type": "session_start",
        "timestamp": "2026-01-28T12:00:00.000Z",
        "data": {...}
      },
      {
        "event_type": "message",
        "timestamp": "2026-01-28T12:00:05.123Z",
        "data": {...}
      }
    ]
  }
}
```

**Error:**
```json
{
  "status": "error",
  "error": {
    "code": "session_not_found",
    "message": "Session not found"
  }
}
```

---

#### GET /api/config

**Purpose:** Get current configuration

**Response:**
```json
{
  "local": {
    "ui_port": 3787,
    "log_level": "info"
  },
  "remote": {
    "server_url": "https://tabs.company.com",
    "api_key_configured": true,
    "api_key_prefix": "tabs_abc1234"
  }
}
```

**Note:** Full API key never returned, only prefix for display

---

#### PUT /api/config

**Purpose:** Update configuration

**Request Body:**
```json
{
  "remote": {
    "server_url": "https://tabs.company.com",
    "api_key": "tabs_abc123def456..."
  }
}
```

**Response:**
```json
{
  "status": "ok",
  "message": "Configuration updated"
}
```

**Implementation:**
1. Validate config values
2. Write to `~/.tabs/config.toml`
3. Return success

---

#### GET /api/daemon/status

**Purpose:** Get daemon status (for health check)

**Response:**
```json
{
  "running": true,
  "pid": 12345,
  "uptime_seconds": 3600,
  "sessions_captured": 42,
  "events_processed": 1337
}
```

**Implementation:**
1. Connect to daemon via Unix socket
2. Send `daemon_status` request
3. Return result

**Error (Daemon not running):**
```json
{
  "running": false
}
```

---

## 3. Remote Server HTTP API

### Overview

**Server:** Go HTTP server (tabs-server)
**Port:** 8080 (containerized, behind load balancer)
**Base URL:** `https://tabs.company.com`
**Authentication:**
  - IAP (Identity-Aware Proxy) for web UI routes
  - API key (Bearer token) for `/api/sessions` POST

---

### 3.1 Public API (API Key Auth)

#### POST /api/sessions

**Purpose:** Upload a session from local client

**Authentication:** `Authorization: Bearer tabs_abc123def456...`

**Request Body:**
```json
{
  "session": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "tool": "claude-code",
    "created_at": "2026-01-28T12:00:00Z",
    "ended_at": "2026-01-28T12:05:00Z",
    "cwd": "/home/user/projects/myapp",
    "events": [
      {
        "event_type": "session_start",
        "timestamp": "2026-01-28T12:00:00.000Z",
        "tool": "claude-code",
        "session_id": "550e8400-e29b-41d4-a716-446655440000",
        "data": {...}
      },
      // ... all events
    ]
  },
  "tags": [
    {"key": "team", "value": "platform"},
    {"key": "repo", "value": "myapp"}
  ]
}
```

**Response (Success - 201 Created):**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "url": "https://tabs.company.com/sessions/123e4567-e89b-12d3-a456-426614174000"
}
```

**Response (Error - 401 Unauthorized):**
```json
{
  "error": {
    "code": "invalid_api_key",
    "message": "Invalid or expired API key"
  }
}
```

**Response (Error - 409 Conflict):**
```json
{
  "error": {
    "code": "duplicate_session",
    "message": "Session already uploaded"
  }
}
```

**Response (Error - 400 Bad Request):**
```json
{
  "error": {
    "code": "invalid_request",
    "message": "Missing required field: session.session_id"
  }
}
```

**Implementation:**
1. Validate API key (query `api_keys` table, check hash)
2. Derive uploader identity from API key: `uploaded_by = api_keys.user_id`, `api_key_id = api_keys.id`
3. Validate request body (required fields, valid UUIDs, timestamps)
4. Check for duplicate: `SELECT id FROM sessions WHERE tool = $1 AND session_id = $2`
5. Start transaction
6. Insert into `sessions` table
7. Batch insert into `messages` table
8. Batch insert into `tools` table
9. Batch insert into `tags` table
10. Commit transaction
11. Update API key `last_used_at` and `usage_count`
12. Return session ID and URL

---

### 3.2 Protected API (IAP Auth)

**IAP Headers:**
- `X-Forwarded-User: alice@company.com` (or custom header depending on IAP provider)
- Used to identify user for audit logging

---

#### GET /api/sessions

**Purpose:** List and search sessions

**Authentication:** IAP

**Query Params:**
- `?page=1` - Page number (default: 1)
- `?limit=20` - Items per page (default: 20, max: 100)
- `?tool=claude-code` - Filter by tool
- `?uploaded_by=alice@company.com` - Filter by uploader
- `?tag=team:platform` - Filter by tag (can repeat for multiple tags)
- `?q=search term` - Full-text search in session messages
- `?sort=created_at` - Sort field (default: created_at)
- `?order=desc` - Sort order (asc or desc, default: desc)

**Response:**
```json
{
  "sessions": [
    {
      "id": "123e4567-e89b-12d3-a456-426614174000",
      "tool": "claude-code",
      "session_id": "550e8400-e29b-41d4-a716-446655440000",
      "created_at": "2026-01-28T12:00:00Z",
      "ended_at": "2026-01-28T12:05:00Z",
      "cwd": "/home/user/projects/myapp",
      "uploaded_by": "alice@company.com",
      "uploaded_at": "2026-01-28T13:00:00Z",
      "duration_seconds": 300,
      "message_count": 12,
      "tool_use_count": 8,
      "tags": [
        {"key": "team", "value": "platform"},
        {"key": "repo", "value": "myapp"}
      ]
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 142,
    "total_pages": 8
  }
}
```

**SQL Query (simplified):**
```sql
SELECT
  s.id, s.tool, s.session_id, s.created_at, s.ended_at, s.cwd,
  s.uploaded_by, s.uploaded_at, s.duration_seconds, s.message_count, s.tool_use_count,
  json_agg(json_build_object('key', t.tag_key, 'value', t.tag_value)) as tags
FROM sessions s
LEFT JOIN tags t ON s.id = t.session_id
WHERE 1=1
  AND ($tool IS NULL OR s.tool = $tool)
  AND ($uploaded_by IS NULL OR s.uploaded_by = $uploaded_by)
  AND ($tag IS NULL OR s.id IN (
    SELECT session_id FROM tags WHERE tag_key = $tag_key AND tag_value = $tag_value
  ))
GROUP BY s.id
ORDER BY s.created_at DESC
LIMIT $limit OFFSET $offset;
```

---

#### GET /api/sessions/:id

**Purpose:** Get full session details

**Authentication:** IAP

**URL Params:**
- `:id` - Session ID (remote UUID, not session_id)

**Response:**
```json
{
  "session": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "tool": "claude-code",
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "created_at": "2026-01-28T12:00:00Z",
    "ended_at": "2026-01-28T12:05:00Z",
    "cwd": "/home/user/projects/myapp",
    "uploaded_by": "alice@company.com",
    "uploaded_at": "2026-01-28T13:00:00Z",
    "duration_seconds": 300,
    "message_count": 12,
    "tool_use_count": 8,
    "tags": [
      {"key": "team", "value": "platform"},
      {"key": "repo", "value": "myapp"}
    ],
    "messages": [
      {
        "id": "abc12345-e89b-12d3-a456-426614174222",
        "timestamp": "2026-01-28T12:00:05Z",
        "seq": 1,
        "role": "user",
        "content": [
          {"type": "text", "text": "Implement a prime checking function"}
        ]
      },
      {
        "id": "def67890-e89b-12d3-a456-426614174333",
        "timestamp": "2026-01-28T12:00:10Z",
        "seq": 2,
        "role": "assistant",
        "model": "claude-sonnet-4-5-20250929",
        "content": [
          {"type": "thinking", "text": "I need to implement..."},
          {"type": "text", "text": "I'll create a function..."}
        ]
      }
    ],
    "tools": [
      {
        "id": "ghi12345-e89b-12d3-a456-426614174444",
        "timestamp": "2026-01-28T12:00:15Z",
        "tool_use_id": "toolu_01ABC123XYZ",
        "tool_name": "write",
        "input": {
          "file_path": "/home/user/projects/myapp/src/prime.ts",
          "content": "export function isPrime..."
        },
        "output": {
          "content": "File written successfully"
        },
        "is_error": false
      }
    ]
  }
}
```

**Error (404 Not Found):**
```json
{
  "error": {
    "code": "session_not_found",
    "message": "Session not found"
  }
}
```

**SQL Queries:**
```sql
-- Get session
SELECT * FROM sessions WHERE id = $1;

-- Get messages
SELECT * FROM messages WHERE session_id = $1 ORDER BY seq;

-- Get tools
SELECT * FROM tools WHERE session_id = $1 ORDER BY timestamp;

-- Get tags
SELECT tag_key, tag_value FROM tags WHERE session_id = $1;
```

---

#### GET /api/tags

**Purpose:** List all unique tags with counts

**Authentication:** IAP

**Query Params:**
- `?key=team` - Filter by tag key
- `?limit=100` - Max tags to return (default: 100)

**Response:**
```json
{
  "tags": [
    {
      "key": "team",
      "value": "platform",
      "count": 42
    },
    {
      "key": "team",
      "value": "frontend",
      "count": 38
    },
    {
      "key": "repo",
      "value": "myapp",
      "count": 67
    }
  ]
}
```

**SQL Query:**
```sql
SELECT tag_key, tag_value, COUNT(*) as count
FROM tags
WHERE ($key IS NULL OR tag_key = $key)
GROUP BY tag_key, tag_value
ORDER BY count DESC
LIMIT $limit;
```

---

#### POST /api/keys

**Purpose:** Create new API key

**Authentication:** IAP

**Request Body:**
```json
{
  "name": "My Laptop"
}
```

**Response (Success - 201 Created):**
```json
{
  "id": "789e0123-e89b-12d3-a456-426614174111",
  "key": "tabs_abc123def456...",
  "key_prefix": "tabs_abc1234",
  "name": "My Laptop",
  "created_at": "2026-01-28T13:00:00Z"
}
```

**Note:** `key` (full API key) is only returned once. User must save it.

**Implementation:**
1. Read user ID from IAP header: `X-Forwarded-User`
2. Generate API key: `tabs_` + 32 random hex chars
3. Hash key: SHA256
4. Extract prefix: first 13 chars (tabs_ + 8 hex)
5. Insert into `api_keys` table
6. Return full key (only this once)

---

#### GET /api/keys

**Purpose:** List user's API keys

**Authentication:** IAP

**Response:**
```json
{
  "keys": [
    {
      "id": "789e0123-e89b-12d3-a456-426614174111",
      "key_prefix": "tabs_abc1234",
      "name": "My Laptop",
      "created_at": "2026-01-28T13:00:00Z",
      "last_used_at": "2026-01-28T14:30:00Z",
      "is_active": true,
      "usage_count": 42
    }
  ]
}
```

**SQL Query:**
```sql
SELECT id, key_prefix, name, created_at, last_used_at, is_active, usage_count
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;
```

---

#### DELETE /api/keys/:id

**Purpose:** Revoke (deactivate) an API key

**Authentication:** IAP

**URL Params:**
- `:id` - API key ID (UUID)

**Response:**
```json
{
  "status": "ok",
  "message": "API key revoked"
}
```

**Implementation:**
```sql
UPDATE api_keys
SET is_active = false
WHERE id = $1 AND user_id = $2;
```

**Note:** Don't delete key, just mark inactive (for audit trail)

---

### 3.3 Web UI Routes (IAP Auth)

All web UI routes serve the TanStack Start React app.

**Routes:**
- `GET /` - Homepage (session timeline)
- `GET /sessions/:id` - Session detail view
- `GET /search` - Search page
- `GET /keys` - API key management

**SSR Data Loading:**
Each route fetches data from PostgreSQL and passes to React as initial props.

---

## 4. Error Handling

### Error Response Format (All APIs)

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error message",
    "details": {
      // Optional additional context
    }
  }
}
```

### HTTP Status Codes

**Success:**
- `200 OK` - Successful GET/PUT/DELETE
- `201 Created` - Successful POST (resource created)

**Client Errors:**
- `400 Bad Request` - Invalid request body or params
- `401 Unauthorized` - Missing or invalid API key
- `403 Forbidden` - IAP authentication failed
- `404 Not Found` - Resource doesn't exist
- `409 Conflict` - Duplicate resource (e.g., session already uploaded)
- `429 Too Many Requests` - Rate limit exceeded

**Server Errors:**
- `500 Internal Server Error` - Unexpected server error
- `503 Service Unavailable` - Database connection failed

### Common Error Codes

**Unix Socket Protocol:**
- `invalid_payload` - Malformed request
- `session_not_found` - Session file doesn't exist
- `write_failed` - JSONL write failed
- `no_api_key` - API key not configured
- `network_error` - Remote server unreachable

**Remote Server:**
- `invalid_api_key` - API key invalid or revoked
- `duplicate_session` - Session already exists
- `invalid_request` - Missing required fields
- `session_not_found` - Session doesn't exist
- `rate_limit_exceeded` - Too many requests

---

## 5. Rate Limiting

### Remote Server

**API Key Endpoints (POST /api/sessions):**
- Limit: 100 requests per hour per API key
- Algorithm: Sliding window
- Response when exceeded: 429 Too Many Requests

**IAP-Protected Endpoints:**
- Limit: 1000 requests per hour per user
- Algorithm: Sliding window

**Implementation:**
```go
// Store in Redis or in-memory map
type RateLimiter struct {
    requests map[string][]time.Time
    limit    int
    window   time.Duration
}

func (rl *RateLimiter) Allow(key string) bool {
    now := time.Now()
    cutoff := now.Add(-rl.window)

    // Remove old requests
    rl.requests[key] = filter(rl.requests[key], func(t time.Time) bool {
        return t.After(cutoff)
    })

    // Check limit
    if len(rl.requests[key]) >= rl.limit {
        return false
    }

    // Add new request
    rl.requests[key] = append(rl.requests[key], now)
    return true
}
```

---

## 6. API Versioning

### Current Version: 1.0

**Version in Requests:**
- Unix socket: `"version": "1.0"` in JSON
- HTTP: `Accept: application/vnd.tabs.v1+json` header (optional, defaults to v1)

**Breaking Changes:**
If API changes incompatibly:
1. Increment version: `1.0` → `2.0`
2. Support both versions for 6 months
3. Deprecate old version with warnings
4. Remove old version after deprecation period

**Example (v2 with breaking changes):**
```json
{
  "version": "2.0",
  "type": "capture_event",
  "payload": {
    // New structure
  }
}
```

Daemon checks version and routes to appropriate handler.

---

## 7. API Examples

### Example 1: Full Capture Flow

**Step 1: Hook fires, CLI forwards to daemon**
```bash
# Hook executes:
echo '{"session_id": "550e8400-...", ...}' | tabs-cli capture-event --tool=claude-code
```

**Step 2: CLI connects to daemon**
```json
// Sent to ~/.tabs/daemon.sock
{
  "version": "1.0",
  "type": "capture_event",
  "payload": {
    "tool": "claude-code",
    "timestamp": "2026-01-28T12:00:00Z",
    "event": {
      "session_id": "550e8400-e29b-41d4-a716-446655440000",
      "transcript_path": "/home/user/.claude/projects/abc123/550e8400.jsonl",
      "cwd": "/home/user/projects/myapp"
    }
  }
}
```

**Step 3: Daemon responds**
```json
{
  "version": "1.0",
  "status": "ok",
  "data": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "events_written": 1
  }
}
```

---

### Example 2: Push Session from UI

**Step 1: User clicks "Share" in local UI**

**Step 2: UI sends to local server**
```javascript
fetch('http://localhost:3787/api/sessions/push', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    session_id: '550e8400-e29b-41d4-a716-446655440000',
    tool: 'claude-code',
    tags: [
      {key: 'team', value: 'platform'},
      {key: 'repo', value: 'myapp'}
    ]
  })
})
```

**Step 3: Local server forwards to daemon via Unix socket**
```json
{
  "version": "1.0",
  "type": "push_session",
  "payload": {
    "session_id": "550e8400-e29b-41d4-a716-446655440000",
    "tool": "claude-code",
    "tags": [...]
  }
}
```

**Step 4: Daemon reads session JSONL, posts to remote**
```bash
curl -X POST https://tabs.company.com/api/sessions \
  -H "Authorization: Bearer tabs_abc123..." \
  -H "Content-Type: application/json" \
  -d '{
    "session": {
      "session_id": "550e8400-...",
      "tool": "claude-code",
      "events": [...]
    },
    "tags": [...]
  }'
```

**Step 5: Remote server responds**
```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "url": "https://tabs.company.com/sessions/123e4567-..."
}
```

**Step 6: Daemon returns to local server**
```json
{
  "version": "1.0",
  "status": "ok",
  "data": {
    "remote_id": "123e4567-...",
    "url": "https://tabs.company.com/sessions/123e4567-..."
  }
}
```

**Step 7: Local server returns to UI**
```json
{
  "status": "ok",
  "remote_id": "123e4567-...",
  "url": "https://tabs.company.com/sessions/123e4567-..."
}
```

**Step 8: UI shows success message and copies URL to clipboard**

---

## 8. Security Considerations

### Unix Socket
- **Permissions:** 0600 (owner read/write only)
- **Location:** User home directory (not /tmp, to avoid cross-user attacks)
- **Validation:** Daemon validates all requests (type checking, bounds checking)

### API Keys
- **Storage:** SHA256 hash in database, never plain text
- **Transmission:** HTTPS only (TLS 1.3+)
- **Display:** Only show prefix in UI (tabs_abc1234...)
- **Rotation:** User can revoke and create new keys

### Remote Server
- **HTTPS Only:** Enforce TLS 1.3+
- **IAP:** All web UI routes behind Identity-Aware Proxy
- **Rate Limiting:** Prevent abuse
- **Input Validation:** Sanitize all inputs, parameterized SQL queries
- **CORS:** Restrict to known origins (if needed)

---

## Conclusion

This API design provides:
- ✅ **Simple protocols** - JSON everywhere, easy to debug
- ✅ **Fast communication** - Unix sockets for local, HTTPS for remote
- ✅ **Secure** - File permissions, API keys, IAP, rate limiting
- ✅ **Idempotent** - Safe to retry uploads (duplicate detection)
- ✅ **Versioned** - Future-proof with version negotiation
- ✅ **Well-defined errors** - Structured error responses with codes

**Next Steps:**
1. Generate Local UI Flows SPEC
2. Generate Remote Server UX SPEC
3. Generate implementation SPECs (daemon, CLI, server)

---

**Document Status:** Ready for review
**Last Updated:** 2026-01-28
