---
status: completed
summary: Deduplicated expandHome() function and added missing go:generate directive
container: pr-reviewer-012-code-cleanup
dark-factory-version: v0.14.5
created: "2026-03-04T21:21:55Z"
queued: "2026-03-04T21:21:55Z"
started: "2026-03-04T21:21:55Z"
completed: "2026-03-04T21:25:22Z"
---
<objective>
Fix code duplication and minor housekeeping issues found in code review.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: expandHome() is duplicated in main.go and pkg/config/config.go.
Precondition: pkg/github/github_suite_test.go is missing //go:generate directive.
</context>

<requirements>
1. Deduplicate expandHome():
   - Keep the implementation in pkg/config/config.go (it's already there)
   - Export it as ExpandHome() in pkg/config
   - Update main.go to use config.ExpandHome() instead of its local copy
   - Remove the local expandHome() from main.go

2. Add //go:generate directive to pkg/github/github_suite_test.go:
   - Add `//go:generate go run -mod=mod github.com/maxbrunsfeld/counterfeiter/v6 -generate`
   - Match the pattern used in pkg/git/git_suite_test.go and pkg/review/review_suite_test.go

3. Do NOT touch any other files or add new features.
</requirements>

<constraints>
- Minimal changes only — fix the two issues, nothing else
- Do NOT add doc.go files
- Do NOT change error handling patterns
- Do NOT add tests beyond what's needed
- Use Ginkgo v2 + Gomega for any new tests
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- Only one expandHome (or ExpandHome) implementation exists
- main.go uses config.ExpandHome()
- pkg/github/github_suite_test.go has //go:generate directive
- make precommit passes
</success_criteria>
