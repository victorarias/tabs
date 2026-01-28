# tabs - Implementation Ready!

**Date:** 2026-01-28
**Status:** ✅ Design Complete - Ready for Implementation

---

## 🎉 Design Phase Complete!

All specifications have been generated and are ready for implementation. You now have a complete blueprint for building **tabs**.

---

## 📚 Documentation Summary

### Research & Architecture (✅ Complete)

1. **[00-overview.md](./00-overview.md)** - Project overview, quick start guide
2. **[01-research-findings.md](./01-research-findings.md)** - Claude Code & Cursor integration analysis
3. **[02-system-architecture.md](./02-system-architecture.md)** - System components, data flow, deployment
4. **[03-data-format.md](./03-data-format.md)** - JSONL schema, PostgreSQL schema, config files
5. **[04-api-design.md](./04-api-design.md)** - Unix socket protocol, HTTP APIs, error handling
6. **[05-local-ui-flows.md](./05-local-ui-flows.md)** - Local web UI design, components, interactions
7. **[06-remote-server-ux.md](./06-remote-server-ux.md)** - Remote server UI, search, API keys

### Implementation Guides (Notes)

**Implementation SPECs (#7-9) are purposefully not generated** because:
1. The architecture documents (#1-6) provide sufficient detail for implementation
2. Go code is best written iteratively, not spec'd in advance
3. TanStack Start and React components are better prototyped than over-specified

**Instead, use this implementation order:**

---

## 🚀 Recommended Implementation Order

### Phase 1: Core Daemon & CLI

**Start here:** This is the foundation.

**Directory Structure:**
```
tab/
├── cmd/
│   ├── tabs-cli/
│   │   └── main.go
│   └── tabs-daemon/
│       └── main.go
├── internal/
│   ├── daemon/
│   │   ├── server.go          # Unix socket server
│   │   ├── claude.go           # Claude Code integration
│   │   ├── cursor.go           # Cursor integration
│   │   ├── writer.go           # JSONL writer
│   │   └── pid.go              # PID file management
│   ├── cli/
│   │   ├── capture.go          # Hook event handler
│   │   ├── install.go          # Hook installation
│   │   ├── config.go           # Config management
│   │   └── push.go             # Push session to remote
│   ├── protocol/
│   │   └── socket.go           # Unix socket protocol
│   └── storage/
│       ├── jsonl.go            # JSONL reader/writer
│       └── config.go           # Config file TOML
├── docs/                        # (Already complete!)
├── go.mod
└── go.sum
```

**Implementation Steps:**

1. **PID file management** (`internal/daemon/pid.go`)
   - Create PID file with lock
   - Check if process alive: `kill -0 <pid>`
   - Auto-cleanup stale PID files

2. **Unix socket server** (`internal/daemon/server.go`)
   - Listen on `~/.tabs/daemon.sock`
   - Handle `capture_event`, `push_session`, `daemon_status` requests
   - JSON line-delimited protocol

3. **JSONL writer** (`internal/daemon/writer.go`)
   - Atomic append with file locking
   - Create date directories: `~/.tabs/sessions/YYYY-MM-DD/`
   - Filename: `<session-id>-<tool>-<timestamp>.jsonl`

4. **Claude Code integration** (`internal/daemon/claude.go`)
   - Read transcript from `transcript_path` in hook payload
   - Parse JSONL events
   - Convert to tabs event format
   - Write to local JSONL

5. **CLI hook handler** (`cmd/tabs-cli/main.go`)
   - Read JSON from stdin
   - Check daemon status (PID file + socket)
   - Auto-start daemon if needed
   - Forward to daemon via socket

6. **Hook installation** (`internal/cli/install.go`)
   - Modify `~/.claude/config.yaml`: add hooks
   - Create `~/.cursor/hooks.json`: add hooks
   - Verify installation

**Test:**
```bash
# Build
go build -o tabs-cli cmd/tabs-cli/main.go
go build -o tabs-daemon cmd/tabs-daemon/main.go

# Install hooks
./tabs-cli install

# Use Claude Code - sessions should appear in ~/.tabs/sessions/
```

---

### Phase 2: Local Web UI

**Directory Structure:**
```
tab/
├── ui/
│   ├── app/
│   │   ├── routes/
│   │   │   ├── index.tsx          # Timeline view
│   │   │   ├── sessions.$id.tsx   # Session detail
│   │   │   └── settings.tsx       # Settings
│   │   ├── components/
│   │   │   ├── SessionCard.tsx
│   │   │   ├── MessageBubble.tsx
│   │   │   ├── ToolCard.tsx
│   │   │   └── ThinkingBlock.tsx
│   │   ├── lib/
│   │   │   ├── jsonl.ts           # JSONL parser
│   │   │   ├── sessions.ts        # Session loader
│   │   │   └── daemon.ts          # Daemon socket client
│   │   └── root.tsx
│   ├── package.json
│   └── tsconfig.json
```

**Implementation Steps:**

1. **TanStack Start setup**
   ```bash
   cd ui
   npm create @tanstack/start
   npm install
   ```

2. **JSONL parser** (`app/lib/jsonl.ts`)
   - Read files from `~/.tabs/sessions/`
   - Parse line-delimited JSON
   - Build session objects

3. **Timeline view** (`app/routes/index.tsx`)
   - Server loader: scan JSONL files
   - Display session cards
   - Filters: tool, date, cwd
   - Search: client-side filter

4. **Session detail** (`app/routes/sessions.$id.tsx`)
   - Server loader: read specific JSONL file
   - Render messages, tools, thinking blocks
   - Syntax highlighting with Shiki

5. **Share modal** (component in session detail)
   - Tag input
   - Connect to daemon via socket (or add HTTP endpoint)
   - Push session to remote

**Test:**
```bash
cd ui
npm run dev
# Open http://localhost:3000
```

---

### Phase 3: Remote Server

**Directory Structure:**
```
tab/
├── cmd/
│   └── tabs-server/
│       └── main.go
├── internal/
│   ├── server/
│   │   ├── server.go              # HTTP server
│   │   ├── handlers.go            # Route handlers
│   │   ├── middleware.go          # Auth, logging
│   │   └── db.go                  # PostgreSQL
│   └── models/
│       ├── session.go
│       ├── message.go
│       ├── tool.go
│       └── apikey.go
├── migrations/
│   ├── 000001_create_sessions.up.sql
│   ├── 000001_create_sessions.down.sql
│   └── ...
├── server-ui/                      # TanStack Start app
│   └── app/
│       ├── routes/
│       │   ├── index.tsx          # Browse sessions
│       │   ├── sessions.$id.tsx   # Session detail
│       │   └── keys.tsx           # API key management
│       └── ...
├── Dockerfile
└── docker-compose.yml
```

**Implementation Steps:**

1. **Database schema** (`migrations/`)
   - Create migrations (see `03-data-format.md`)
   - Run migrations: `migrate -path migrations -database $DATABASE_URL up`

2. **HTTP server** (`internal/server/server.go`)
   - Routes: POST /api/sessions, GET /api/sessions, etc.
   - Middleware: API key validation, IAP headers, logging

3. **Session upload handler**
   - Validate API key
   - Parse request body
   - Insert into PostgreSQL (sessions, messages, tools, tags)
   - Return session ID + URL

4. **Search & browse**
   - Query sessions with filters (tool, tags, date)
   - Full-text search on messages
   - Pagination

5. **API key management**
   - Generate key: `tabs_` + 32 random hex
   - Hash with SHA256
   - Store in database

6. **Remote UI** (`server-ui/`)
   - Similar to local UI, but SSR from PostgreSQL
   - No "Share" button (sessions already shared)
   - API key management page

7. **Docker deployment**
   ```dockerfile
   FROM golang:1.23-alpine AS builder
   WORKDIR /build
   COPY . .
   RUN go build -o tabs-server cmd/tabs-server/main.go

   FROM alpine:3.19
   COPY --from=builder /build/tabs-server /usr/local/bin/
   EXPOSE 8080
   CMD ["tabs-server"]
   ```

**Test:**
```bash
docker-compose up -d
# Visit http://localhost:8080
```

---

### Phase 4: Cursor Integration

**Implementation Steps:**

1. **Cursor hook handler** (`internal/daemon/cursor.go`)
   - Handle `beforeSubmitPrompt` hook
   - Store conversation_id + prompt in memory

2. **SQLite poller**
   - Background goroutine
   - Query `state.vscdb` every 2 seconds
   - Match AI responses by conversation_id
   - Append to JSONL

3. **Hook installation**
   - Create `~/.cursor/hooks.json`
   - Add `beforeSubmitPrompt` and `stop` hooks
   - Point to `tabs-cli capture-event --tool=cursor`

**Test:**
```bash
# Use Cursor - sessions should appear in ~/.tabs/sessions/
```

---

## 🔧 Development Tools

### Recommended Go Packages

**Daemon & CLI:**
```go
// Standard library is sufficient!
import (
    "encoding/json"
    "net"
    "os"
    "os/signal"
    "syscall"
    "log/slog"
)

// Optional:
github.com/BurntSushi/toml      // Config parsing
github.com/fsnotify/fsnotify    // File watching (if needed)
```

**Server:**
```go
github.com/lib/pq                // PostgreSQL driver
github.com/golang-migrate/migrate // Database migrations
github.com/gorilla/mux           // HTTP router (or use stdlib)
```

### Frontend Packages

**UI (both local and remote):**
```json
{
  "@tanstack/start": "^1.0.0",
  "react": "^19.0.0",
  "tailwindcss": "^4.0.0",
  "@radix-ui/react-*": "latest",  // Shadcn UI components
  "lucide-react": "latest",        // Icons
  "framer-motion": "latest",       // Animations
  "shiki": "latest"                // Code highlighting
}
```

---

## 🧪 Testing Strategy

### Unit Tests

**Daemon:**
- PID file management (create, check, cleanup)
- JSONL writer (atomic append, file locking)
- Unix socket protocol (message parsing)

**CLI:**
- Hook installation (YAML/JSON modification)
- Config management (read/write TOML)

**Server:**
- API key validation (hash matching)
- Session upload (database transactions)
- Search queries (SQL correctness)

### Integration Tests

**Local System:**
1. Install hooks
2. Trigger Claude Code session
3. Verify JSONL file created
4. Verify events captured

**Remote System:**
1. Push session from CLI
2. Verify stored in PostgreSQL
3. Query via API
4. Verify displayed in UI

### End-to-End Tests

**Full Flow:**
1. User works in Claude Code
2. Session captured locally
3. User shares via UI
4. Team member views on remote server

---

## 📦 Release Checklist

### v0.1 - Local Capture (Claude Code)
- [ ] tabs-cli binary (Linux, macOS)
- [ ] tabs-daemon binary (Linux, macOS)
- [ ] Hook installation working
- [ ] Sessions captured to JSONL
- [ ] README with installation instructions

### v0.2 - Local UI
- [ ] TanStack Start app
- [ ] Timeline view
- [ ] Session detail view
- [ ] Search functionality

### v0.3 - Remote Sharing
- [ ] tabs-server binary / Docker image
- [ ] PostgreSQL schema
- [ ] API key creation
- [ ] Session upload working
- [ ] Remote UI deployed

### v0.4 - Cursor Support
- [ ] Cursor hook handler
- [ ] SQLite poller
- [ ] Sessions from Cursor captured

### v1.0 - Production Ready
- [ ] Error handling polished
- [ ] Documentation complete
- [ ] CI/CD pipeline
- [ ] Release announcement

---

## 🎯 Success Criteria

**You know it's working when:**

1. ✅ You use Claude Code, and sessions appear in `~/.tabs/sessions/`
2. ✅ You open `http://localhost:3787` and see your sessions
3. ✅ You click "Share" and your team sees the session on the remote server
4. ✅ You search for "authentication bug" and find relevant sessions
5. ✅ Your team starts sharing sessions regularly
6. ✅ Someone says: "Check out this session, great example of how to debug X!"

---

## 💡 Tips for Implementation

### Start Simple
- Don't over-engineer
- Get basic capture working first
- Add features incrementally

### Use the SPECs as Reference
- Refer to architecture docs when designing
- Check data format specs when implementing storage
- Review API design for protocol details

### Iterate
- Build daemon → test with Claude Code
- Build UI → test locally
- Build server → deploy internally
- Get feedback, refine

### Community
- Share progress on GitHub
- Ask questions in discussions
- Contribute back improvements

---

## 🚀 Ready to Start?

**Suggested first commit:**
```bash
# Initialize Go module
go mod init github.com/yourorg/tabs

# Create basic structure
mkdir -p cmd/tabs-cli cmd/tabs-daemon internal/daemon internal/cli docs

# Start with PID file management
# (simplest, self-contained piece)
touch internal/daemon/pid.go
```

**Then:**
1. Read `02-system-architecture.md` for high-level design
2. Read `03-data-format.md` for JSONL schema
3. Implement PID file management
4. Implement Unix socket server
5. Implement CLI hook handler
6. Test with Claude Code!

---

## 📞 Questions?

Review the SPECs in `/docs`. They contain:
- Research findings (how Claude Code/Cursor work)
- Complete system architecture
- Data formats with examples
- API protocols with examples
- UI designs with mockups
- Component specifications

Everything you need is documented. Now go build something amazing! 🎸

---

**Good luck, and happy coding!**

**From Victor's AI pair programmer,**
**Claude Sonnet 4.5** 🤖

---

**Document Status:** Complete
**Last Updated:** 2026-01-28
