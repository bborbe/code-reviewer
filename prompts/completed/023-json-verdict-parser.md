---
status: completed
summary: Added JSON verdict parser with heuristic fallback and strip functionality
container: pr-reviewer-023-json-verdict-parser
dark-factory-version: v0.17.2
created: "2026-03-05T18:37:07Z"
queued: "2026-03-05T18:37:07Z"
started: "2026-03-05T18:37:07Z"
completed: "2026-03-05T18:44:39Z"
---
<objective>
Update the verdict parser to extract a structured JSON verdict block from review output, with fallback to the existing heuristic section scanning.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read pkg/verdict/verdict.go for the current heuristic parser.
Read pkg/verdict/verdict_test.go for existing test coverage.
Read specs/007-sandboxed-review-execution.md for the full spec.
</context>

<requirements>
1. Add a new function `parseJSONVerdict(reviewText string) (Result, bool)` that:
   - Scans the review text for a JSON line matching `{"verdict": "...", "reason": "..."}`
   - The JSON may be inside a markdown code fence (```json ... ```) or on a bare line
   - Returns the parsed Result and true if found, or zero Result and false if not found
   - Valid verdict values: "approve", "request-changes", "comment"
   - Invalid/missing verdict value → return false (fall back to heuristic)

2. Update `Parse(reviewText string) Result` to:
   - First try `parseJSONVerdict(reviewText)`
   - If found, return the JSON result
   - If not found, fall back to existing heuristic section scanning (unchanged)

3. Add a new function `StripJSONVerdict(reviewText string) string` that:
   - Removes the JSON verdict line (and surrounding code fence if present) from the review text
   - Returns the cleaned review text for posting as a PR comment
   - Collapses extra trailing blank lines left after removal
   - If no JSON verdict found, returns the text unchanged

4. Add comprehensive tests:
   - JSON verdict on bare line
   - JSON verdict inside ```json code fence
   - JSON verdict with extra whitespace
   - Invalid JSON → falls back to heuristic
   - Missing verdict field → falls back to heuristic
   - Unknown verdict value → falls back to heuristic
   - No JSON at all → falls back to heuristic (existing behavior preserved)
   - StripJSONVerdict removes the JSON line
   - StripJSONVerdict removes code fence wrapper
   - StripJSONVerdict preserves text when no JSON found
</requirements>

<constraints>
- All existing tests must continue to pass (backward compatibility)
- The heuristic parser must remain unchanged — JSON is tried first, heuristic is fallback
- JSON verdict regex must not match arbitrary JSON in code examples within the review
- The JSON line should be near the end of the review text (last 50 lines) to avoid false matches
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
