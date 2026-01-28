# Claude Code Ralph Supervisor

You are supervising Codex in a Ralph loop. Your role is **Human-In-The-Loop** - you drive Codex, review its work, and verify quality.

## Your Mission (One Iteration)

1. **Read the PRD:**
   - Read `prd.json` to find the next story where `passes: false`
   - Read `progress.txt` to see what's been learned

2. **Call Codex to implement the story:**
   - Use bash to call: `codex exec --skip-git-repo-check --full-auto -C /home/victor/projects/tab "$(cat CODEX.md)"`
   - Monitor Codex's output
   - Let Codex complete the implementation

3. **Review Codex's work:**
   - Check what files Codex created/modified
   - Read the code to ensure it matches specs
   - Look for issues, bugs, or deviations from design

4. **Actually test the functionality:**
   - **Don't just verify with screenshots - actually USE the code!**
   - For CLI: Run the commands, verify they work
   - For daemon: Start it, send events, check JSONL files created
   - For hooks: Trigger an actual Claude Code session, verify capture works
   - For UI: Use Playwright to interact (click, type, navigate), not just screenshot
   - For remote server: Start Docker Compose, test API endpoints, verify database
   - **Be thorough - test the acceptance criteria from prd.json**

5. **Run quality checks:**
   - Run `make build` - must succeed
   - Run `go vet ./...` - must pass
   - Run `make test` - must pass (if tests exist)
   - All functional tests from step 4 must pass

6. **If checks pass:**
   - Update `prd.json`: Set `passes: true` for the completed story
   - Append to `progress.txt`: Document learnings, gotchas, decisions
   - Commit if Codex didn't already
   - Report success

7. **If checks fail:**
   - Analyze the failure
   - Fix the issue yourself OR call Codex again with fix instructions
   - Do NOT mark story as complete
   - Document the failure in progress.txt

8. **Exit cleanly:**
   - The bash loop will call you again for the next story
   - Don't try to continue to the next story yourself

---

## Testing Examples (Be This Thorough!)

**Phase 1 - CLI & Daemon:**
```bash
# Build binaries
make build

# Start daemon in background
./bin/tabs-daemon &
DAEMON_PID=$!

# Test CLI commands
./bin/tabs-cli status                              # Should show daemon running
./bin/tabs-cli config --set server.url=https://test.com
cat ~/.tabs/config.toml                            # Verify config saved

# Kill daemon
kill $DAEMON_PID
```

**Phase 1 - Hook Integration:**
```bash
# Install hook (command varies based on implementation)
./bin/tabs-cli install-hooks

# Verify hook installed
ls -la ~/.claude/hooks/
cat ~/.claude/hooks/on-user-prompt-submit.sh       # Read the hook script

# CRITICAL: Test with THREE separate multi-turn Claude Code sessions
# Session 1: Multi-turn conversation
cd /tmp/test-project-1
claude "write a hello world function"
# Continue the session with follow-up prompts:
# - "now add a goodbye function"
# - "write tests for both functions"
# Exit Claude Code (Ctrl+D or /exit)

# Session 2: Different project, multi-turn
cd /tmp/test-project-2
claude "create a fibonacci function"
# Continue:
# - "optimize it with memoization"
# - "add error handling"
# Exit Claude Code

# Session 3: Another project, multi-turn
cd /tmp/test-project-3
claude "implement a binary search"
# Continue:
# - "add edge case handling"
# - "write documentation"
# Exit Claude Code

# Verify ALL THREE sessions captured
ls ~/.tabs/sessions/$(date +%Y-%m-%d)/             # Should see 3 JSONL files
wc -l ~/.tabs/sessions/$(date +%Y-%m-%d)/*.jsonl   # Each should have multiple events

# Check session IDs are different
grep -h "session_id" ~/.tabs/sessions/$(date +%Y-%m-%d)/*.jsonl | sort -u  # Should see 3 unique IDs

# Verify multi-turn captured (each file should have multiple messages)
for f in ~/.tabs/sessions/$(date +%Y-%m-%d)/*.jsonl; do
  echo "=== $f ==="
  grep -c "\"type\":\"message\"" "$f"  # Should be > 1 for multi-turn
done
```

**Phase 2 - Local UI:**
```javascript
// Use Playwright - don't just screenshot!
await page.goto('http://localhost:3787');

// Verify timeline loads
await expect(page.locator('.session-card')).toBeVisible();

// Click a session
await page.locator('.session-card').first().click();
await expect(page).toHaveURL(/sessions\//);

// Test search
await page.locator('input[type="search"]').fill('hello world');
await expect(page.locator('.session-card')).toHaveCount(1);

// Test share modal
await page.locator('button:has-text("Share")').click();
await expect(page.locator('dialog')).toBeVisible();
await page.locator('input[name="tag"]').fill('test:demo');
await page.locator('button:has-text("Share →")').click();
```

**Phase 3 - Remote Server:**
```bash
# Start PostgreSQL
docker compose up -d
docker compose ps  # Wait for "healthy"

# Run migrations
./bin/tabs-server migrate

# Start server
./bin/tabs-server &
SERVER_PID=$!

# Test API
curl http://localhost:8080/api/sessions  # Should return empty array

# Push a session
./bin/tabs-cli push <session-id>

# Verify in database
docker compose exec postgres psql -U tabs -c "SELECT id, cwd FROM sessions;"

# Test with Playwright
# (navigate to remote UI, verify session appears, test filtering, etc.)

# Cleanup
kill $SERVER_PID
docker compose down
```

---

## Design Guidelines (Remind Codex if it deviates)

**Typography:**
- DM Serif Display (headings) + IBM Plex Sans (body)
- NOT Inter or generic fonts

**Colors:**
- Local UI: Amber accent (#d97706) - "vinyl & mahogany"
- Remote UI: Emerald accent (#059669)
- Warm, aged paper backgrounds (#faf9f7)

**Musical Theme:**
- Guitar pick icon (not generic dots)
- Staggered reveals (40ms delay)
- Crescendo hover effects
- Metronome pulse for loading

**Database:**
- Use Docker Compose for PostgreSQL testing
- Start: `docker compose up -d`

## Key Points

- **Be critical:** If Codex's code doesn't match the design specs, flag it
- **Take screenshots:** For UI work, actually see what it looks like
- **Verify, don't assume:** Run the quality checks yourself
- **One story per iteration:** Complete one story, then exit
- **Update the PRD:** This is how the bash loop knows to continue

## Current Iteration

Find the next incomplete story in `prd.json` and supervise Codex implementing it.
