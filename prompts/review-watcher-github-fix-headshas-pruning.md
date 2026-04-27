---
status: draft
created: "2026-04-28T00:00:00Z"
---

<summary>
- The HeadSHAs map in cursor state grows unboundedly — one entry per PR ever seen
- Closed or merged PRs are never removed because the search query only returns open PRs
- Over months of operation the cursor.json file and in-memory map grow without limit
- The fix is to rebuild HeadSHAs at the end of each poll cycle from only the current batch
- PRs no longer in the open set are silently pruned from the cursor
- This is safe because re-opening a PR with the same number gets the same task ID and publishes a new CreateTaskCommand
- Tests must verify that closed PRs are removed from the cursor after a poll cycle
</summary>

<objective>
Prevent unbounded growth of `cursor.HeadSHAs` by rebuilding it at the end of each `Poll` cycle to contain only the task IDs seen in the current open-PR batch. Stale entries for closed/merged PRs are discarded naturally on each cycle.
</objective>

<context>
Read `CLAUDE.md` for project conventions.

Files to read before making changes (read ALL first):
- `watcher/github/pkg/watcher.go` (~lines 58-80, 124-155): `Poll`, `processPRs` — where HeadSHAs is read and written
- `watcher/github/pkg/cursor.go` (~lines 20-24): `Cursor` struct with `HeadSHAs map[string]string`
- `watcher/github/pkg/watcher_test.go` (full): existing tests that verify HeadSHAs content after Poll
</context>

<requirements>
1. **In `watcher/github/pkg/watcher.go`**, modify `processPRs` to build a `newHeadSHAs` map and replace `cursorState.HeadSHAs` at the end of the iteration:

   At the start of `processPRs` (after `maxUpdatedAt := since`), initialize:
   ```go
   newHeadSHAs := make(map[string]string, len(allPRs))
   ```

   After each successful PR processing (where `handlePR` returns true or a PR is skipped via `ShouldSkipPR`):
   - Copy the known or newly-fetched head SHA into `newHeadSHAs[taskIDStr] = headSHA`
   - For skipped (filtered) PRs: if the task ID was already in `cursorState.HeadSHAs`, copy it to `newHeadSHAs` to preserve it; if not known, skip (no entry)

   At the end of `processPRs`, before returning `maxUpdatedAt`:
   ```go
   cursorState.HeadSHAs = newHeadSHAs
   ```

   Note: PRs where `fetchHeadSHA` returned an error are NOT added to `newHeadSHAs`. On the next poll cycle those PRs will appear as "new" and trigger a `CreateTaskCommand` — this is the correct behavior (idempotent task creation is handled by the task executor).

2. **Update `watcher/github/pkg/watcher_test.go`** to add a test:
   - First poll: PRs A and B both processed, both in cursor HeadSHAs
   - Second poll: only PR A returned (PR B was closed/merged)
   - Assert: after second poll, cursor HeadSHAs contains only PR A's task ID, not PR B's

3. Run `cd watcher/github && make test` — must pass.

4. Run `cd watcher/github && make precommit` — must exit 0.
</requirements>

<constraints>
- Only change files in `watcher/github/`
- Do NOT commit — dark-factory handles git
- Existing tests must still pass
- The pruning must happen via `newHeadSHAs` rebuild — do NOT mutate `cursorState.HeadSHAs` by deleting keys from the existing map while iterating
- Use `errors.Wrapf(ctx, err, "...")` from `github.com/bborbe/errors` — never `fmt.Errorf`
- Filtered PRs (via `ShouldSkipPR`) that are already known should have their SHA preserved in `newHeadSHAs` to avoid re-publishing them as "new" on the next cycle
</constraints>

<verification>
cd watcher/github && grep -n "newHeadSHAs" pkg/watcher.go
# Expected: declaration, population, and assignment back to cursorState

cd watcher/github && make precommit
</verification>
