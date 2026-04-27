---
status: draft
created: "2026-04-28T00:00:00Z"
---

<summary>
- Project convention requires using libtime.DateTime instead of time.Time in structs and function signatures
- Project convention requires injecting time via libtime.CurrentDateTimeGetter instead of calling time.Now() directly
- The watcher has time.Time in two struct fields, two interface method signatures, two function signatures, and one constructor parameter
- time.Now() is called directly in main.go at startup
- Replacing these types requires updating struct literals, interface, mocks (regenerated), and test fixtures
- The mock for GitHubClient must be regenerated after the interface changes
- Test fixtures using time.Time must be updated to libtime.DateTime
</summary>

<objective>
Replace all `time.Time` usages in structs and function signatures with `libtime.DateTime` from `github.com/bborbe/time`, and replace the `time.Now().UTC()` call in `main.go` with a `libtime.CurrentDateTimeGetter` injection. This follows the project's mandatory time injection pattern and makes time controllable in tests.
</objective>

<context>
Read `CLAUDE.md` for project conventions.
Read `~/.claude/plugins/marketplaces/coding/docs/go-time-injection.md` for the libtime pattern.

Files to read before making changes (read ALL first):
- `watcher/github/pkg/githubclient.go` (~lines 19-53): `PullRequest.UpdatedAt time.Time`, `SearchResult.RateResetAt time.Time`, `SearchPRs` signature with `since time.Time`
- `watcher/github/pkg/cursor.go` (~lines 21-28): `Cursor.LastUpdatedAt time.Time`, `LoadCursor` signature with `startTime time.Time`
- `watcher/github/pkg/watcher.go` (~lines 25-55): `NewWatcher` parameter `startTime time.Time`, `watcher.startTime time.Time`, `fetchAllPRs` with `since time.Time`, `processPRs` returning `time.Time`
- `watcher/github/pkg/factory/factory.go` (~line 53): `startTime time.Time` parameter in `CreateWatcher`
- `watcher/github/main.go` (~line 54): `startTime := time.Now().UTC()`
- `watcher/github/pkg/githubclient_test.go`: test fixtures using `time.Time`
- `watcher/github/pkg/watcher_test.go`: test fixtures using `time.Time` and `startTime`
- `watcher/github/pkg/cursor_test.go`: test fixtures using `time.Time`

Grep-verify libtime symbols before writing:
```bash
grep -rn "type DateTime\|func.*DateTime\|CurrentDateTimeGetter" \
  $(go env GOPATH)/pkg/mod/github.com/bborbe/time@*/... 2>/dev/null | head -20
```
</context>

<requirements>
1. **`watcher/github/pkg/githubclient.go`** — replace `time.Time` with `libtime.DateTime`:
   - `PullRequest.UpdatedAt` (~line 29): `UpdatedAt libtime.DateTime`
   - `SearchResult.RateResetAt` (~line 38): `RateResetAt libtime.DateTime`
   - `GitHubClient.SearchPRs` interface (~line 48): `since libtime.DateTime`
   - `githubClient.SearchPRs` impl (~line 72): `since libtime.DateTime`
   - Inside `SearchPRs` body: update `since.UTC().Format(time.RFC3339)` to use the libtime API (grep for `.Time()` or `.String()` on the DateTime type to confirm the right method)
   - Update the mapping of `issue.GetUpdatedAt().Time` → cast to `libtime.DateTime`
   - Update the mapping of `resp.Rate.Reset.Time` → cast to `libtime.DateTime`
   - In `fetchAllPRs` rate-limit sleep: update `result.RateResetAt` usage (call `.Time()` to get `time.Time` for `time.Until`)
   - Update imports: replace `"time"` with `libtime "github.com/bborbe/time"` (keep `"time"` only if needed for `time.RFC3339` or `time.Until`)

2. **`watcher/github/pkg/cursor.go`** — replace `time.Time` with `libtime.DateTime`:
   - `Cursor.LastUpdatedAt` (~line 22): `LastUpdatedAt libtime.DateTime`
   - `LoadCursor` signature (~line 28): `startTime libtime.DateTime`
   - Inside `LoadCursor`: update `startTime.Format(time.RFC3339)` to use libtime string method
   - Update imports accordingly

3. **`watcher/github/pkg/watcher.go`** — replace `time.Time` with `libtime.DateTime`:
   - `NewWatcher` parameter (~line 29): `startTime libtime.DateTime`
   - `watcher.startTime` field (~line 51): `startTime libtime.DateTime`
   - `fetchAllPRs` parameter (~line 86): `since libtime.DateTime`
   - `processPRs` return type (~line 129): `libtime.DateTime`
   - Inside `processPRs`: update all `time.Time` comparisons (`.After`, assignments) to use `libtime.DateTime`
   - Inside `fetchAllPRs`: the rate-limit sleep uses `time.Until(result.RateResetAt)` — call `.Time()` to get the `time.Time` value for `time.Until`
   - Update `glog` format calls for times: use `.String()` or `.Format()`
   - Update imports

4. **`watcher/github/pkg/factory/factory.go`** — update `CreateWatcher`:
   - `startTime time.Time` parameter (~line 53): `startTime libtime.DateTime`
   - Update imports

5. **`watcher/github/main.go`** — inject `CurrentDateTimeGetter`:
   - Replace `startTime := time.Now().UTC()` (~line 54)
   - Construct `now := libtime.NewCurrentDateTime()` and call `startTime := now.Now()` (or the correct libtime method — grep-verify the API)
   - Update imports

6. **Regenerate mocks** after the interface changes in step 1:
   ```bash
   cd watcher/github && go generate ./pkg/...
   ```

7. **Update test fixtures** in:
   - `watcher/github/pkg/cursor_test.go`: replace `time.Time` literals with `libtime.DateTime`
   - `watcher/github/pkg/githubclient_test.go`: replace `time.Now()` calls and `time.Time` literals
   - `watcher/github/pkg/watcher_test.go`: replace `time.Time` literals (already uses `startTime = time.Date(...)` — convert to libtime equivalent); replace all `time.Now()` in PR fixtures with fixed `libtime.DateTime` values

8. Run `cd watcher/github && make test` — must pass.

9. Run `cd watcher/github && make precommit` — must exit 0.
</requirements>

<constraints>
- Only change files in `watcher/github/`
- Do NOT commit — dark-factory handles git
- Existing tests must still pass
- Use `libtime.DateTime` for all `time.Time` struct fields and function signatures — never `time.Time`
- Use `libtime.CurrentDateTimeGetter` injection in main.go, not `time.Now()` directly
- Grep-verify all libtime symbols before writing (especially `NewCurrentDateTime`, the `Now()` method, and how to convert from `time.Time` to `libtime.DateTime`)
- The mock files in `pkg/mocks/` are regenerated — do NOT hand-edit them
- Use `errors.Wrapf(ctx, err, "...")` from `github.com/bborbe/errors` — never `fmt.Errorf`
</constraints>

<verification>
cd watcher/github && grep -rn "time\.Time\|time\.Now()" pkg/ main.go | grep -v "_test.go" | grep -v "time\.RFC3339\|time\.After\|time\.Until\|time\.Duration\|time\.NewTicker\|time\.Second\|time\.Minute"
# Expected: no matches (all time.Time/time.Now() replaced)

cd watcher/github && make precommit
</verification>
