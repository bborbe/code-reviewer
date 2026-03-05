---
status: completed
summary: Integrated StripJSONVerdict to remove JSON verdict block from PR comments while preserving full review in stdout
container: pr-reviewer-024-strip-verdict-before-posting
dark-factory-version: v0.17.2
created: "2026-03-05T18:44:41Z"
queued: "2026-03-05T18:44:41Z"
started: "2026-03-05T18:44:41Z"
completed: "2026-03-05T18:49:01Z"
---
<objective>
Use StripJSONVerdict to remove the JSON verdict block from review text before posting it as a PR comment.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read main.go — specifically `runReview()`, `submitGitHubReview()`, and `submitBitbucketReview()`.
Read pkg/verdict/verdict.go for the `StripJSONVerdict` function (added by prompt 023).
</context>

<requirements>
1. In `runReview()` in main.go, the exact ordering must be:
   a. `fmt.Println(reviewText)` — print original to stdout (unchanged, for debugging)
   b. `result := verdict.Parse(reviewText)` — parse verdict from original text
   c. `cleanedText := verdict.StripJSONVerdict(reviewText)` — strip JSON block for posting
   d. Return `cleanedText` and `result` from `runReview()`

2. The cleaned text is used for posting comments (passed to submit functions).
   The original text with JSON verdict is only printed to stdout.

3. Both GitHub and Bitbucket submit functions receive the cleaned text (no changes needed in submit functions themselves).
</requirements>

<constraints>
- stdout output must still show the full review including JSON verdict (for debugging)
- Only the text posted to PR comments is stripped
- If no JSON verdict exists, text is unchanged (StripJSONVerdict is a no-op)
- `--comment-only` flag behavior unchanged
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
