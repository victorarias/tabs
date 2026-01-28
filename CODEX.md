# Tabs Implementation - Ralph Loop

You are implementing the tabs project according to the specifications in docs/.

## CRITICAL: Design System

**When implementing UI (Phase 2 & 3), you MUST follow:**
- **docs/07-frontend-design-brief.md** - The definitive design system
  - Typography: DM Serif Display + IBM Plex Sans (NOT Inter!)
  - Colors: Warm "vinyl & mahogany" palette with amber accents
  - Musical theme with guitar pick icons and staff line spacing
  - Animations: Musical micro-interactions (staggered notes, metronome pulse)

**DO NOT use** the generic design from docs/05-local-ui-flows.md. That doc has page layouts, but doc 07 has the correct visual design language.

## Your Task

1. Read `prd.json` to see all implementation stories
2. Read `progress.txt` to see what has been learned so far
3. Find the next story where `passes: false` (lowest priority number)
4. Implement that story following the specs in docs/
   - **For UI stories:** Use design system from docs/07-frontend-design-brief.md
5. Run quality checks:
   - `make build` - must succeed
   - `make test` - must pass (once tests exist)
   - `go vet ./...` - must pass
   - **For UI stories**: Use Playwright to verify browser functionality works
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
