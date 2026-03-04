---
spec: 003-approve-request-changes
---
<objective>
Add SubmitReview method to the GitHub Client interface that submits structured reviews (approve / request-changes) via gh CLI, alongside the existing PostComment method.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Read pkg/github/client.go for the existing Client interface and ghClient implementation.
Read pkg/verdict/verdict.go for the Verdict type (created in prompt 014).
Precondition: Client interface has GetPRBranch and PostComment methods.
Precondition: pkg/verdict defines VerdictApprove and VerdictRequestChanges constants.
</context>

<requirements>
1. Add `SubmitReview` to the `Client` interface in `pkg/github/client.go`:
   ```go
   SubmitReview(ctx context.Context, owner, repo string, number int, body string, verdict verdict.Verdict) error
   ```

2. Implement `SubmitReview` on `ghClient`:
   - If verdict is `VerdictApprove`: run `gh pr review <number> --repo <owner/repo> --approve --body <body>`
   - If verdict is `VerdictRequestChanges`: run `gh pr review <number> --repo <owner/repo> --request-changes --body <body>`
   - If verdict is `VerdictComment` or any other value: return error (caller should use PostComment instead)
   - Set `GH_TOKEN` in env if configured (same pattern as existing methods)

3. Run `go generate ./...` to regenerate the Counterfeiter mock for Client (it now has 3 methods).

4. Add tests in `pkg/github/client_test.go`:
   - Test: SubmitReview with VerdictComment returns error
   - Other paths hard to test without real gh CLI — document why in a comment

5. Do NOT change PostComment — it stays for the fallback case.
</requirements>

<constraints>
- Follow existing ghClient patterns exactly (GH_TOKEN, nosec comment, stderr buffer)
- Import verdict package: `github.com/bborbe/pr-reviewer/pkg/verdict`
- Use Ginkgo v2 + Gomega for tests
- Run `go generate ./...` after interface change
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- Client interface has SubmitReview method
- ghClient implements approve and request-changes via gh CLI
- Mock regenerated with SubmitReview
- make precommit passes
</success_criteria>
