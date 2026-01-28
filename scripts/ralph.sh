#!/bin/bash
set -euo pipefail

# Ralph loop for tabs implementation using Codex
# Based on: https://www.aihero.dev/getting-started-with-ralph

MAX_ITERATIONS="${1:-50}"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PRD_FILE="$PROJECT_DIR/prd.json"
PROGRESS_FILE="$PROJECT_DIR/progress.txt"
PROMPT_FILE="$PROJECT_DIR/CODEX.md"

cd "$PROJECT_DIR"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Handle Ctrl+C gracefully
cleanup() {
    echo ""
    log_warn "Ralph loop interrupted by user"
    log_info "Progress saved to prd.json and progress.txt"
    exit 130
}

trap cleanup SIGINT SIGTERM

log_info() {
    echo -e "${BLUE}[Ralph]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[Ralph]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[Ralph]${NC} $1"
}

log_error() {
    echo -e "${RED}[Ralph]${NC} $1"
}

# Initialize files if they don't exist
if [[ ! -f "$PRD_FILE" ]]; then
    log_info "Creating initial PRD from IMPLEMENTATION-READY.md..."
    cat > "$PRD_FILE" << 'EOF'
{
  "branchName": "ralph/tabs-implementation",
  "stories": [
    {
      "id": "phase1-cli",
      "title": "Implement tabs-cli commands",
      "description": "Implement core CLI commands: capture, status, config. Parse flags and communicate with daemon via unix socket.",
      "priority": 1,
      "passes": false,
      "acceptance": [
        "tabs-cli capture --session-id <id> --event <json> sends event to daemon",
        "tabs-cli status shows daemon status",
        "tabs-cli config --set key=value persists configuration"
      ]
    },
    {
      "id": "phase1-daemon-core",
      "title": "Implement daemon lifecycle and IPC",
      "description": "Implement daemon auto-start, PID file management with daemon.lock, unix socket server, and graceful shutdown.",
      "priority": 2,
      "passes": false,
      "acceptance": [
        "Daemon auto-starts when CLI calls it",
        "PID file with atomic lock prevents multiple instances",
        "Unix socket accepts JSON-LD messages from CLI",
        "Daemon shuts down cleanly on SIGTERM"
      ]
    },
    {
      "id": "phase1-daemon-storage",
      "title": "Implement JSONL storage and deduplication",
      "description": "Implement event writing to JSONL files with daily folders, per-session cursor state for deduplication, and metadata extraction.",
      "priority": 3,
      "passes": false,
      "acceptance": [
        "Events written to ~/.tabs/sessions/YYYY-MM-DD/<session-id>-<tool>-<timestamp>.jsonl",
        "Cursor state prevents duplicate events",
        "Session metadata extracted and stored"
      ]
    },
    {
      "id": "phase1-claude-integration",
      "title": "Claude Code hook integration",
      "description": "Implement hook script that calls tabs-cli to capture events from Claude Code transcript.",
      "priority": 4,
      "passes": false,
      "acceptance": [
        "Hook script installs to ~/.claude/hooks/",
        "Events captured from transcript on project:start and user:prompt-submit",
        "No duplicate events written"
      ]
    },
    {
      "id": "phase2-local-webserver",
      "title": "Implement local web server",
      "description": "Build HTTP server that reads JSONL files and serves JSON API with CSRF protection.",
      "priority": 5,
      "passes": false,
      "acceptance": [
        "Server binds to 127.0.0.1 only",
        "Origin/Host checks prevent CSRF",
        "API returns sessions, messages, tools from JSONL files"
      ]
    },
    {
      "id": "phase2-local-ui",
      "title": "Implement local UI with TanStack Start",
      "description": "Build SSR React UI for browsing sessions with timeline view, session detail, and search.",
      "priority": 6,
      "passes": false,
      "acceptance": [
        "Timeline shows sessions grouped by day",
        "Session detail shows messages and tool interactions",
        "Search works across sessions"
      ]
    },
    {
      "id": "phase3-remote-server",
      "title": "Implement remote server with PostgreSQL",
      "description": "Build tabs-server that accepts uploads via API key auth and stores in PostgreSQL.",
      "priority": 7,
      "passes": false,
      "acceptance": [
        "Server accepts POST /api/sessions with API key",
        "Sessions stored in PostgreSQL with normalization",
        "uploaded_by derived from api_keys.user_id"
      ]
    },
    {
      "id": "phase3-remote-ui",
      "title": "Implement remote UI with tagging",
      "description": "Build remote UI that displays all uploaded sessions with tag filtering and search.",
      "priority": 8,
      "passes": false,
      "acceptance": [
        "UI shows all sessions from all users",
        "Tag-based filtering works",
        "Search spans all sessions"
      ]
    }
  ]
}
EOF
    log_success "Created prd.json"
fi

if [[ ! -f "$PROGRESS_FILE" ]]; then
    echo "# Tabs Implementation Progress" > "$PROGRESS_FILE"
    echo "" >> "$PROGRESS_FILE"
    echo "This file tracks learnings and progress across Ralph iterations." >> "$PROGRESS_FILE"
    echo "" >> "$PROGRESS_FILE"
    log_success "Created progress.txt"
fi

if [[ ! -f "$PROMPT_FILE" ]]; then
    log_info "Creating CODEX.md prompt file..."
    cat > "$PROMPT_FILE" << 'EOF'
# Tabs Implementation - Ralph Loop

You are implementing the tabs project according to the specifications in docs/.

## Your Task

1. Read `prd.json` to see all implementation stories
2. Read `progress.txt` to see what has been learned so far
3. Find the next story where `passes: false` (lowest priority number)
4. Implement that story following the specs in docs/
5. Run quality checks:
   - `make build` - must succeed
   - `make test` - must pass (once tests exist)
   - `go vet ./...` - must pass
6. If checks pass:
   - Commit your changes with a clear commit message
   - Update `prd.json`: set `passes: true` for the completed story
   - Append learnings to `progress.txt`: document any gotchas, patterns, or decisions
7. If checks fail:
   - Fix the issues
   - Do not mark story as complete
   - Document the failure in progress.txt

## Guidelines

- Follow the specs in docs/ exactly
- Keep it simple - no over-engineering
- Test as you go
- Document non-obvious decisions in progress.txt
- Each iteration should complete one story from the PRD

## Current Iteration

Check prd.json for the next incomplete story and implement it.
EOF
    log_success "Created CODEX.md"
fi

# Get next incomplete story
get_next_story() {
    python3 -c "
import json
import sys

with open('$PRD_FILE') as f:
    prd = json.load(f)

incomplete = [s for s in prd['stories'] if not s.get('passes', False)]
if not incomplete:
    sys.exit(1)

# Sort by priority
incomplete.sort(key=lambda s: s.get('priority', 999))
story = incomplete[0]

print(f\"{story['id']}: {story['title']}\")
" 2>/dev/null
}

# Update PRD to mark story as complete
mark_story_complete() {
    local story_id="$1"
    python3 -c "
import json

with open('$PRD_FILE', 'r') as f:
    prd = json.load(f)

for story in prd['stories']:
    if story['id'] == '$story_id':
        story['passes'] = True
        break

with open('$PRD_FILE', 'w') as f:
    json.dump(prd, f, indent=2)
"
}

# Main Ralph loop
log_info "Starting Ralph loop (max $MAX_ITERATIONS iterations)"
log_info "PRD: $PRD_FILE"
log_info "Progress: $PROGRESS_FILE"
log_info "Prompt: $PROMPT_FILE"
echo ""

for iteration in $(seq 1 "$MAX_ITERATIONS"); do
    log_info "=== Iteration $iteration/$MAX_ITERATIONS ==="

    # Get next story
    next_story=$(get_next_story) || {
        log_success "All stories complete! 🎉"
        exit 0
    }

    log_info "Next story: $next_story"

    # Call Codex with prompt from file
    log_info "Calling Codex..."
    if ! codex exec \
        --skip-git-repo-check \
        --full-auto \
        -C "$PROJECT_DIR" \
        "$(cat "$PROMPT_FILE")"; then
        log_error "Codex execution failed"
        log_warn "Check the output above for errors"
        log_warn "You may need to manually fix issues and resume Ralph"
        exit 1
    fi

    # Run quality checks
    log_info "Running quality checks..."

    if ! make build; then
        log_error "Build failed!"
        log_warn "Codex should have fixed this. Check the code."
        exit 1
    fi

    if ! go vet ./...; then
        log_error "go vet failed!"
        log_warn "Codex should have fixed this. Check the code."
        exit 1
    fi

    # Run tests if they exist
    if ls *_test.go internal/**/*_test.go 2>/dev/null | grep -q .; then
        if ! make test; then
            log_error "Tests failed!"
            log_warn "Codex should have fixed this. Check the code."
            exit 1
        fi
    fi

    log_success "Quality checks passed"

    # Check if Codex already committed
    if ! git diff --quiet; then
        log_warn "Uncommitted changes found - Codex should have committed"
        log_info "Committing on behalf of Codex..."
        git add -A
        git commit -m "Ralph iteration $iteration: $next_story

Co-Authored-By: Ralph <ralph@aihero.dev>"
    fi

    log_success "Iteration $iteration complete"
    echo ""

    sleep 2  # Brief pause between iterations
done

log_warn "Reached max iterations ($MAX_ITERATIONS)"
log_info "Check progress.txt and prd.json to see status"
