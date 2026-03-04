<objective>
Fix "branch already checked out" error when creating worktree for a PR branch that is currently checked out in the main working tree.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/git exists with WorktreeManager interface.
Bug: `git worktree add <path> <branch>` fails with "already used by worktree" when the local repo has that branch checked out.
This happens when the user's working copy is on the same branch as the PR.
</context>

<requirements>
1. Fix `CreateWorktree` in `pkg/git/git.go`:
   - Instead of `git worktree add <path> <branch>`, use `git worktree add --detach <path>` then `git -C <path> checkout <branch>` — NO, this has the same problem.
   - Correct fix: use `git worktree add <path> --detach <commitish>` where commitish is `origin/<branch>` or the remote tracking ref.
   - Alternative: use `git worktree add <path> --detach` then `git -C <path> reset --hard origin/<branch>`.
   - Simplest correct fix: `git worktree add --detach <path> origin/<branch>` — creates detached HEAD worktree at the remote branch's commit. This avoids the "branch already checked out" error entirely.

2. Update the git command in CreateWorktree:
   - Change from: `git -C <repoPath> worktree add <worktreePath> <branch>`
   - Change to: `git -C <repoPath> worktree add --detach <worktreePath> origin/<branch>`
   - This works whether or not the branch is checked out locally.

3. Add test case in `pkg/git/git_test.go`:
   - Create a test where the branch is already checked out in the main repo
   - Verify CreateWorktree still succeeds
   - Verify the worktree contains the correct code (same commit as the branch)

4. Keep existing tests passing — the detached HEAD approach should be transparent to callers.
</requirements>

<constraints>
- Must use `git` CLI via exec.Command — NOT go-git library
- Must use `git worktree add/remove` — never checkout in the main working tree
- Package name must be `git_test` (external test package)
- Use Ginkgo v2 + Gomega for tests
- Do NOT touch other packages or main.go
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- CreateWorktree succeeds even when branch is checked out in main working tree
- Uses `origin/<branch>` with --detach to avoid branch lock conflict
- New test case covers the "branch already checked out" scenario
- All existing tests still pass
- make precommit passes
</success_criteria>
