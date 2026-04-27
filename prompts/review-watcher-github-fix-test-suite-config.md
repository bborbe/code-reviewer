---
status: draft
created: "2026-04-28T00:00:00Z"
---

<summary>
- pkg/suite_test.go calls RunSpecs without passing suiteConfig or reporterConfig
- This means the pkg test suite has no timeout cap and ignores reporter settings
- The main_test.go in the same repo correctly uses GinkgoConfiguration() + suiteConfig.Timeout = 60s
- Twelve-plus test fixtures use time.Now() making tests non-deterministic
- Fixed dates should be used instead of time.Now() in all PR and test fixtures
- Affected files: pkg/suite_test.go, pkg/watcher_test.go, pkg/githubclient_test.go
</summary>

<objective>
Fix `pkg/suite_test.go` to use `GinkgoConfiguration()` with a 60-second timeout and pass `suiteConfig`/`reporterConfig` to `RunSpecs`, matching the pattern in `main_test.go`. Replace all `time.Now()` calls in test fixtures with fixed `time.Date(...)` values for determinism.
</objective>

<context>
Read `CLAUDE.md` for project conventions.
Read `~/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md` for the Ginkgo suite pattern.

Files to read before making changes (read ALL first):
- `watcher/github/pkg/suite_test.go` (full): current `RunSpecs` call without config (~line 22)
- `watcher/github/main_test.go` (full): the correct `GinkgoConfiguration()` + `suiteConfig.Timeout` pattern to replicate
- `watcher/github/pkg/watcher_test.go`: all `time.Now()` usages in PR fixture construction (~lines 199, 222, 293, 299, 372, 397)
- `watcher/github/pkg/githubclient_test.go`: all `time.Now()` usages (~lines 47, 85, 117, 162, 171, 195, 217, 237)
</context>

<requirements>
1. **Fix `watcher/github/pkg/suite_test.go`** to match `main_test.go`'s suite pattern:

   Change:
   ```go
   func TestSuite(t *testing.T) {
       time.Local = time.UTC
       format.TruncatedDiff = false
       RegisterFailHandler(Fail)
       RunSpecs(t, "Pkg Suite")
   }
   ```
   To:
   ```go
   func TestSuite(t *testing.T) {
       time.Local = time.UTC
       format.TruncatedDiff = false
       RegisterFailHandler(Fail)
       suiteConfig, reporterConfig := GinkgoConfiguration()
       suiteConfig.Timeout = 60 * time.Second
       RunSpecs(t, "Pkg Suite", suiteConfig, reporterConfig)
   }
   ```

2. **Fix `watcher/github/pkg/watcher_test.go`** — replace all `time.Now()` in PR/result fixtures with a fixed date.

   Define a package-level constant near the top of the test file (or in the `BeforeEach` setup block):
   ```go
   var fixedNow = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
   ```

   Replace all occurrences of:
   - `UpdatedAt: time.Now()` → `UpdatedAt: fixedNow`
   - `RateResetAt: time.Now().Add(-1 * time.Second)` → `RateResetAt: fixedNow.Add(-1 * time.Second)`
   (or use `fixedNow` directly since the intent is "already past")

3. **Fix `watcher/github/pkg/githubclient_test.go`** — replace `time.Now()` with fixed values:

   Define a similar fixed reference:
   ```go
   var fixedNow = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
   ```

   Replace:
   - `resetAt := time.Now().Add(time.Hour).Unix()` → `resetAt := fixedNow.Add(time.Hour).Unix()`
   - `time.Now().Add(-24*time.Hour)` (as `since` argument) → `fixedNow.Add(-24*time.Hour)` (or `time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)`)
   - All other `time.Now()` references → equivalent fixed expressions

4. Run `cd watcher/github && make test` — must pass.

5. Run `cd watcher/github && make precommit` — must exit 0.
</requirements>

<constraints>
- Only change files in `watcher/github/`
- Do NOT commit — dark-factory handles git
- Existing tests must still pass with the same assertions — only the time values change
- Do NOT change test logic, only substitute `time.Now()` with fixed values
- Use `errors.Wrapf(ctx, err, "...")` from `github.com/bborbe/errors` — never `fmt.Errorf`
</constraints>

<verification>
cd watcher/github && grep -n "time\.Now()" pkg/suite_test.go pkg/watcher_test.go pkg/githubclient_test.go
# Expected: no matches

cd watcher/github && grep -n "GinkgoConfiguration\|suiteConfig" pkg/suite_test.go
# Expected: two matches (call + usage)

cd watcher/github && make precommit
</verification>
