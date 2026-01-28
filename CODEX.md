# Tabs Implementation - Ralph Loop

You are implementing the tabs project according to the specifications in docs/.

## Design Philosophy

**Musical Theme:** tabs = guitar tablatures. The UI should feel like reading beautifully typeset sheet music.

**When implementing UI (Phase 2 & 3):**
- Typography: DM Serif Display (headings) + IBM Plex Sans (body) - see docs/05
- Colors: Warm "vinyl & mahogany" palette with amber accents (#d97706)
- Musical micro-interactions: staggered reveals (40ms), crescendo hovers, metronome pulse
- Custom guitar pick icon (not generic dots)
- 8px grid system (like musical staff lines)

## Your Task

1. Read `prd.json` to see all implementation stories
2. Read `progress.txt` to see what has been learned so far
3. Find the next story where `passes: false` (lowest priority number)
4. Implement that story following the specs in docs/
5. Run quality checks:
   - `make build` - must succeed
   - `make test` - must pass (once tests exist)
   - `go vet ./...` - must pass
   - **For UI stories**: Use Playwright to verify browser functionality works
   - **For remote server stories**: Use Docker Compose for PostgreSQL
     - Start: `docker compose up -d`
     - Test migrations and API
     - Stop: `docker compose down`
6. If checks pass:
   - Commit your changes with a clear commit message
   - Update `prd.json`: set `passes: true` for the completed story
   - Append learnings to `progress.txt`: document any gotchas, patterns, or decisions
7. If checks fail:
   - Fix the issues
   - Do not mark story as complete
   - Document the failure in progress.txt

## Browser Testing with Playwright

**CRITICAL**: When implementing UI features (Phase 2 & 3), you MUST verify them with Playwright:

1. Start the web server (local or remote)
2. Use Playwright MCP tools to:
   - Navigate to the UI
   - Verify key elements render
   - Test interactions (clicks, navigation, forms)
   - Capture screenshots if something looks wrong
3. Only mark the story complete if Playwright tests pass

Example Playwright verification:
```javascript
// Navigate to local UI
await page.goto('http://localhost:3787');

// Verify timeline loads
await expect(page.getByRole('heading', { name: /sessions/i })).toBeVisible();

// Click session
await page.getByRole('button').first().click();

// Verify detail page
await expect(page).toHaveURL(/sessions\//);
```

## Guidelines

- Follow the specs in docs/ exactly
- Keep it simple - no over-engineering
- Test as you go
- **Use Playwright for all UI verification**
- Document non-obvious decisions in progress.txt
- Each iteration should complete one story from the PRD

## Current Iteration

Check prd.json for the next incomplete story and implement it.
