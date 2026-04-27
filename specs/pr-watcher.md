---
status: draft
---

## Summary

- A new long-running service polls GitHub on a configurable interval and creates a vault task per new pull request in scope, so the existing PR-reviewer agent gets triggered automatically without human task creation
- Default repo scope is `bborbe/*` (configurable via env); default poll interval is 5 minutes (configurable)
- Filters out drafts and bot-authored PRs by default; an allowlist of bot user names is env-configurable
- Re-triggers a review when an open PR receives new commits (`synchronize`-style event), marking the prior task body as outdated
- Vault is the dedup source-of-truth: before creating a task, the watcher checks for an existing task with the same `pr_url` to avoid duplicates

## Problem

The PR-reviewer agent at `agent/pr-reviewer/` works mechanically (v0.14.2 deployed to dev) but only runs when a human manually creates a vault task pointing at a PR. For the agent to deliver daily value across all `bborbe/*` repos as scoped in [GitHub PR Reviewer](../../../Obsidian/Personal/23%20Goals/GitHub%20PR%20Reviewer.md), trigger has to be automatic. Without auto-trigger, the agent's existence is invisible to anyone who doesn't already know it exists.

## Goal

After completion, opening (or pushing to) a pull request in any in-scope `bborbe/*` repo causes the watcher to detect it within one poll interval, write a vault task with the PR URL and metadata into the existing OpenClaw task pipeline, and push it via `obsidian-git`. The existing pr-reviewer agent then picks the task up and reviews. No human action required between PR open and review running.

## Non-goals

- Webhook-based triggering — polling is the MVP; webhook is a possible future upgrade
- Bitbucket support — separate effort, separate spec, lives in [PR Reviewer Ideas](../../../Obsidian/Personal/23%20Goals/PR%20Reviewer%20Ideas.md)
- Posting verdicts back to the PR — the agent's job (separate spec, [GitHub PR Reviewer] Task A4)
- Reviewing the same PR twice on identical commits — explicit dedup
- Multi-tenant operation — one watcher pod per environment, single-pod by design
- Reviewing closed/merged PRs — only `state:open` PRs trigger
- Per-PR filter rules (labels, branches, assignees) — defer to v2 once we observe spam patterns

## Desired Behavior

1. The service runs as a persistent process, polling GitHub on a configurable interval (default: 5 minutes)
2. Each poll, the watcher fetches all open PRs in the configured repo scope (default: `bborbe/*`) updated since the last poll cursor
3. On cold start (no cursor), the cursor is set to service-start-time — no historical PRs are picked up
4. For each candidate PR, the watcher applies filters: skip drafts; skip PRs authored by users in the configured bot allowlist (default: `dependabot[bot]`, `renovate[bot]`)
5. For each PR that passes filters, the watcher checks the vault's task directory for an existing task whose body or frontmatter references the same `pr_url`. If present and the head SHA is unchanged, skip
6. If the PR has no existing task: create a new task file with the PR URL + metadata, commit + push via the existing `obsidian-git` flow, so the OpenClaw controller picks it up
7. If the PR has an existing task but a new head SHA (force-push or new commits): update the existing task by prepending an `## Outdated by force-push <old-sha>` block to the body, then re-trigger by resetting `phase: planning` + `status: in_progress` + `trigger_count: 0`
8. After a successful poll, the cursor is persisted to a PVC-backed file so restarts resume from the last seen timestamp
9. All filter, scope, interval, and allowlist values come from env vars — no recompile to change behavior

## Constraints

- Lives at `agent/pr-watcher/` in `bborbe/code-reviewer` — new Go module with its own `go.mod`, mirroring the multi-module layout used by `agent/pr-reviewer/`
- Authenticates to GitHub via a `GH_TOKEN` env (separate teamvault entry from the agent — watcher only needs `repo` read scope)
- Writes vault task files under the OpenClaw repo's `tasks/` directory using a checked-out clone, then commits + pushes via `git`
- Vault task schema must match what the existing pr-reviewer agent already consumes — same frontmatter fields (`assignee: pr-reviewer-agent`, `phase: planning`, `status: in_progress`, `stage: <env>`, `task_identifier: <uuid>`, `title: <PR title>`) plus a body containing the PR URL
- One pod per environment (single-replica K8s Deployment); no concurrent-watcher locking required
- Cursor stored at `/data/cursor.json` (PVC-backed) — survives pod restarts; corrupt or missing cursor falls back to service-start-time
- All HTTP calls to GitHub use the official `gh` CLI invoked as a subprocess (consistent with pr-reviewer's pattern, no separate REST client needed) — pass args separately, never via shell interpolation
- Errors wrapped via `github.com/bborbe/errors` (per `docs/dod.md`); structured logging via `glog`
- Tests use Ginkgo/Gomega + Counterfeiter for mocking the gh-subprocess boundary and the vault-write boundary (specific interface names are a prompt-level decision)
- Configurable env vars: `POLL_INTERVAL` (default `5m`), `REPO_SCOPE` (default `bborbe`), `BOT_ALLOWLIST` (comma-separated, default `dependabot[bot],renovate[bot]`), `VAULT_REPO_PATH` (no default — must be set), `VAULT_TASK_DIR` (default `tasks`), `STAGE` (no default — must be explicitly set per environment)
- Startup validation: fail-fast (process exits non-zero before any poll) if `VAULT_REPO_PATH`, `STAGE`, or `GH_TOKEN` are unset. Defaults exist only for genuinely-optional behavior knobs (interval, scope, allowlist, task dir) — never for environment-routing fields where a silent default could write to the wrong place
- Stale-task body marker MUST follow the exact format `## Outdated by force-push <old-sha>` so future tooling can grep for it
- A poll cycle that fails entirely (GitHub unreachable, auth fails) must NOT crash the process — log error, retain cursor, retry next interval

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| Empty result from GitHub query (no PRs) | Cursor advances to `now`, no tasks created | n/a |
| GitHub API returns non-zero on `gh search prs` | Logged error, cursor unchanged, retry next poll | Automatic |
| `GH_TOKEN` unauthorized (HTTP 401 from `gh`) | Logged error with rotation guidance, cursor unchanged, retry next poll | Operator rotates teamvault entry |
| GitHub rate-limit hit (HTTP 429 / 403 with rate-limit headers) | Logged warning, cursor unchanged, retry next poll | Automatic next interval |
| Vault git push fails (network, auth, conflict) | Task file unstaged from this attempt, cursor not advanced past this PR, retry next poll | Automatic; persistent failure surfaces in logs |
| Vault git pull conflicts with local state | Logged error, no tasks created this cycle, retry next poll | Operator resolves vault state |
| Cursor file corrupt / unparseable | Treat as cold start (service-start-time), log warning | Automatic |
| Cursor file missing | Treat as cold start, log info | Automatic (first boot) |
| PR with no existing task and creation fails mid-flow | No partial commit pushed; cursor not advanced past this PR | Retry next interval |
| Duplicate detection finds matching task | Skip task creation, log debug | n/a |
| Bot author matches allowlist | Skip PR, log debug with author + reason | n/a |
| Draft PR | Skip PR, log debug with reason | n/a |
| Force-push detected (existing task, new head SHA) | Update existing task body with outdated marker + reset frontmatter for re-review | Automatic |
| Filesystem write to PVC fails (disk full, permissions) | Log error, do not commit cursor, retry next poll | Operator investigates PVC |
| Context cancellation (pod shutdown signal) | Finish current poll cycle, persist cursor, exit cleanly | n/a |

## Security / Abuse Cases

- `GH_TOKEN` only needs `repo` read scope — minimum privilege; verify scope at startup (similar to pr-reviewer's preflight pattern)
- All `gh` CLI invocations pass args as separate exec arguments, never via shell interpolation — repo names and PR URLs come from GitHub API, but shell-out hygiene matters
- Vault writes are limited to the configured `VAULT_TASK_DIR` only — no path traversal from PR titles or branch names that could write outside `tasks/`
- Task body content sanitizes PR title / description for any markdown injection that could break the task file's frontmatter parsing
- The PR-reviewer agent eventually consumes these task bodies and feeds them to Claude — if a PR title contains prompt-injection content, the agent inherits that risk. Document this constraint; do not attempt content-level sanitization in the watcher (premature and ineffective)

## Acceptance Criteria

- [ ] Service exists at `agent/pr-watcher/` with its own `go.mod`, `Dockerfile`, `Makefile`, and `k8s/` manifests
- [ ] `main.go` reads env vars, starts the polling loop, handles SIGTERM gracefully
- [ ] Filters (drafts, bot allowlist) and dedup (vault task scan by `pr_url`) implemented and unit-tested
- [ ] Re-trigger on new head SHA writes the `## Outdated by force-push <old-sha>` marker and resets frontmatter
- [ ] Counterfeiter fakes at the gh-subprocess boundary and the vault-write boundary (interface names are a prompt decision)
- [ ] Unit tests cover all rows of the Failure Modes table
- [ ] `make precommit` passes for `agent/pr-watcher/`
- [ ] Released as `code-reviewer v0.16.0`
- [ ] **E2E scenario** (HTTP integration seam to GitHub + filesystem integration seam to vault repo — required per spec-writing.md): deploy v0.16.0 to dev; create a test PR in `bborbe/code-reviewer`; observe within 5 minutes that (a) a vault task appears under `tasks/` with `assignee: pr-reviewer-agent`, (b) the existing pr-reviewer agent picks it up and runs the 3-phase pipeline, (c) verdict is posted to the task body, (d) closing the test PR does not create new tasks, (e) opening a draft PR is skipped, (f) opening a PR authored by a bot in the allowlist is skipped — confirms the polling, filtering, dedup, vault-write, and downstream-trigger seams all work end-to-end

## Verification

```sh
cd agent/pr-watcher && make precommit
```

Manual verification in dev (after deploy):

```sh
# Watcher running
kubectlquant -n dev get pods -l app=pr-watcher

# Logs show poll cycles
kubectlquant -n dev logs deploy/pr-watcher --tail=50

# Cursor persisted
kubectlquant -n dev exec deploy/pr-watcher -- cat /data/cursor.json
```

Then trigger a real PR (open + push + draft + bot-authored) in `bborbe/code-reviewer` and observe the vault for task creation per the E2E acceptance criterion.

## Do-Nothing Option

Stay manual: continue creating vault tasks by hand per PR. This is what blocks [GitHub PR Reviewer](../../../Obsidian/Personal/23%20Goals/GitHub%20PR%20Reviewer.md) from delivering its value — the agent works mechanically but daily-life invisible. The MVP cannot complete without auto-trigger; this spec is on the critical path.

A weaker alternative — a cronjob inside the existing pr-reviewer pod that polls + creates tasks — couples concerns (watcher and reviewer) into one binary, makes scaling harder, and conflates the producer/consumer roles the [Agent Task Controller Architecture](../../../Obsidian/Personal/50%20Knowledge%20Base/Agent%20Task%20Controller%20Architecture.md) is built around.
