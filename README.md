# pr-reviewer

CLI tool to review pull requests using local Claude Code with full project context.

## What it does

`pr-reviewer` takes a GitHub or Bitbucket Server PR URL, resolves it to a local repository checkout, creates a git worktree, runs Claude Code review in that worktree (picking up CLAUDE.md and project-specific rules), and posts the review back as a PR comment. This enables context-aware reviews that respect your project's coding guidelines and custom review agents.

## Installation

```bash
go install github.com/bborbe/pr-reviewer@latest
```

## Usage

```bash
pr-reviewer [-v] <pr-url>
```

Examples:

```bash
# Review a GitHub PR
pr-reviewer https://github.com/bborbe/teamvault-docker/pull/4

# Verbose output
pr-reviewer -v https://github.com/bborbe/pr-reviewer/pull/1
```

## Configuration

Create `~/.pr-reviewer.yaml`:

**Minimal configuration:**

```yaml
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: ~/Documents/workspaces/teamvault-docker
  - url: https://github.com/bborbe/pr-reviewer
    path: ~/Documents/workspaces/pr-reviewer
```

**Full configuration with all options:**

```yaml
github:
  token: ${GITHUB_TOKEN}  # optional: use env var reference or literal token
model: sonnet             # optional: claude model (default: sonnet)
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: ~/Documents/workspaces/teamvault-docker
    reviewCommand: /code-review  # optional: custom review command (default: /code-review)
  - url: https://github.com/bborbe/pr-reviewer
    path: ~/Documents/workspaces/pr-reviewer
    reviewCommand: /code-review short
```

**Configuration notes:**

- `github.token`: Optional GitHub token. If not specified or empty after env var resolution, uses `gh` CLI authentication.
- `model`: Claude model to use (e.g., `sonnet`, `opus`, `haiku`). Defaults to `sonnet`.
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

BSD 3-Clause License. See [LICENSE](LICENSE) file for details.
