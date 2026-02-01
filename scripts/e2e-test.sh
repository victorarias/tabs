#!/usr/bin/env bash
set -euo pipefail

# E2E Test for tabs - drives Claude Code and verifies capture + UI rendering
#
# This script:
# 1. Runs a multi-turn Claude session with tool calls
# 2. Captures all stream-json events
# 3. Waits for tabs daemon to process
# 4. Verifies events were captured correctly
# 5. Verifies UI API returns the data

TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

echo "=== Tabs E2E Test ==="
echo "Temp dir: $TEMP_DIR"

# Ensure daemon is running
if ! pgrep -f tabs-daemon > /dev/null; then
    echo "Starting tabs-daemon..."
    tabs-daemon &
    sleep 2
fi

echo ""
echo "--- Step 1: Run multi-turn Claude session ---"

CLAUDE_OUTPUT="$TEMP_DIR/claude-output.jsonl"
cd "$TEMP_DIR"

# Generate a session ID to use across turns
SESSION_UUID=$(uuidgen | tr '[:upper:]' '[:lower:]')
echo "Using session ID: $SESSION_UUID"

COMMON_ARGS="--output-format stream-json --verbose --dangerously-skip-permissions --model sonnet"

# Turn 1: Write file
echo ""
echo "=== Turn 1: Write file ==="
claude -p "Write a new file called config.json with this exact content: {\"name\": \"test-app\", \"version\": \"1.0.0\", \"debug\": false}" \
    --session-id "$SESSION_UUID" \
    $COMMON_ARGS \
    2>&1 | tee -a "$CLAUDE_OUTPUT"

# Turn 2: Edit version
echo ""
echo "=== Turn 2: Edit version ==="
claude -p "Edit config.json to change version to 1.1.0" \
    --resume "$SESSION_UUID" \
    $COMMON_ARGS \
    2>&1 | tee -a "$CLAUDE_OUTPUT"

# Turn 3: Edit debug flag
echo ""
echo "=== Turn 3: Edit debug ==="
claude -p "Edit config.json to change debug to true" \
    --resume "$SESSION_UUID" \
    $COMMON_ARGS \
    2>&1 | tee -a "$CLAUDE_OUTPUT"

# Turn 4: Bash cat
echo ""
echo "=== Turn 4: Bash command ==="
claude -p "Run: cat config.json" \
    --resume "$SESSION_UUID" \
    $COMMON_ARGS \
    2>&1 | tee -a "$CLAUDE_OUTPUT"

# Turn 5: Done
echo ""
echo "=== Turn 5: Final ==="
claude -p "Say exactly: MULTI_TURN_COMPLETE" \
    --resume "$SESSION_UUID" \
    $COMMON_ARGS \
    2>&1 | tee -a "$CLAUDE_OUTPUT"

echo ""
echo "--- Step 2: Session ID ---"
SESSION_ID="$SESSION_UUID"  # Use same variable name for consistency in rest of script
echo "Session ID: $SESSION_ID"

# Give daemon time to process
echo "Waiting for daemon to process..."
sleep 2

echo ""
echo "--- Step 3: Verify tabs captured the session ---"

# Find the session file
SESSION_FILE=$(ls -t ~/.tabs/sessions/*/[${SESSION_ID:0:8}]*.jsonl 2>/dev/null | head -1 || true)
if [[ -z "$SESSION_FILE" ]]; then
    # Try broader search
    SESSION_FILE=$(find ~/.tabs/sessions -name "${SESSION_ID}*.jsonl" 2>/dev/null | head -1 || true)
fi

if [[ -z "$SESSION_FILE" ]]; then
    echo "ERROR: Session file not found for $SESSION_ID"
    echo "Available session files:"
    ls -la ~/.tabs/sessions/*/ 2>/dev/null | tail -10
    exit 1
fi
echo "Session file: $SESSION_FILE"

# Count captured events
EVENT_COUNT=$(wc -l < "$SESSION_FILE" | tr -d ' ')
echo "Events captured: $EVENT_COUNT"

# Verify we have the expected event types
echo ""
echo "Captured event types:"
jq -r '.event_type' "$SESSION_FILE" | sort | uniq -c

# Check for specific events
HAS_SESSION_START=$(jq -e 'select(.event_type == "session_start")' "$SESSION_FILE" > /dev/null 2>&1 && echo "yes" || echo "no")
HAS_USER_MSG=$(jq -e 'select(.event_type == "message" and .data.role == "user")' "$SESSION_FILE" > /dev/null 2>&1 && echo "yes" || echo "no")
HAS_ASSISTANT_MSG=$(jq -e 'select(.event_type == "message" and .data.role == "assistant")' "$SESSION_FILE" > /dev/null 2>&1 && echo "yes" || echo "no")
HAS_TOOL_USE=$(jq -e 'select(.event_type == "tool_use")' "$SESSION_FILE" > /dev/null 2>&1 && echo "yes" || echo "no")
HAS_TOOL_RESULT=$(jq -e 'select(.event_type == "tool_result")' "$SESSION_FILE" > /dev/null 2>&1 && echo "yes" || echo "no")

# Count specific tool types
WRITE_COUNT=$(jq -r 'select(.event_type == "tool_use" and .data.tool_name == "Write") | .data.tool_name' "$SESSION_FILE" | wc -l | tr -d ' ')
EDIT_COUNT=$(jq -r 'select(.event_type == "tool_use" and .data.tool_name == "Edit") | .data.tool_name' "$SESSION_FILE" | wc -l | tr -d ' ')
BASH_COUNT=$(jq -r 'select(.event_type == "tool_use" and .data.tool_name == "Bash") | .data.tool_name' "$SESSION_FILE" | wc -l | tr -d ' ')

echo ""
echo "Event verification:"
echo "  session_start:  $HAS_SESSION_START"
echo "  user message:   $HAS_USER_MSG"
echo "  assistant msg:  $HAS_ASSISTANT_MSG"
echo "  tool_use:       $HAS_TOOL_USE"
echo "  tool_result:    $HAS_TOOL_RESULT"
echo ""
echo "Tool usage breakdown:"
echo "  Write calls:    $WRITE_COUNT"
echo "  Edit calls:     $EDIT_COUNT"
echo "  Bash calls:     $BASH_COUNT"

# Show tool calls with their inputs
echo ""
echo "Tool calls captured:"
jq -r 'select(.event_type == "tool_use") | "  - \(.data.tool_name): \(.data.input | tostring | .[0:60])..."' "$SESSION_FILE" 2>/dev/null || true

# Verify assistant response
ASSISTANT_CONTENT=$(jq -r 'select(.event_type == "message" and .data.role == "assistant") | .data.content[0].text // empty' "$SESSION_FILE" | tail -1)
echo ""
echo "Final assistant response: ${ASSISTANT_CONTENT:0:80}..."

echo ""
echo "--- Step 4: Verify UI API (if running) ---"

UI_PORT=3000
if curl -s "http://localhost:$UI_PORT/api/sessions" > /dev/null 2>&1; then
    echo "UI server is running on port $UI_PORT"

    # Check if session appears in API
    API_SESSIONS=$(curl -s "http://localhost:$UI_PORT/api/sessions")
    if echo "$API_SESSIONS" | jq -e ".sessions[] | select(.session_id == \"$SESSION_ID\")" > /dev/null 2>&1; then
        echo "✓ Session found in UI API"

        # Fetch session detail
        SESSION_DETAIL=$(curl -s "http://localhost:$UI_PORT/api/sessions/$SESSION_ID")
        API_EVENT_COUNT=$(echo "$SESSION_DETAIL" | jq '.session.events | length')
        echo "  Events in API: $API_EVENT_COUNT"

        # Check for specific event types in API
        echo "  Event types in API:"
        echo "$SESSION_DETAIL" | jq -r '.session.events[].event_type' | sort | uniq -c | sed 's/^/    /'

        # Verify events match
        if [[ "$API_EVENT_COUNT" -ge 3 ]]; then
            echo "✓ UI API has events"
        else
            echo "⚠ UI API event count seems low: $API_EVENT_COUNT"
        fi
    else
        echo "⚠ Session not found in UI API"
        echo "  Available sessions:"
        echo "$API_SESSIONS" | jq -r '.sessions[0:3] | .[].session_id' | sed 's/^/    /'
    fi
else
    echo "UI server not running on port $UI_PORT (skipping UI verification)"
fi

echo ""
echo "--- Step 5: Summary ---"

PASS=true
[[ "$HAS_SESSION_START" == "yes" ]] || { echo "FAIL: Missing session_start"; PASS=false; }
[[ "$HAS_USER_MSG" == "yes" ]]      || { echo "FAIL: Missing user message"; PASS=false; }
[[ "$HAS_ASSISTANT_MSG" == "yes" ]] || { echo "FAIL: Missing assistant message"; PASS=false; }

if $PASS; then
    echo ""
    echo "✅ E2E TEST PASSED"
    echo ""
    echo "Full session data:"
    jq -c '{event_type, role: .data.role, tool: .data.tool_name, content_preview: (if .data.content | type == "array" then .data.content[0].text[:50] else (.data.content[:50] // null) end)}' "$SESSION_FILE" 2>/dev/null || jq -c '{event_type, role: .data.role, tool: .data.tool_name}' "$SESSION_FILE"
    exit 0
else
    echo ""
    echo "❌ E2E TEST FAILED"
    echo ""
    echo "Debug: Full session file:"
    cat "$SESSION_FILE" | jq .
    exit 1
fi
