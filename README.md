# Code Reviewer

CLI tool to review pull requests using local Claude Code with full project context.

## What it does

`code-reviewer` takes a GitHub or Bitbucket Server PR URL, resolves it to a local repository checkout, creates a git worktree, runs Claude Code review in that worktree (picking up CLAUDE.md and project-specific rules), and posts the review back as a PR comment. This enables context-aware reviews that respect your project's coding guidelines and custom review agents.

## Installation

```bash
go install github.com/bborbe/code-reviewer@latest
```

## Usage

```bash
code-reviewer [-v] <pr-url>
```

Examples:

```bash
# Review a GitHub PR
code-reviewer https://github.com/bborbe/teamvault-docker/pull/4

# Verbose output
code-reviewer -v https://github.com/bborbe/code-reviewer/pull/1
```

## Configuration

Create `~/.code-reviewer.yaml`:

**Minimal configuration:**

```yaml
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: ~/Documents/workspaces/teamvault-docker
  - url: https://github.com/bborbe/code-reviewer
    path: ~/Documents/workspaces/code-reviewer
```

**Full configuration with all options:**

```yaml
github:
  token: ${PR_REVIEWER_GITHUB_TOKEN}  # optional: env var reference (this is also the default)
model: sonnet             # optional: claude model (default: sonnet)
autoApprove: false        # optional: enable auto-approve on clean reviews (default: false)
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: ~/Documents/workspaces/teamvault-docker
    reviewCommand: /code-review  # optional: custom review command (default: /code-review)
  - url: https://github.com/bborbe/code-reviewer
    path: ~/Documents/workspaces/code-reviewer
    reviewCommand: /code-review short
```

**Configuration notes:**

- `github.token`: Optional GitHub token. If not specified or empty after env var resolution, uses `gh` CLI authentication.
- `model`: Claude model to use (e.g., `sonnet`, `opus`, `haiku`). Defaults to `sonnet`.
- `autoApprove`: Enable automatic PR approval when review verdict is "approve". Defaults to `false` (safe default - only posts comments). When `true`, PRs will be auto-approved. Request-changes verdicts are always submitted regardless of this setting.
- `repos[].reviewCommand`: Command passed to Claude Code CLI. Defaults to `/code-review`.

## How it works

1. Parse PR URL to extract owner, repo, and PR number
2. Look up local repository path from config
3. Fetch PR metadata (source branch) via GitHub API
4. Run `git fetch` to update local branches
5. Create temporary git worktree for the PR's source branch
6. Run Claude Code review in the worktree directory (picks up CLAUDE.md and project context)
7. Post review output as a PR comment via GitHub API
8. Remove worktree and clean up

## Requirements

- Go 1.26 or later
- [Claude Code CLI](https://claude.com/claude-code) installed and available in PATH
- [GitHub CLI (`gh`)](https://cli.github.com/) installed and authenticated

## License

BSD 2-Clause License. See [LICENSE](LICENSE) file for details.

<!-- test pr for pr-reviewer agent -->

