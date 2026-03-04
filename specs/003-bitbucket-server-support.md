---
tags:
  - dark-factory
  - spec
---
Tags: [[Dark Factory - Spec Writing Guide]]

---

## Problem

Spec 001 only supports GitHub. Bitbucket Server PRs at `bitbucket.seibert.tools` have no automated review tooling at all.

## Goal

After completion, `pr-reviewer` also accepts Bitbucket Server PR URLs, resolves the repo to a local checkout, runs the same Claude review workflow, and posts the review as a Bitbucket PR comment.

## Non-goals

- Bitbucket Cloud (different API)
- Approve/reject via Bitbucket API (separate spec if needed)
- Bitbucket-specific review features (inline comments, tasks)

## Desired Behavior

1. URL parsing detects Bitbucket Server pattern alongside existing GitHub pattern
2. Config lookup uses normalized repo URL (same mechanism as GitHub)
3. PR metadata (source branch) fetched via Bitbucket Server API
4. Worktree creation and Claude review run identically to GitHub flow
5. Review posted as a PR comment via Bitbucket Server API
6. Auth via token from environment variable

## Constraints

- Existing GitHub support (spec 001) must continue to work unchanged
- Config format must remain backward-compatible — adding Bitbucket repos must not break existing GitHub-only configs
- Must use Bitbucket Server REST API (not Bitbucket Cloud API)
- Token must come from environment variable — never hardcoded in config
- HTTPS required for all Bitbucket API calls
- Token must never appear in log output

## Config Addition

```yaml
# ~/.pr-reviewer.yaml
repos:
  # Existing GitHub repos still work
  - url: https://github.com/bborbe/teamvault-docker
    path: /Users/bborbe/Documents/workspaces/teamvault-docker
  # New: Bitbucket Server repos
  - url: https://bitbucket.seibert.tools/projects/OC/repos/root
    path: /Users/bborbe/Documents/workspaces/oc-root

bitbucket:
  token: ${BITBUCKET_TOKEN}  # env var reference
```

## URL Pattern

```
https://{host}/projects/{project}/repos/{repo}/pull-requests/{number}/overview
https://{host}/projects/{project}/repos/{repo}/pull-requests/{number}
```
Extract: host, project, repo, PR number.
Repo URL for config lookup: `https://{host}/projects/{project}/repos/{repo}`

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| BITBUCKET_TOKEN env var not set | Exit 1 with "BITBUCKET_TOKEN not set" | User sets env var |
| Token invalid (401) | Exit 1 with "authentication failed for <host>" | User checks token |
| Token lacks permissions (403) | Exit 1 with "insufficient permissions for <host>" | User checks token scope |
| Bitbucket Server unreachable | Exit 1 with connection error | User checks network/VPN |
| PR not found (404) | Exit 1 with "PR not found: <url>" | User checks PR exists |
| PR already merged/declined | Exit 1 with "PR is not open" | User checks PR state |
| API returns unexpected format | Exit 1 with parse error, print raw response | User reports bug |
| Posting comment fails | Exit 1 with API error, but print review to stdout as fallback | User posts manually |
| GitHub PR URL (existing) | Handled by spec 001 path — no change | Automatic |

## Security / Abuse Cases

- Token from env var only — config file must reference `${ENV_VAR}`, never contain raw token
- HTTPS enforced for all API calls — reject HTTP URLs
- Token must not appear in log output or error messages
- Validate Bitbucket host against config — don't send token to arbitrary hosts

## Open Questions

- Token per host or single global token? (Matters if multiple Bitbucket instances)
- Bitbucket Server version minimum for REST API compatibility?

## Acceptance Criteria

- [ ] `pr-reviewer https://bitbucket.seibert.tools/projects/OC/repos/root/pull-requests/254/overview` produces a review comment
- [ ] Existing GitHub PRs still work unchanged
- [ ] Auth failure exits 1 with helpful message
- [ ] Missing BITBUCKET_TOKEN exits 1 with helpful message
- [ ] Worktree is cleaned up after success and failure (same as spec 001)
- [ ] Review is printed to stdout even if posting fails
- [ ] Token does not appear in any output

## Verification

```
make precommit
```

## Do-Nothing Option

No Bitbucket review automation. Manual reviews only. Acceptable if Bitbucket usage is low or team is moving to GitHub.
