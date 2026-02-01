# Tabs - Claude Code Session Capture

## Project Overview

Tabs captures Claude Code sessions via hooks, stores them locally, and provides a UI for viewing conversation history with tool calls and file diffs.

## Architecture

- `tabs-daemon`: Background process that receives events via Unix socket
- `tabs-cli`: CLI for capturing events, pushing sessions, and serving UI
- Hooks: Claude Code hooks (SessionStart, UserPromptSubmit, Stop) call `tabs-cli capture-event`

## Development Commands

```bash
make build          # Build all binaries
make install        # Install to ~/.local/bin
make test           # Run unit tests
```

## E2E Testing (IMPORTANT)

**After any changes to event capture or parsing, run the E2E test:**

```bash
./scripts/e2e-test.sh
```

This test:
- Drives Claude Code through a real multi-turn conversation
- Verifies all event types are captured (messages, tool_use, tool_result)
- Checks the UI API if running

See `docs/E2E-TESTING.md` for details.

## Key Files

- `internal/daemon/claude.go` - Claude Code transcript parsing
- `internal/daemon/server.go` - Daemon socket server and event handling
- `cmd/tabs-cli/main.go` - CLI commands including `capture-event`

## Common Tasks

### Restart daemon after code changes

```bash
make install && pkill -f tabs-daemon && tabs-daemon &
```

### Check captured sessions

```bash
ls -t ~/.tabs/sessions/$(date +%Y-%m-%d)/*.jsonl | head -5
jq . ~/.tabs/sessions/$(date +%Y-%m-%d)/SESSION_ID*.jsonl
```

### Debug hook issues

```bash
cat ~/.tabs/daemon.log
cat ~/.tabs/state/SESSION_ID.json  # Check cursor state
```
