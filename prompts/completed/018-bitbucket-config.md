---
status: completed
summary: Added Bitbucket token configuration to pr-reviewer config following GitHub token pattern
container: pr-reviewer-018-bitbucket-config
dark-factory-version: v0.17.0
created: "2026-03-05T12:34:23Z"
queued: "2026-03-05T12:34:23Z"
started: "2026-03-05T12:34:23Z"
completed: "2026-03-05T12:38:04Z"
---
<objective>
Add Bitbucket token configuration to pr-reviewer config, following the same env var pattern as the GitHub token.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Read pkg/config/config.go for the existing Config struct, GitHubConfig, ResolvedGitHubToken, resolveEnvVar, and DefaultGitHubToken.
Read pkg/config/config_test.go for existing test patterns.
Precondition: Config struct has GitHub field with token resolution.
Precondition: resolveEnvVar() handles ${VAR_NAME} pattern.
</context>

<requirements>
1. Add `BitbucketConfig` struct to `pkg/config/config.go`:
   ```go
   type BitbucketConfig struct {
       Token string `yaml:"token"`
   }
   ```

2. Add `Bitbucket BitbucketConfig` field to `Config` struct:
   ```go
   Bitbucket BitbucketConfig `yaml:"bitbucket"`
   ```

3. Add `DefaultBitbucketToken` constant:
   ```go
   // #nosec G101 -- not a credential, just an env var name
   const DefaultBitbucketToken = "BITBUCKET_TOKEN"
   ```

4. Add `ResolvedBitbucketToken()` method on `Config`:
   - If `Bitbucket.Token` is non-empty, resolve via `resolveEnvVar()`
   - Otherwise, fall back to `os.Getenv(DefaultBitbucketToken)`
   - Same pattern as `ResolvedGitHubToken()`

5. Add tests in `pkg/config/config_test.go`:
   - Bitbucket token from config with env var reference
   - Bitbucket token from default env var when not configured
   - Bitbucket token empty when neither configured nor env var set
   - Existing GitHub token tests still pass

6. Config backward-compatibility: YAML without `bitbucket:` section must parse without error (zero-value BitbucketConfig is fine).
</requirements>

<constraints>
- Existing GitHub config must continue to work unchanged
- Config format must remain backward-compatible
- Token must come from environment variable — never hardcoded
- Token must never appear in log output
- Use Ginkgo v2 + Gomega for tests
- Do NOT modify main.go — wiring comes in a later prompt
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- BitbucketConfig with token field added to Config
- ResolvedBitbucketToken works like ResolvedGitHubToken
- Default falls back to BITBUCKET_TOKEN env var
- Backward-compatible with existing configs (no bitbucket section = no error)
- make precommit passes
</success_criteria>
