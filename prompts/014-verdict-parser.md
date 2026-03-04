---
spec: 003-approve-request-changes
---
<objective>
Add a verdict parser that analyzes Claude review output and determines the appropriate GitHub review action (approve, request-changes, or plain comment fallback).
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Read specs/003-approve-request-changes.md for the full spec.
Precondition: Claude review output uses markdown sections: "### Must Fix (Critical)", "### Should Fix (Important)", "### Nice to Have (Optional)".
</context>

<requirements>
1. Create `pkg/verdict/verdict.go`:
   - Define type `Verdict string` with constants: `VerdictApprove`, `VerdictRequestChanges`, `VerdictComment`
   - Define `Result` struct with `Verdict` and `Reason string` fields
   - Function `Parse(reviewText string) Result` that:
     a. Searches for "### Must Fix" or "## Must Fix" section (case-insensitive)
     b. If found AND has content below it (non-empty lines before next heading or EOF) → `VerdictRequestChanges` with reason "must-fix items found"
     c. If "Must Fix" section is empty or says "None" / "none" / "*None*" → `VerdictApprove` with reason "no must-fix items"
     d. If no "Must Fix" section at all but other review sections exist ("Should Fix", "Nice to Have") → `VerdictApprove` with reason "no must-fix section"
     e. If review text doesn't match any expected format → `VerdictComment` with reason "unparseable review format"

2. Create `pkg/verdict/verdict_test.go`:
   - Test: review with "### Must Fix (Critical)\n- Security issue" → `VerdictRequestChanges`
   - Test: review with "### Must Fix (Critical)\n\n*None*" → `VerdictApprove`
   - Test: review with "### Must Fix (Critical)\n\nNone identified." → `VerdictApprove`
   - Test: review with only "### Should Fix" and "### Nice to Have" → `VerdictApprove`
   - Test: review with no recognizable sections → `VerdictComment`
   - Test: empty review text → `VerdictComment`
   - Test: review with "## Must Fix" (h2 instead of h3) → works same as h3

3. The parser must be conservative:
   - When in doubt → `VerdictComment` (safest fallback)
   - Only `VerdictApprove` when clearly no must-fix items
   - Only `VerdictRequestChanges` when clearly has must-fix items
</requirements>

<constraints>
- Pure function, no side effects, no I/O
- Use Ginkgo v2 + Gomega for tests
- No external dependencies (stdlib only for parsing)
- Do NOT touch any other packages
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- pkg/verdict/verdict.go exists with Parse function
- pkg/verdict/verdict_test.go covers all verdict paths
- Conservative: unparseable → VerdictComment, not VerdictApprove
- make precommit passes
</success_criteria>
