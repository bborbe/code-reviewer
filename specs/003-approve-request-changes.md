---
tags:
  - dark-factory
  - spec
status: completed
---
Tags: [[Dark Factory - Spec Writing Guide]]

---

## Problem

Spec 001 posts reviews as plain comments. GitHub supports structured review verdicts (approve, request changes) that integrate with branch protection. Currently the reviewer must manually approve/reject after reading the comment.

## Goal

After completion, `pr-reviewer` parses the Claude review output for a pass/fail verdict and submits the review as a structured GitHub review with the appropriate action (approve or request changes) instead of a plain comment.

## Non-goals

- Closing/merging PRs based on verdict
- Configurable thresholds or scoring weights
- Multi-reviewer workflows
- Bitbucket approve/reject (separate spec if needed)
- Changing the `/code-review` output format

## Desired Behavior

The `/code-review` output uses three severity sections:
- `### Must Fix (Critical)` — security vulnerabilities, data correctness, concurrency bugs
- `### Should Fix (Important)` — error handling, architectural violations, missing tests
- `### Nice to Have (Optional)` — style, minor docs, optional optimizations

Verdict logic based on review content:

| Review content | GitHub action |
|---|---|
| "Must Fix" section has items | `request-changes` |
| Only "Should Fix" and/or "Nice to Have" | `approve` (review body contains the comments) |
| No issues found | `approve` |
| Cannot parse review sections | plain comment (spec 001 fallback) |

1. Claude review output is parsed for the presence of "Must Fix" items
2. If "Must Fix" section exists and contains items → `request-changes`
3. If no "Must Fix" items (only "Should Fix" / "Nice to Have" or clean) → `approve`
4. If review output doesn't match expected format → fall back to plain comment (spec 001)
5. User can override verdict via CLI flag (`--comment-only` forces plain comment)
6. Detected verdict and reason are logged to stderr

## Constraints

- Spec 001 plain-comment behavior must remain the default when verdict is unclear
- Must not auto-approve without a clear, unambiguous pass signal from Claude
- Verdict parsing must be conservative — when in doubt, fall back to comment
- `--comment-only` flag must always be available as escape hatch
- Must use `gh` CLI for GitHub review API

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---------|-------------------|----------|
| Claude output has no clear verdict | Fall back to plain comment (spec 001) | Automatic |
| Verdict parser disagrees with review content | Conservative: fall back to comment | User reviews and approves manually |
| GitHub API rejects review submission | Exit 1 with API error, print review to stdout | User posts manually |
| PR already approved by this user | Post new review (GitHub allows updating) | Automatic |
| PR is closed/merged | Exit 1 with "PR is not open" | User checks PR state |
| `--comment-only` combined with other flags | `--comment-only` takes precedence | By design |

## Security / Abuse Cases

- Auto-approving is security-relevant — false approvals could bypass branch protection
- Only approve when "Must Fix" section is empty or absent — conservative by default
- If parsing fails or output is unexpected, always fall back to plain comment (never approve unclear reviews)
- Log the detected verdict to stderr so the user can verify

## Acceptance Criteria

- [ ] Review with "Must Fix" items → `request-changes` via GitHub review API
- [ ] Review with only "Should Fix" / "Nice to Have" → `approve` via GitHub review API
- [ ] Clean review (no issues) → `approve` via GitHub review API
- [ ] Unparseable review output → falls back to plain comment (spec 001 behavior)
- [ ] `--comment-only` flag skips verdict and posts as plain comment
- [ ] Detected verdict and reason logged to stderr
- [ ] `make precommit` passes

## Verification

```
make precommit
```

## Do-Nothing Option

Keep posting plain comments (spec 001). Manual approve/reject after reading. Works fine for single-reviewer workflows where the human always checks the review anyway.
