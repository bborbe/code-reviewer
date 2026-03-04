---
tags:
  - dark-factory
  - spec
---
Tags: [[Dark Factory - Spec Writing Guide]]

---

## Problem

No tool exists to review pull requests using the local Claude Code CLI with full project context (CLAUDE.md, coding guidelines, subagents, `/code-review` skill). The existing GitHub Actions approach (`claude-code-action`) runs generic reviews without project-specific rules.

## Goal

After completion, a Go CLI tool `pr-reviewer` accepts a single GitHub PR URL, resolves the repo to a local checkout via config, creates a temporary git worktree for the PR branch, runs a Claude Code review in that worktree, and posts the review as a comment on the PR.

## Non-goals

- Bitbucket Server support — spec 003
- Approve/request-changes verdict — spec 002
- Batch/polling mode (review all open PRs)
- Creating or merging PRs
- Reviewing without a local checkout (no cloning)
- Interactive mode / TUI

## Desired Behavior

1. User runs `pr-reviewer <URL>` where URL is a GitHub PR
2. Tool parses the URL to extract owner, repo, and PR number
3. Tool looks up the local repo path from config
4. Tool fetches the PR metadata (source branch) via GitHub API
5. Tool runs `git fetch` in the local repo to ensure branches are up to date
6. Tool creates a git worktree for the PR's source branch
7. Tool runs Claude Code review in the worktree directory — picks up the repo's CLAUDE.md and all project-specific context
8. Tool posts the review output as a PR comment via GitHub API
9. Tool removes the git worktree and cleans up
10. Tool exits 0 on success, non-zero on failure with clear error message

## Constraints

- Config file location: `~/.pr-reviewer.yaml`
- Must use `gh` CLI for GitHub operations (already authenticated)
- Must use `claude` CLI for reviews — not the SDK, not the API directly
- Must use `git worktree add/remove` — never checkout in the main working tree
- Worktree must be inside the repo directory, not in /tmp
- Always clean up worktree on exit (including on error/signal)

## Config Format

```yaml
# ~/.pr-reviewer.yaml
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /Users/bborbe/Documents/workspaces/teamvault-docker
  - url: https://github.com/bborbe/pr-reviewer
    path: /Users/bborbe/Documents/workspaces/pr-reviewer
```

GitHub auth is handled by `gh` CLI — no token needed in config.

## URL Pattern

```
https://github.com/{owner}/{repo}/pull/{number}
```
Extract: owner, repo, PR number.
Repo URL for config lookup: `https://github.com/{owner}/{repo}`

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| No arguments | Exit 1 with usage message | User provides URL |
| Unknown URL format | Exit 1 with "unsupported URL format: <url>" | User fixes URL |
| Config file doesn't exist | Exit 1 with "config not found: ~/.pr-reviewer.yaml" | User creates config |
| Config file invalid YAML | Exit 1 with parse error | User fixes config |
| Repo not in config | Exit 1 with "repo not found in config, add to ~/.pr-reviewer.yaml: <url>" | User adds mapping |
| Local path doesn't exist | Exit 1 with "local path not found: <path>" | User clones repo |
| Local path not a git repo | Exit 1 with "not a git repo: <path>" | User checks path |
| git fetch fails | Exit 1 with git error | User checks network/auth |
| PR branch not found after fetch | Exit 1 with "branch not found: <branch>" | User checks PR is open |
| Worktree already exists (stale) | Remove stale worktree, then create fresh | Automatic |
| Worktree creation fails | Exit 1 with git error | User checks for conflicts |
| claude CLI not found | Exit 1 with "claude not found in PATH" | User installs Claude Code |
| claude exits non-zero | Exit 1 with claude's stderr | User investigates |
| Posting comment fails | Exit 1 with API error, but print review to stdout as fallback | User posts manually |
| Signal (SIGINT/SIGTERM) | Clean up worktree, then exit | Automatic |

## Security / Abuse Cases

- URL parsing: validate against GitHub pattern only, reject anything else
- Worktree path: derived deterministically from repo path + PR number, no user-controlled path components
- Claude output: posted as-is — trusted internal tool, no sanitization needed

## Acceptance Criteria

- [ ] `pr-reviewer https://github.com/bborbe/teamvault-docker/pull/4` produces a review comment on the PR
- [ ] Unknown URL exits 1 with helpful message
- [ ] Missing config file exits 1 with helpful message
- [ ] Missing repo in config exits 1 with helpful message
- [ ] Stale worktree from previous run is cleaned up automatically
- [ ] Worktree is cleaned up after success
- [ ] Worktree is cleaned up after failure
- [ ] Worktree is cleaned up after SIGINT
- [ ] Review is printed to stdout even if posting fails

## Verification

```
make precommit
```

## Do-Nothing Option

Continue using GitHub Actions `claude-code-action` for GitHub. Reviews remain generic without project-specific rules. Acceptable for repos with simple conventions, but inadequate for repos with detailed CLAUDE.md and custom `/code-review` skills.
