#!/bin/bash
set -euo pipefail

# Ralph loop with Claude Code supervising Codex
# Claude Code acts as HITL, reviewing Codex's work

MAX_ITERATIONS="${1:-50}"
PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PRD_FILE="$PROJECT_DIR/prd.json"
PROGRESS_FILE="$PROJECT_DIR/progress.txt"
CLAUDE_PROMPT="$PROJECT_DIR/CLAUDE-RALPH.md"

cd "$PROJECT_DIR"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[Ralph Supervised]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[Ralph Supervised]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[Ralph Supervised]${NC} $1"
}

log_error() {
    echo -e "${RED}[Ralph Supervised]${NC} $1"
}

# Handle Ctrl+C gracefully
cleanup() {
    echo ""
    log_warn "Ralph loop interrupted by user"
    log_info "Progress saved to prd.json and progress.txt"
    exit 130
}

trap cleanup SIGINT SIGTERM

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

# Main Ralph loop
log_info "Starting Ralph Supervised loop (max $MAX_ITERATIONS iterations)"
log_info "PRD: $PRD_FILE"
log_info "Progress: $PROGRESS_FILE"
log_info "Claude Code will supervise Codex on each iteration"
echo ""

for iteration in $(seq 1 "$MAX_ITERATIONS"); do
    log_info "=== Iteration $iteration/$MAX_ITERATIONS ==="

    # Get next story
    next_story=$(get_next_story) || {
        log_success "All stories complete! 🎉"
        exit 0
    }

    log_info "Next story: $next_story"

    # Call Claude Code to supervise this iteration
    log_info "Calling Claude Code to supervise Codex..."

    if ! (cd "$PROJECT_DIR" && claude \
        --print \
        --output-format stream-json \
        --verbose \
        --permission-mode bypassPermissions \
        "$(cat "$CLAUDE_PROMPT")"); then
        log_error "Claude Code supervision failed"
        log_warn "Check the output above for errors"
        log_warn "You may need to manually fix issues and resume Ralph"
        exit 1
    fi

    log_success "Iteration $iteration complete"
    echo ""

    sleep 2  # Brief pause between iterations
done

log_warn "Reached max iterations ($MAX_ITERATIONS)"
log_info "Check progress.txt and prd.json to see status"
