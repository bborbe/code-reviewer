---
status: completed
summary: Implemented git worktree management with Fetch, CreateWorktree, RemoveWorktree methods and comprehensive tests
container: pr-reviewer-003-git-operations
dark-factory-version: v0.14.5
created: "2026-03-04T14:10:33Z"
queued: "2026-03-04T14:10:33Z"
started: "2026-03-04T14:10:33Z"
completed: "2026-03-04T14:15:40Z"
---
<objective>
Implement git operations: fetch remote, create worktree for a PR branch, clean up worktree. The worktree isolates the PR review from the working copy. Cleanup must happen on success, failure, and signals (SIGINT/SIGTERM).
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config and pkg/github exist.
Spec: specs/001-mvp-review-single-pr.md — Desired Behavior 5-6, 9 and Failure Modes.
</context>

<requirements>
1. Create `pkg/git/git.go`:
   - Define `WorktreeManager` interface:
     ```go
     type WorktreeManager interface {
         Fetch(ctx context.Context, repoPath string) error
         CreateWorktree(ctx context.Context, repoPath string, branch string, prNumber int) (worktreePath string, err error)
         RemoveWorktree(ctx context.Context, repoPath string, worktreePath string) error
     }
     ```
   - Implement `worktreeManager` struct
   - Constructor: `NewWorktreeManager() WorktreeManager`

2. `Fetch` implementation:
   - Run `git -C <repoPath> fetch --all --prune` via exec.Command
   - Validate repoPath exists and is a git repo first
   - On failure: return error with git stderr

3. `CreateWorktree` implementation:
   - Worktree path: `<repoPath>/.worktrees/pr-<number>` (deterministic, inside repo)
   - If worktree path already exists (stale from previous run): call RemoveWorktree first, then create fresh
   - Run `git -C <repoPath> worktree add <worktreePath> <branch>`
   - On failure: return error with git stderr
   - Return the worktree path

4. `RemoveWorktree` implementation:
   - Run `git -C <repoPath> worktree remove <worktreePath> --force`
   - If worktree doesn't exist: return nil (idempotent)
   - On failure: return error with git stderr

5. Create `pkg/git/git_test.go`:
   - Ginkgo v2 + Gomega test suite
   - Test Fetch with non-existent path → error
   - Test Fetch with non-git directory → error
   - Test CreateWorktree returns deterministic path
   - Test RemoveWorktree on non-existent path → no error (idempotent)
   - Use real git repos in temp dirs for integration tests:
     - Create a temp git repo with `git init`, add a commit, create a branch
     - Test full cycle: Fetch (local) → CreateWorktree → verify dir exists → RemoveWorktree → verify dir gone

6. Add validation errors matching spec:
   - Path doesn't exist: `"local path not found: <path>"`
   - Not a git repo: `"not a git repo: <path>"`
   - Branch not found: `"branch not found: <branch>"`
</requirements>

<implementation>
Use exec.Command for git operations (not go-git library):

```go
func (m *worktreeManager) runGit(ctx context.Context, repoPath string, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("git %s: %s", args[0], stderr.String())
    }
    return stdout.String(), nil
}
```

Worktree path pattern:
```go
func worktreePath(repoPath string, prNumber int) string {
    return filepath.Join(repoPath, ".worktrees", fmt.Sprintf("pr-%d", prNumber))
}
```
</implementation>

<constraints>
- Must use `git` CLI via exec.Command — NOT go-git library
- Must use `git worktree add/remove` — never checkout in the main working tree
- Worktree must be inside the repo directory, not in /tmp
- Worktree path: derived deterministically from repo path + PR number
- Always clean up worktree on exit (including on error/signal)
- Package name must be `git_test` (external test package)
- Use Ginkgo v2 + Gomega for tests
- Do NOT touch pkg/config, pkg/github, or main.go
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- pkg/git package exists with WorktreeManager interface
- Stale worktree from previous run is cleaned up automatically
- Worktree path is deterministic: <repoPath>/.worktrees/pr-<number>
- RemoveWorktree is idempotent
- Error messages match spec failure modes
- Integration test covers full fetch → create → remove cycle
- make precommit passes
</success_criteria>
