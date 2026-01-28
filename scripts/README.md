# Tabs Implementation Scripts

## Ralph Loop (`ralph.sh`)

Autonomous implementation loop using Codex CLI based on the [Ralph technique](https://www.aihero.dev/getting-started-with-ralph).

### What is Ralph?

Ralph is a bash loop that repeatedly calls an AI coding agent with the same prompt. The agent:
1. Reads a PRD (Product Requirements Document) to see all tasks
2. Reads a progress file to see what's been learned
3. Picks the next incomplete task
4. Implements it following the specs
5. Runs quality checks (build, tests, vet)
6. Commits if checks pass
7. Updates PRD and progress file
8. Repeats until all tasks are done

Memory persists through:
- **Git history** - previous commits
- **progress.txt** - append-only learnings and gotchas
- **prd.json** - task completion status

### Prerequisites

- [Codex CLI](https://github.com/codex) installed and configured
- Python 3 (for JSON manipulation)
- Go 1.23+ (for building)

### Usage

```bash
# Run with default max iterations (50)
./scripts/ralph.sh

# Run with custom max iterations
./scripts/ralph.sh 100
```

### Files Created

On first run, Ralph creates:

- **prd.json** - Product requirements with 8 implementation phases
- **progress.txt** - Learnings and decisions from each iteration
- **CODEX.md** - Prompt template for Codex

### How It Works

Each iteration:

1. **Pick next task** - Finds lowest priority incomplete story in `prd.json`
2. **Call Codex** - Runs `codex exec` with the `CODEX.md` prompt
3. **Quality checks**:
   - `make build` must succeed
   - `go vet ./...` must pass
   - `make test` must pass (when tests exist)
4. **Commit** - If checks pass, commits changes
5. **Update progress** - Codex updates `prd.json` and `progress.txt`
6. **Repeat** - Moves to next story

### Monitoring Progress

```bash
# Check which stories are complete
cat prd.json | jq '.stories[] | select(.passes == true) | .title'

# Check which stories remain
cat prd.json | jq '.stories[] | select(.passes == false) | .title'

# Read learnings
cat progress.txt
```

### Stopping and Resuming

- Press Ctrl+C to stop Ralph
- Ralph is idempotent - you can resume by running `./scripts/ralph.sh` again
- It will pick up from the next incomplete story

### Tips

1. **Monitor the first few iterations** - Make sure Codex understands the specs correctly
2. **Review commits** - Ralph runs autonomously but you should review the code
3. **Update progress.txt manually** - Add important patterns or gotchas for Codex to learn from
4. **Keep specs updated** - If requirements change, update docs/ and progress.txt

### Troubleshooting

**Codex fails with errors:**
- Check the output for details
- You may need to manually fix issues
- Update `progress.txt` with the fix so Codex learns

**Build fails:**
- Ralph will stop and show the error
- Fix manually, commit, and resume Ralph

**Infinite loop on one story:**
- Story may be too large - break it down in `prd.json`
- Specs may be ambiguous - clarify in docs/
- Add hints in `progress.txt`

## References

- [Getting Started With Ralph](https://www.aihero.dev/getting-started-with-ralph)
- [Ralph GitHub](https://github.com/snarktank/ralph)
- [The Ralph Wiggum Approach](https://dev.to/sivarampg/the-ralph-wiggum-approach-running-ai-coding-agents-for-hours-not-minutes-57c1)
