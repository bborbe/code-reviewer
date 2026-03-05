---
tags:
  - dark-factory
  - spec
status: completed
---
Tags: [[Dark Factory - Spec Writing Guide]]

---

## Problem

The current review runs `/code-review` which reviews the entire codebase in the worktree. For PRs, only the diff between the source branch and the target branch matters. Reviewing everything wastes tokens, produces irrelevant findings, and misses the actual changes.

Additionally, `vendor/` directories (Go module vendoring) are included in the review context, which is noise — vendored code is third-party and not authored by the PR submitter.

## Goal

After completion, `pr-reviewer` uses a `/pr-review` command (or inline prompt) that reviews only the PR diff against the target branch, excluding vendor directories.

## Non-goals

- Changing `/code-review` behavior (it stays as-is for full repo reviews)
- Bitbucket Cloud support
- Reviewing individual commits within a PR (only the full diff matters)
- Customizable exclude patterns beyond vendor (future spec)

## Desired Behavior

1. Claude receives a prompt that instructs it to diff the current branch against `origin/<target-branch>` and review only the changes
2. Vendor directories (`vendor/`, `node_modules/`) are excluded from the diff
3. Target branch name is fetched from the PR metadata (GitHub: `baseRefName`, Bitbucket: `toRef.displayId`)
4. The review prompt focuses on: correctness, bugs, security, style — scoped to changed code only
5. Context from unchanged files is used for understanding, but findings only apply to changed lines
6. The `PRInfo` struct carries the target branch name
7. The review command string is constructed dynamically, not hardcoded as `/code-review`

## Constraints

- Must work for both GitHub and Bitbucket PRs
- Must not break `--comment-only` flag or verdict parsing
- Review output format must remain compatible with the verdict parser (spec 003)
- Existing `/code-review` slash command must not be modified
- Target branch defaults to `master` if not determinable
- The review prompt must explicitly instruct Claude to exclude vendor directories from the diff

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| Target branch not in PR metadata | Default to `origin/master` | Automatic |
| `origin/<target>` not fetched | `git fetch` already runs before worktree creation | Automatic |
| Diff is empty (no changes) | Exit 0 with "no changes to review" | Automatic |
| Diff too large for Claude context | Claude truncates, reviews what fits | Acceptable |

## Acceptance Criteria

- [ ] PR review scopes to diff between source and target branch
- [ ] Vendor directories excluded from review
- [ ] Target branch fetched from GitHub API (baseRefName)
- [ ] Target branch fetched from Bitbucket API (toRef.displayId)
- [ ] Default target branch is `master` when not available
- [ ] Verdict parser still works on new review output
- [ ] Existing `--comment-only` flag still works
- [ ] `make precommit` passes

## Verification

```
make precommit
```

## Do-Nothing Option

Keep reviewing the full codebase with `/code-review`. Reviews are noisier and more expensive but still functional. Acceptable for small repos, painful for large monorepos.
