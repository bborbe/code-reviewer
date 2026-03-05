---
status: approved
---
<objective>
Add configurable auto-approve behavior. By default, pr-reviewer should only post comments and mark needs-work on issues — but NOT auto-approve. Auto-approve should be opt-in via config.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read main.go — specifically `submitGitHubReview()` and `submitBitbucketReview()`.
Read pkg/config/config.go for the Config struct.
Read pkg/verdict/verdict.go for the verdict types.
</context>

<requirements>
1. Add `AutoApprove bool` field to top-level Config struct (yaml: `autoApprove`), default false.

2. Pass `autoApprove` to both `submitGitHubReview()` and `submitBitbucketReview()`.

3. When `autoApprove` is false (default):
   - `VerdictApprove` → post comment only, do NOT call Approve API. Log: "skipping auto-approve (disabled in config)"
   - `VerdictRequestChanges` → post comment AND mark needs-work (unchanged)
   - `VerdictComment` → post comment only (unchanged)

4. When `autoApprove` is true:
   - `VerdictApprove` → post comment AND call Approve API (current behavior)
   - `VerdictRequestChanges` → post comment AND mark needs-work (unchanged)
   - `VerdictComment` → post comment only (unchanged)

5. GitHub flow: same logic — when autoApprove=false and verdict=approve, use PostComment instead of SubmitReview with --approve.

6. Add tests for the new behavior (autoApprove true vs false paths).

7. Update README.md config example to document autoApprove field.
</requirements>

<constraints>
- Default must be false (safe default — never auto-approve unless explicitly configured)
- `--comment-only` flag still overrides everything (skips all verdicts)
- Needs-work/request-changes is NOT affected by autoApprove (always submitted when issues found)
- No per-repo override needed — global config only
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
