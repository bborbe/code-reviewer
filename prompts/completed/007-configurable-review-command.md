---
status: completed
summary: Made Claude review command configurable per repo with default /code-review
container: pr-reviewer-007-configurable-review-command
dark-factory-version: v0.14.5
created: "2026-03-04T19:13:19Z"
queued: "2026-03-04T19:13:19Z"
started: "2026-03-04T19:13:19Z"
completed: "2026-03-04T19:19:26Z"
---
<objective>
Make the Claude review command configurable per repo in the config file. Default to "/code-review" if not specified.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config, pkg/review exist and are tested.
Currently hardcoded: `claude --print "/code-review"` in pkg/review/review.go.
</context>

<requirements>
1. Update `RepoConfig` in `pkg/config/config.go`:
   - Add `ReviewCommand string \`yaml:"reviewCommand"\`` field
   - Default to `/code-review` when empty

2. Update `Config.FindRepoPath` or add a new method to return both path and review command:
   - Option: change `FindRepoPath` to `FindRepo` returning a struct with Path and ReviewCommand
   - Or add `FindReviewCommand(repoURL string) string` method
   - Whichever is cleaner — prefer the struct approach to avoid double lookup

3. Update `Reviewer` interface in `pkg/review/review.go`:
   - `Review(ctx context.Context, worktreePath string, command string) (string, error)`
   - The command parameter replaces the hardcoded "/code-review"
   - `claude --print "<command>"` where command comes from config

4. Update `main.go` to pass the review command from config to the reviewer.

5. Update tests:
   - Config test: verify reviewCommand is parsed from YAML
   - Config test: verify default "/code-review" when reviewCommand is empty
   - Review mock: update to match new interface signature

6. Regenerate counterfeiter mocks after interface changes:
   - Run `go generate ./...`
</requirements>

<constraints>
- Default must be "/code-review" — existing configs without reviewCommand must keep working
- Must not break existing config files (backward compatible)
- Use Ginkgo v2 + Gomega for tests
- Use counterfeiter for mocks
</constraints>

<verification>
Run `go generate ./...` -- must succeed.
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- reviewCommand field in config YAML parsed correctly
- Default "/code-review" when not specified
- Claude invoked with configurable command
- Existing configs without reviewCommand still work
- make precommit passes
</success_criteria>
