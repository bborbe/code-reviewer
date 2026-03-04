<objective>
Wire everything together in main.go: parse URL, load config, fetch PR metadata, create worktree, run review, post comment, clean up. Also implement the GitHub comment posting and PR metadata fetching via `gh` CLI. This is the final prompt — after this, the tool works end-to-end.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config, pkg/github, pkg/git, pkg/review all exist and tested.
Spec: specs/001-mvp-review-single-pr.md — all Desired Behaviors, all Failure Modes.
</context>

<requirements>
1. Create `pkg/github/client.go`:
   - Define `Client` interface:
     ```go
     type Client interface {
         GetPRBranch(ctx context.Context, owner, repo string, number int) (string, error)
         PostComment(ctx context.Context, owner, repo string, number int, body string) error
     }
     ```
   - Implement `ghClient` struct using `gh` CLI
   - Constructor: `NewGHClient() Client`

2. `GetPRBranch` implementation:
   - Run: `gh pr view <number> --repo <owner>/<repo> --json headRefName --jq .headRefName`
   - Return the branch name
   - On failure: return error with gh stderr

3. `PostComment` implementation:
   - Run: `gh pr comment <number> --repo <owner>/<repo> --body <body>`
   - On failure: return error with gh stderr (but caller should still print review to stdout)

4. Add counterfeiter mock generation for Client interface:
   - Add `//counterfeiter:generate . Client` comment
   - Run `go generate ./...`

5. Create `pkg/github/client_test.go`:
   - Test the interface contract via counterfeiter mock
   - Cannot test real `gh` calls, but verify mock behavior

6. Rewrite `main.go` — full CLI wiring:
   ```
   a. Parse args (require exactly 1 URL argument)
   b. Parse PR URL → PRInfo
   c. Load config from ~/.pr-reviewer.yaml
   d. Find local repo path from config
   e. Get PR branch name via gh CLI
   f. Fetch in local repo
   g. Create worktree for PR branch
   h. defer: remove worktree (cleanup on success, failure, signal)
   i. Run claude review in worktree
   j. Print review to stdout (always, even if posting fails)
   k. Post review as PR comment via gh CLI
   l. Exit 0 on success, non-zero on failure
   ```

7. Signal handling:
   - Set up context with signal.NotifyContext for SIGINT, SIGTERM
   - The deferred worktree removal handles cleanup
   - On signal: context cancels, deferred cleanup runs, exit

8. Stdout fallback:
   - Always print review to stdout (step j) BEFORE posting comment (step k)
   - If posting fails: print error to stderr, exit 1
   - This way the user always has the review text even if GitHub API fails

9. Error handling — each failure mode from spec must produce the correct message:
   - No arguments → "usage: pr-reviewer <pr-url>"
   - Bad URL → "unsupported URL format: <url>"
   - No config → "config not found: ~/.pr-reviewer.yaml"
   - Repo not in config → "repo not found in config, add to ~/.pr-reviewer.yaml: <url>"
   - Local path missing → "local path not found: <path>"
   - Not a git repo → "not a git repo: <path>"
   - Claude not found → "claude not found in PATH"
</requirements>

<implementation>
Main flow pattern:
```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    if err := run(ctx); err != nil {
        fmt.Fprintf(os.Stderr, "error: %v\n", err)
        os.Exit(1)
    }
}

func run(ctx context.Context) error {
    if len(os.Args) < 2 {
        return fmt.Errorf("usage: pr-reviewer <pr-url>")
    }
    // ... wire all components
}
```

For gh CLI commands:
```go
func (c *ghClient) GetPRBranch(ctx context.Context, owner, repo string, number int) (string, error) {
    cmd := exec.CommandContext(ctx, "gh", "pr", "view",
        strconv.Itoa(number),
        "--repo", fmt.Sprintf("%s/%s", owner, repo),
        "--json", "headRefName",
        "--jq", ".headRefName",
    )
    // capture stdout/stderr
}
```
</implementation>

<constraints>
- Config file location: ~/.pr-reviewer.yaml
- Must use `gh` CLI for GitHub operations (already authenticated)
- Must use `claude` CLI for reviews — not the SDK, not the API directly
- Must use `git worktree add/remove` — never checkout in the main working tree
- Worktree must be inside the repo directory, not in /tmp
- Always clean up worktree on exit (including on error/signal)
- Review must be printed to stdout even if posting fails
- Package name must be `github_test` (external test package) for client_test.go
- Use Ginkgo v2 + Gomega for tests
- Use counterfeiter for mocks
</constraints>

<verification>
Run `go generate ./...` -- must succeed.
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- main.go wires all components together in correct order
- Signal handling cleans up worktree on SIGINT/SIGTERM
- Review printed to stdout before posting attempt
- All error messages match spec failure modes exactly
- GitHub Client interface with counterfeiter mock generated
- make precommit passes
- `go build -o pr-reviewer .` produces a working binary
</success_criteria>
