# E2E Testing for Tabs

## Overview

Tabs captures Claude Code sessions via hooks. To verify the full pipeline works, we have an E2E test that:

1. **Drives Claude Code** programmatically using `--output-format stream-json`
2. **Runs multi-turn conversations** using `--resume` for true interleaved user/assistant turns
3. **Exercises multiple tool types**: Write, Edit, Read, Bash
4. **Verifies daemon capture**: Checks session files contain all expected events
5. **Verifies UI API** (if running): Confirms events are accessible via HTTP

## Running the E2E Test

```bash
./scripts/e2e-test.sh
```

### Prerequisites

- `tabs-daemon` must be running (script will start it if needed)
- `claude` CLI must be available
- For UI verification, run `tabs-cli ui` first

### What It Tests

The test creates a 5-turn conversation:

| Turn | User Prompt | Expected Tools |
|------|-------------|----------------|
| 1 | Write config.json | Write |
| 2 | Edit version to 1.1.0 | Read, Edit |
| 3 | Edit debug to true | Edit |
| 4 | Run cat config.json | Bash |
| 5 | Say MULTI_TURN_COMPLETE | (none) |

### Expected Output

```
Events captured: 26
- 14 messages (5 user + 9 assistant)
- 5 tool_use
- 5 tool_result
- 2 session_start

✅ E2E TEST PASSED
```

## Key Claude CLI Flags for Testing

```bash
# Single turn with full event capture
claude -p "prompt" \
    --output-format stream-json \
    --verbose \
    --dangerously-skip-permissions \
    --model haiku

# Multi-turn: first message with explicit session ID
claude -p "first message" \
    --session-id "$UUID" \
    --output-format stream-json \
    --verbose

# Multi-turn: subsequent messages resume the session
claude -p "follow up" \
    --resume "$UUID" \
    --output-format stream-json \
    --verbose
```

## Verifying Captured Events

After running Claude, check the session file:

```bash
# Find latest session
ls -t ~/.tabs/sessions/$(date +%Y-%m-%d)/*.jsonl | head -1

# Count event types
jq -r '.event_type' SESSION_FILE | sort | uniq -c

# Check specific tools were captured
jq 'select(.event_type == "tool_use") | .data.tool_name' SESSION_FILE
```

## Common Issues

### "no such file or directory" on SessionStart hook

This happens when running from temp directories - Claude Code's transcript path doesn't exist yet when the hook fires. This is benign; subsequent hooks will capture the data.

### "0 events" captured

The hook ran before the transcript was written. This is normal for UserPromptSubmit - the full capture happens on the Stop hook.

### Missing tool_use/tool_result events

Check that `extractToolUse` and `extractToolResult` in `internal/daemon/claude.go` are correctly parsing `message.content[]` arrays.

## Adding New E2E Scenarios

To test additional scenarios, add new turns to `scripts/e2e-test.sh`:

```bash
# Example: Test file deletion
claude -p "Delete the config.json file using rm" \
    --resume "$SESSION_UUID" \
    $COMMON_ARGS \
    2>&1 | tee -a "$CLAUDE_OUTPUT"
```

Then update the verification section to check for the expected events.

## For Future Agents

When making changes to the tabs daemon or event capture:

1. **Always run `./scripts/e2e-test.sh`** after changes
2. If adding new event types, update the test to verify them
3. If changing parsing logic, check that existing event types still work
4. The test should pass completely before considering a change done
