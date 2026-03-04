---
status: completed
summary: Added progress logging to stderr with -v verbose flag
container: pr-reviewer-009-verbose-progress-logging
dark-factory-version: v0.14.5
created: "2026-03-04T20:53:29Z"
queued: "2026-03-04T20:53:29Z"
started: "2026-03-04T20:53:29Z"
completed: "2026-03-04T20:58:46Z"
---
<objective>
Add progress logging to stderr so the user knows pr-reviewer is alive during long operations. Default shows minimal output, -v flag shows detailed steps.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: main.go wires all components together.
Problem: claude --print takes minutes with zero output. User thinks tool is hung.
</context>

<requirements>
1. Add `-v` flag to main.go:
   - Parse with `flag` package before the URL argument
   - `-v` enables verbose mode
   - URL is the first non-flag argument: `flag.Args()[0]`

2. Default output (no -v) — minimal, just enough to show progress:
   - Before claude review: `reviewing PR #<number> (<repo>)...`
   - After review done: `posting comment...`
   - On success: `done`
   - All to stderr via `fmt.Fprintf(os.Stderr, ...)`

3. Verbose output (-v) — detailed steps, all to stderr:
   - `parsing URL: <url>`
   - `loading config: ~/.pr-reviewer.yaml`
   - `repo: <path>`
   - `fetching branch: <branch>`
   - `creating worktree: <worktreePath>`
   - `running review... (this may take a few minutes)`
   - `posting comment...`
   - `done`

4. Implementation approach:
   - Pass a `verbose bool` to the `run` function
   - Use a simple helper: `func logProgress(verbose bool, always bool, format string, args ...interface{})`
   - `always=true` prints in both modes, `always=false` only in verbose mode
   - Or simpler: two functions `logAlways` and `logVerbose`

5. Update usage message to include -v flag:
   - `"usage: pr-reviewer [-v] <pr-url>"`

6. Stdout must remain clean — only the review text goes to stdout.
   All progress goes to stderr.
</requirements>

<constraints>
- Use `flag` package from stdlib for flag parsing
- All progress output to stderr (never stdout)
- Stdout reserved for review text only
- Do NOT touch pkg/ packages — only main.go changes
- Use Ginkgo v2 + Gomega for any new tests
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- Default mode shows minimal progress (PR number + repo before review, posting, done)
- Verbose mode (-v) shows all steps
- All progress on stderr, review on stdout
- Usage message updated
- make precommit passes
</success_criteria>
