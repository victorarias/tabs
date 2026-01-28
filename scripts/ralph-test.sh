#!/bin/bash
set -euo pipefail

# Test version of Ralph loop using test PRD
# This is a simplified version for testing the Ralph concept

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PRD_FILE="$PROJECT_DIR/prd-test.json"
PROGRESS_FILE="$PROJECT_DIR/progress-test.txt"
PROMPT_FILE="$PROJECT_DIR/CODEX-TEST.md"

cd "$PROJECT_DIR"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[Ralph Test]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[Ralph Test]${NC} $1"
}

log_error() {
    echo -e "${RED}[Ralph Test]${NC} $1"
}

# Handle Ctrl+C gracefully
cleanup() {
    echo ""
    log_error "Test interrupted by user"
    exit 130
}

trap cleanup SIGINT SIGTERM

log_info "Starting Ralph test with trivial task..."
log_info "PRD: $PRD_FILE"
log_info "Prompt: $PROMPT_FILE"
echo ""

# Get story status
story_passes=$(python3 -c "
import json
with open('$PRD_FILE') as f:
    prd = json.load(f)
print(prd['stories'][0].get('passes', False))
" 2>/dev/null)

if [[ "$story_passes" == "True" ]]; then
    log_success "Story already complete! Check internal/hello/hello.go"
    exit 0
fi

log_info "Story status: Not complete"
log_info "Calling Codex to implement hello package..."
echo ""

# Call Codex with prompt from file
if ! codex exec \
    --skip-git-repo-check \
    --full-auto \
    -C "$PROJECT_DIR" \
    "$(cat "$PROMPT_FILE")"; then
    log_error "Codex execution failed"
    exit 1
fi

# Run quality checks
log_info "Running quality checks..."

if ! make build 2>&1 | head -20; then
    log_error "Build failed!"
    exit 1
fi

if ! go vet ./...; then
    log_error "go vet failed!"
    exit 1
fi

log_success "Quality checks passed!"

# Check if story is now marked complete
story_passes=$(python3 -c "
import json
with open('$PRD_FILE') as f:
    prd = json.load(f)
print(prd['stories'][0].get('passes', False))
" 2>/dev/null)

if [[ "$story_passes" == "True" ]]; then
    log_success "Story marked complete by Codex ✓"
else
    log_error "Story NOT marked complete - check prd-test.json"
fi

# Show the result
if [[ -f "internal/hello/hello.go" ]]; then
    log_success "Created file: internal/hello/hello.go"
    echo ""
    cat internal/hello/hello.go
else
    log_error "File internal/hello/hello.go not found!"
fi

echo ""
log_info "Test complete! Check git log to see the commit."
