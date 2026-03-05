<objective>
Wire the Bitbucket review verdict (approve/needs-work) into the main workflow. After posting the review comment, submit the appropriate Bitbucket verdict based on the parsed review output — same logic as the existing GitHub path (spec 003).
</objective>

<context>
Read CLAUDE.md for project conventions.
Read specs/005-bitbucket-approve-request-changes.md for the full spec.
Read main.go — specifically the `submitBitbucketReview()` function which currently posts comments only.
Read pkg/bitbucket/client.go for the Bitbucket client interface (should already have Approve, NeedsWork, GetProfile methods from the previous prompt).
Read pkg/verdict/verdict.go for the verdict parser.
</context>

<requirements>
1. Update `submitBitbucketReview()` in main.go to submit verdicts after posting the comment:
   - `VerdictApprove` → call `bbClient.Approve(ctx, host, project, repo, number)`
   - `VerdictRequestChanges` → call `bbClient.GetProfile(ctx, host)` to get user slug, then `bbClient.NeedsWork(ctx, host, project, repo, number, userSlug)`
   - `VerdictComment` → no verdict action, comment already posted (current behavior)

2. `--comment-only` flag must skip verdict submission (post comment only, same as current behavior)

3. Log verdict action to stderr using `logAlways()`:
   - "approving PR..." before Approve call
   - "marking PR as needs-work..." before NeedsWork call

4. If verdict submission fails after comment was already posted:
   - Log error to stderr
   - Exit 1 with error (comment is already posted as fallback)

5. Update existing tests in main_test.go (if any) or add integration test coverage

6. Remove the `_ = commentOnly` and `_ = result` placeholder lines in submitBitbucketReview
</requirements>

<constraints>
- Only Bitbucket Server/Data Center REST API — no Bitbucket Cloud
- Existing GitHub verdict flow in submitGitHubReview must not change
- `--comment-only` flag applies to both GitHub and Bitbucket
- Comment is always posted first (spec 004 behavior preserved)
- Conservative parsing: when verdict is unclear, fall back to comment only
</constraints>

<verification>
Run `make precommit` -- must pass.
</verification>
