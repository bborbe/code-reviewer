---
status: completed
summary: Added configurable Claude model with default sonnet and --model flag always passed to claude CLI
container: pr-reviewer-010-configurable-model
dark-factory-version: v0.14.5
created: "2026-03-04T21:01:26Z"
queued: "2026-03-04T21:01:26Z"
started: "2026-03-04T21:01:26Z"
completed: "2026-03-04T21:06:54Z"
---
<objective>
Add configurable Claude model with default sonnet. Always pass --model flag to claude CLI.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config, pkg/review, main.go exist and are tested.
</context>

<requirements>
1. Update `Config` in `pkg/config/config.go`:
   - Add `Model string \`yaml:"model"\`` field to `Config`
   - Add constant `DefaultModel = "sonnet"`
   - Add method `ResolvedModel() string` — returns Model if non-empty, else DefaultModel

2. Update `Reviewer.Review` in `pkg/review/review.go`:
   - Add `model` parameter: `Review(ctx context.Context, worktreePath string, command string, model string) (string, error)`
   - Build claude args: `claude --print --model <model> <command>`

3. Update `main.go`:
   - Pass `cfg.ResolvedModel()` to reviewer.Review

4. Update tests:
   - Config test: no model field → ResolvedModel() returns "sonnet"
   - Config test: model field set → ResolvedModel() returns configured value
   - Review mock: update to match new interface signature

5. Regenerate counterfeiter mocks: `go generate ./...`
</requirements>

<constraints>
- Default model is always sonnet
- --model flag always passed to claude CLI (never omitted)
- Backward compatible — existing configs without model field use default
- Use Ginkgo v2 + Gomega for tests
- Use counterfeiter for mocks
</constraints>

<verification>
Run `go generate ./...` -- must succeed.
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- model field parsed from config YAML
- Default sonnet when not specified
- --model always passed to claude CLI
- Existing configs without model field still work
- make precommit passes
</success_criteria>
