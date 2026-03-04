---
tags:
  - dark-factory
  - spec
---
Tags: [[Dark Factory - Spec Writing Guide]]

---

## Problem

Spec 001 posts reviews as plain comments. GitHub supports structured review verdicts (approve, request changes) that integrate with branch protection. Currently the reviewer must manually approve/reject after reading the comment.

## Goal

After completion, `pr-reviewer` parses the Claude review output for a pass/fail verdict and submits the review as a structured GitHub review with the appropriate action (approve or request changes) instead of a plain comment.

## Non-goals

- Closing/merging PRs based on verdict
- Custom verdict thresholds or scoring
- Multi-reviewer workflows
- Bitbucket approve/reject (separate spec if needed)

## Desired Behavior

1. Claude review output is analyzed for a verdict (pass / fail / unclear)
2. On pass: tool submits a GitHub review with approve action
3. On fail: tool submits a GitHub review with request-changes action
4. On unclear verdict: tool falls back to plain comment (spec 001 behavior)
5. User can override verdict via CLI flag (`--comment-only` forces plain comment)

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
- Verdict parsing must be conservative: only approve on explicit, unambiguous pass signal
- Never approve if Claude output contains any error/warning indicators alongside a pass signal
- Log the detected verdict to stdout so the user can verify

## Open Questions

- How does `/code-review` signal pass/fail? Structured output? Keywords? Exit code?
- Should verdict keywords be configurable or hardcoded?

## Acceptance Criteria

- [ ] Clean review → PR approved via GitHub review API
- [ ] Review with issues → PR gets request-changes via GitHub review API
- [ ] Ambiguous verdict → falls back to plain comment (spec 001 behavior)
- [ ] `--comment-only` flag skips verdict and posts as plain comment
- [ ] Detected verdict is logged to stdout
- [ ] Existing spec 001 behavior unchanged when no verdict flag is used

## Verification

```
make precommit
```

## Do-Nothing Option

Keep posting plain comments (spec 001). Manual approve/reject after reading. Works fine for single-reviewer workflows where the human always checks the review anyway.
