---
status: draft
created: "2026-04-28T00:00:00Z"
---

<summary>
- ParseBotAllowlist contains a for loop and conditionals — business logic that must not live in a factory file
- The factory guide requires zero logic in factory functions
- Moving it to pkg/filter.go puts it alongside the related ShouldSkipPR and IsBotAuthor functions
- The syncProducer.Close() error is silently discarded with "_ = err" — should be logged as a warning
- The factory package has no tests at all despite containing testable logic
- A factory_suite_test.go and factory_test.go are needed to cover ParseBotAllowlist after it moves
- The pollInterval parameter in CreateWatcher is accepted but never used inside the factory — remove it
</summary>

<objective>
Move `ParseBotAllowlist` from `pkg/factory/factory.go` to `pkg/filter.go`, log the `syncProducer.Close()` error instead of discarding it, remove the unused `pollInterval` parameter from `CreateWatcher`, and add a test suite with coverage for `ParseBotAllowlist` in its new location.
</objective>

<context>
Read `CLAUDE.md` for project conventions.
Read `~/.claude/plugins/marketplaces/coding/docs/go-factory-pattern.md` for the zero-logic factory rule.

Files to read before making changes (read ALL first):
- `watcher/github/pkg/factory/factory.go` (full): `ParseBotAllowlist` (~lines 76-89), `syncProducer.Close()` cleanup (~lines 36-39), `pollInterval` parameter (~line 52)
- `watcher/github/pkg/filter.go` (full): `ShouldSkipPR` and `IsBotAuthor` — `ParseBotAllowlist` will be added here
- `watcher/github/pkg/filter_test.go` (full): existing tests to understand the test pattern for this package
- `watcher/github/pkg/suite_test.go`: Ginkgo suite file pattern to replicate for the factory package
- `watcher/github/main.go` (~line 53): `factory.ParseBotAllowlist` call site — must be updated to `pkg.ParseBotAllowlist`
</context>

<requirements>
1. **Move `ParseBotAllowlist` from `pkg/factory/factory.go` to `watcher/github/pkg/filter.go`**:
   - Append the function to `filter.go` (after `ShouldSkipPR`)
   - Remove `ParseBotAllowlist` and its `strings` import from `factory.go`
   - Remove the `strings` import from `factory.go` if it is only used by `ParseBotAllowlist`

2. **Update `watcher/github/main.go`** (~line 53):
   - Change `factory.ParseBotAllowlist(a.BotAllowlist)` → `pkg.ParseBotAllowlist(a.BotAllowlist)`
   - Verify `pkg` is already imported; if not, add the import

3. **Fix `syncProducer.Close()` error handling in `pkg/factory/factory.go`** (~lines 36-39):
   - Replace `_ = err` with `glog.Warningf("close kafka sync producer: %v", err)`
   - Add `"github.com/golang/glog"` to imports if not already present

4. **Remove unused `pollInterval time.Duration` parameter from `CreateWatcher`** (~line 52 of factory.go):
   - Remove the parameter from the function signature
   - Remove it from the call site in `main.go` (~line 63 — `pollInterval` arg passed to `CreateWatcher`)
   - Remove the `"time"` import from `factory.go` if it is only used for this parameter

5. **Create `watcher/github/pkg/factory/factory_suite_test.go`** following the exact pattern of `watcher/github/pkg/suite_test.go`:
   ```go
   // Copyright (c) 2026 Benjamin Borbe All rights reserved.
   // Use of this source code is governed by a BSD-style
   // license that can be found in the LICENSE file.

   package factory_test

   import (
       "testing"
       "time"

       . "github.com/onsi/ginkgo/v2"
       . "github.com/onsi/gomega"
       "github.com/onsi/gomega/format"
   )

   func TestSuite(t *testing.T) {
       time.Local = time.UTC
       format.TruncatedDiff = false
       RegisterFailHandler(Fail)
       suiteConfig, reporterConfig := GinkgoConfiguration()
       suiteConfig.Timeout = 60 * time.Second
       RunSpecs(t, "Factory Suite", suiteConfig, reporterConfig)
   }
   ```

6. **Create `watcher/github/pkg/factory/factory_test.go`** with tests for `ParseBotAllowlist` (now in `pkg/`, not `factory/`). Since `ParseBotAllowlist` has moved to `pkg/`, these tests should be in `pkg/filter_test.go` instead. Add them there:

   In `watcher/github/pkg/filter_test.go`, add a `Describe("ParseBotAllowlist")` block covering:
   - Empty string input → returns nil
   - Single entry → returns `[]string{"entry"}`
   - Multiple comma-separated entries → returns slice of all
   - Entries with leading/trailing whitespace → trimmed
   - Entries that are only whitespace after trimming → filtered out
   - Input with trailing comma → trailing empty entry filtered

7. Run `cd watcher/github && make test` — must pass.

8. Run `cd watcher/github && make precommit` — must exit 0.
</requirements>

<constraints>
- Only change files in `watcher/github/`
- Do NOT commit — dark-factory handles git
- Existing tests must still pass
- `factory.go` must contain zero loops, conditionals, or business logic after the change
- `ParseBotAllowlist` is moved to `pkg/filter.go` — it is NOT deleted
- Use `errors.Wrapf(ctx, err, "...")` from `github.com/bborbe/errors` — never `fmt.Errorf`
</constraints>

<verification>
cd watcher/github && grep -n "ParseBotAllowlist\|for.*parts\|strings.Split" pkg/factory/factory.go
# Expected: no matches (function moved out)

cd watcher/github && grep -n "func ParseBotAllowlist" pkg/filter.go
# Expected: one match

cd watcher/github && grep -n "pkg\.ParseBotAllowlist" main.go
# Expected: one match

cd watcher/github && grep -n "_ = err" pkg/factory/factory.go
# Expected: no matches (replaced with glog.Warningf)

cd watcher/github && grep -n "pollInterval" pkg/factory/factory.go
# Expected: no matches (parameter removed)

cd watcher/github && make precommit
</verification>
