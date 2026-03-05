---
status: created
---
<objective>
Replace git worktree with `git clone --local` to /tmp. Worktrees create a `.git` file with absolute host paths that break when mounted into Docker containers. A local clone creates a standalone `.git` directory that works with Docker volume mounts.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read pkg/git/git.go for the current WorktreeManager implementation.
Read pkg/git/git_test.go for existing tests.
Read main.go for how createWorktreeAndFetch is used.
</context>

<requirements>
1. Replace `CreateWorktree` in WorktreeManager with a `CreateClone` method:
   - `CreateClone(ctx, repoPath, branch string, prNumber int) (clonePath string, err error)`
   - First run `git fetch --all --prune` in repoPath (existing Fetch method)
   - Then run `git clone --local --no-checkout <repoPath> <clonePath>` where clonePath is `/tmp/pr-reviewer-<repoName>-pr-<prNumber>`
   - Then run `git checkout origin/<branch>` in the clone directory (detached HEAD)
   - Remove stale clone directory if it already exists before cloning

2. Replace `RemoveWorktree` with `RemoveClone`:
   - `RemoveClone(ctx, clonePath string) error`
   - Simply `os.RemoveAll(clonePath)`
   - No longer needs repoPath parameter (clone is standalone)
   - Idempotent: returns nil if path doesn't exist

3. Update the `WorktreeManager` interface:
   - Rename to `RepoManager` (or keep name if renaming causes too much churn)
   - Replace `CreateWorktree` signature with `CreateClone`
   - Replace `RemoveWorktree` signature with `RemoveClone`
   - Keep `Fetch` method unchanged

4. Update `createWorktreeAndFetch` in main.go:
   - Call `CreateClone` instead of `CreateWorktree`
   - Cleanup calls `RemoveClone` instead of `RemoveWorktree`
   - Rename function to `createCloneAndFetch` (or similar)

5. Update all tests to match new interface.

6. Regenerate mocks with `go generate ./...`
</requirements>

<constraints>
- Clone path must be under os.TempDir() (same as current worktree path)
- Use `git clone --local` (hardlinks objects, fast, no network)
- Use `--no-checkout` + separate `git checkout` to control branch selection
- The clone must be a standalone git repo (no symlinks back to source)
- All existing test patterns must be preserved (adapted for new method names)
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
