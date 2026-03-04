---
status: completed
summary: Implemented Claude Code review execution with Reviewer interface and claudeReviewer implementation
container: pr-reviewer-004-review-execution
dark-factory-version: v0.14.5
created: "2026-03-04T14:16:45Z"
queued: "2026-03-04T14:16:45Z"
started: "2026-03-04T14:16:45Z"
completed: "2026-03-04T14:22:42Z"
---
<objective>
Implement Claude Code review execution. Run `claude` CLI in a worktree directory to produce a code review using the repo's CLAUDE.md and project context. Capture the review output for posting.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config, pkg/github, pkg/git exist.
Spec: specs/001-mvp-review-single-pr.md — Desired Behavior 7, Constraints, Failure Modes.
</context>

<requirements>
1. Create `pkg/review/review.go`:
   - Define `Reviewer` interface:
     ```go
     type Reviewer interface {
         Review(ctx context.Context, worktreePath string) (string, error)
     }
     ```
   - Implement `claudeReviewer` struct
   - Constructor: `NewClaudeReviewer() Reviewer`

2. `Review` implementation:
   - Check that `claude` is in PATH: use `exec.LookPath("claude")`
   - If not found: return error `"claude not found in PATH"`
   - Run: `claude --print "/code-review"` with working directory set to worktreePath
   - Capture stdout (the review text)
   - If claude exits non-zero: return error with stderr content
   - Return the review text (stdout)

3. Create `pkg/review/review_test.go`:
   - Ginkgo v2 + Gomega test suite
   - Since we can't run real `claude` in tests, use counterfeiter mock for the Reviewer interface
   - Test the interface contract: Review returns string on success, error on failure
   - Add `//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate` to package
   - Add `//counterfeiter:generate . Reviewer` comment to interface

4. Create `pkg/review/mocks_generate.go` (or use go:generate comment in review.go):
   - Generate counterfeiter mock for Reviewer interface

5. Run `go generate ./...` to generate mocks
</requirements>

<implementation>
```go
func (r *claudeReviewer) Review(ctx context.Context, worktreePath string) (string, error) {
    claudePath, err := exec.LookPath("claude")
    if err != nil {
        return "", fmt.Errorf("claude not found in PATH")
    }
    cmd := exec.CommandContext(ctx, claudePath, "--print", "/code-review")
    cmd.Dir = worktreePath
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("claude review failed: %s", stderr.String())
    }
    return stdout.String(), nil
}
```
</implementation>

<constraints>
- Must use `claude` CLI for reviews — not the SDK, not the API directly
- Must run claude with --print flag (non-interactive, outputs to stdout)
- Must set working directory to worktreePath so claude picks up the repo's CLAUDE.md
- Claude output: captured as-is — trusted internal tool, no sanitization needed
- Error messages must match spec: "claude not found in PATH"
- Package name must be `review_test` (external test package)
- Use Ginkgo v2 + Gomega for tests
- Use counterfeiter for mocks
- Do NOT touch other packages or main.go
</constraints>

<verification>
Run `go generate ./...` -- must succeed.
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- pkg/review package exists with Reviewer interface and claudeReviewer implementation
- claude CLI invoked with --print "/code-review" in worktree directory
- Missing claude binary detected with clear error message
- Counterfeiter mock generated for Reviewer interface
- make precommit passes
</success_criteria>
