---
status: completed
summary: Added separate GitHub reviewer identity via GH_TOKEN configuration
container: pr-reviewer-008-separate-reviewer-identity
dark-factory-version: v0.14.5
created: "2026-03-04T19:40:56Z"
queued: "2026-03-04T19:40:56Z"
started: "2026-03-04T19:40:56Z"
completed: "2026-03-04T19:45:59Z"
---
<objective>
Add separate GitHub reviewer identity via GH_TOKEN so pr-reviewer can review PRs created by dark-factory (same user can't self-review).
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config, pkg/github, pkg/review, main.go exist and are tested.
Spec: specs/004-separate-reviewer-identity.md
</context>

<requirements>
1. Update `Config` in `pkg/config/config.go`:
   - Add `GitHubConfig` struct with `Token string \`yaml:"token"\``
   - Add `GitHub GitHubConfig \`yaml:"github"\`` field to `Config`
   - Add `resolveEnvVar(value string) string` — if value matches `${VAR_NAME}` pattern, return `os.Getenv("VAR_NAME")`, otherwise return as-is
   - Add `ResolvedGitHubToken() string` method on `Config` that calls resolveEnvVar
   - Call resolveEnvVar during `Load()` after YAML parsing

2. Add config file permission check in `pkg/config/config.go`:
   - After reading the config file, check file permissions with `os.Stat`
   - If file is world-readable (mode & 0004 != 0), log warning: `"config file is world-readable, consider: chmod 600 ~/.pr-reviewer.yaml"`
   - Warning only — do not fail

3. Update `NewGHClient` in `pkg/github/client.go`:
   - Change signature: `NewGHClient(token string) Client`
   - Store token in `ghClient` struct
   - In `GetPRBranch` and `PostComment`, if token is non-empty: `cmd.Env = append(os.Environ(), "GH_TOKEN="+c.token)`
   - If token is empty: leave cmd.Env nil (inherits parent env)

4. Update `main.go`:
   - Pass `cfg.ResolvedGitHubToken()` to `NewGHClient`
   - If `cfg.GitHub.Token != ""` but `cfg.ResolvedGitHubToken() == ""`: log warning "github.token configured but env var is empty, using default gh auth"

5. Update tests in `pkg/config/config_test.go`:
   - Config without `github` section → `ResolvedGitHubToken()` returns empty string
   - Config with `github.token: ${TEST_VAR}` + env var set → resolves correctly
   - Config with `github.token: ${TEST_VAR}` + env var unset → returns empty string
   - Config with `github.token: literal-value` → returns literal
   - Existing configs without `github` field parse successfully (backward compat)

6. Update tests in `pkg/github/client_test.go`:
   - Verify mock accepts token parameter

7. Regenerate counterfeiter mocks: `go generate ./...`
</requirements>

<constraints>
- Backward compatible — existing configs without `github.token` must keep working
- Token must NEVER appear in log output or error messages
- Token must NEVER be passed as CLI argument (only via cmd.Env)
- Token must NEVER be written to disk
- Use Ginkgo v2 + Gomega for tests
- Use counterfeiter for mocks
</constraints>

<verification>
Run `go generate ./...` -- must succeed.
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- github.token field parsed from config YAML
- ${VAR} env var resolution works
- gh CLI receives GH_TOKEN when configured
- Warning logged for empty env var
- Warning logged for world-readable config file
- Existing configs without github section still work
- Token never appears in logs or error messages
- make precommit passes
</success_criteria>
