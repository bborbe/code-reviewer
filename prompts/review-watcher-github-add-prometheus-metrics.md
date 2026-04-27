---
status: draft
created: "2026-04-28T00:00:00Z"
---

<summary>
- The watcher exposes a /metrics endpoint but has zero application-level metrics
- Poll cycles, PR publish counts, errors, and rate-limit aborts are completely invisible to operators
- The project requires a Metrics interface with counterfeiter annotation for testability
- Metrics must be registered in init() via prometheus.MustRegister on package-level vars
- Known label values must be pre-initialized with .Add(0) / .Inc(0) so rate() works from first scrape
- The Watcher struct must accept a Metrics dependency via injection
- The factory must wire the real Prometheus implementation
- Tests for the watcher must use the counterfeiter fake
</summary>

<objective>
Add application-level Prometheus metrics to the `watcher/github` service. The end state is a `Metrics` interface in `pkg/`, a real Prometheus implementation registered in `init()`, a counterfeiter mock, and the `Watcher` accepting `Metrics` via injection so poll results, PR publish counts, and rate-limit events are observable.
</objective>

<context>
Read `CLAUDE.md` for project conventions.
Read `~/.claude/plugins/marketplaces/coding/docs/go-prometheus-metrics-guide.md` for the required metrics pattern (interface, init() registration, pre-initialization, counterfeiter annotation).

Files to read before making changes (read ALL first):
- `watcher/github/pkg/watcher.go` (full): `NewWatcher` constructor and all paths to instrument
- `watcher/github/pkg/factory/factory.go` (full): `CreateWatcher` to wire the new dependency
- `watcher/github/main.go` (full): `Run` method to verify no additional wiring needed
- `watcher/github/pkg/suite_test.go`: test suite file for the mock //go:generate directive location
- `watcher/github/pkg/watcher_test.go` (full): existing tests to understand what `It` blocks exist and where to add metric assertions

Grep-verify Prometheus symbols:
```bash
grep -rn "NewCounterVec\|NewGaugeVec\|MustRegister" \
  $(go env GOPATH)/pkg/mod/github.com/prometheus/client_golang@*/prometheus/ 2>/dev/null | head -10
```
</context>

<requirements>
1. **Create `watcher/github/pkg/metrics.go`**:

   Define the `Metrics` interface with counterfeiter annotation:
   ```go
   //counterfeiter:generate -o mocks/metrics.go --fake-name Metrics . Metrics
   type Metrics interface {
       // IncPollCycle increments the poll cycle counter with the given result label.
       // result: "success", "rate_limited", "github_error"
       IncPollCycle(result string)
       // IncPRPublished increments the PR-published counter with the given command label.
       // command: "create", "update_frontmatter", "skipped", "error"
       IncPRPublished(command string)
   }
   ```

   Implement the real Prometheus metrics:
   ```go
   var (
       pollCyclesTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
           Name: "github_pr_watcher_poll_cycles_total",
           Help: "Total number of GitHub poll cycles by result.",
       }, []string{"result"})

       prPublishedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
           Name: "github_pr_watcher_prs_total",
           Help: "Total number of PRs processed by command type.",
       }, []string{"command"})
   )

   func init() {
       prometheus.MustRegister(pollCyclesTotal, prPublishedTotal)
       // Pre-initialize all known label values so rate() works from first scrape
       for _, result := range []string{"success", "rate_limited", "github_error"} {
           pollCyclesTotal.WithLabelValues(result).Add(0)
       }
       for _, cmd := range []string{"create", "update_frontmatter", "skipped", "error"} {
           prPublishedTotal.WithLabelValues(cmd).Add(0)
       }
   }

   type prometheusMetrics struct{}

   func NewMetrics() Metrics {
       return &prometheusMetrics{}
   }

   func (m *prometheusMetrics) IncPollCycle(result string) {
       pollCyclesTotal.WithLabelValues(result).Inc()
   }

   func (m *prometheusMetrics) IncPRPublished(command string) {
       prPublishedTotal.WithLabelValues(command).Inc()
   }
   ```

2. **Add `//go:generate` directive** for the new mock in `watcher/github/pkg/suite_test.go` — it already has the `//go:generate go run -mod=mod github.com/maxbrunsfeld/counterfeiter/v6 -generate` directive. Add the counterfeiter annotation to `metrics.go` (done in step 1).

3. **Regenerate mocks**:
   ```bash
   cd watcher/github && go generate ./pkg/...
   ```
   Creates `watcher/github/pkg/mocks/metrics.go`.

4. **Update `watcher/github/pkg/watcher.go`**:
   - Add `metrics Metrics` field to `watcher` struct (~line 47)
   - Add `metrics Metrics` parameter to `NewWatcher` constructor (~line 25)
   - Instrument `Poll`:
     - After `fetchAllPRs` returns `ok=false` due to GitHub error (~line 66): `w.metrics.IncPollCycle("github_error")`
     - After `fetchAllPRs` returns `ok=false` due to rate limit (in `fetchAllPRs` at ~line 105): `w.metrics.IncPollCycle("rate_limited")`; this requires passing metrics into `fetchAllPRs` or using a bool return to distinguish the two abort reasons. Preferred: have `fetchAllPRs` return a typed abort reason string (e.g. `""` for success, `"github_error"`, `"rate_limited"`) instead of `([]PullRequest, bool)`, and call the right metric in `Poll`.
     - After `SaveCursor` on the happy path: `w.metrics.IncPollCycle("success")`
   - Instrument `processPRs` / per-PR handling:
     - In `publishCreate` success: `w.metrics.IncPRPublished("create")`
     - In `publishCreate` error: `w.metrics.IncPRPublished("error")`
     - In `publishForcePush` success: `w.metrics.IncPRPublished("update_frontmatter")`
     - In `publishForcePush` error: `w.metrics.IncPRPublished("error")`
     - In `handlePR` no-change (skipped) branch: `w.metrics.IncPRPublished("skipped")`
     - For filtered PRs (`ShouldSkipPR` returns true): `w.metrics.IncPRPublished("skipped")`

5. **Update `watcher/github/pkg/factory/factory.go`**:
   - Add `pkg.NewMetrics()` construction in `CreateWatcher`
   - Pass it to `pkg.NewWatcher(...)` as the new `metrics` parameter

6. **Update `watcher/github/pkg/watcher_test.go`**:
   - Add `FakeMetrics` from `pkg/mocks/metrics.go` to the test setup
   - Pass it to `pkg.NewWatcher(...)` in all test `BeforeEach` / `JustBeforeEach` blocks
   - Add assertions for key metric calls in relevant test cases (at minimum: "success poll cycle increments poll counter", "github error increments github_error counter")

7. Run `cd watcher/github && make test` — must pass.

8. Run `cd watcher/github && make precommit` — must exit 0.
</requirements>

<constraints>
- Only change files in `watcher/github/`
- Do NOT commit — dark-factory handles git
- Existing tests must still pass
- Metrics interface must have counterfeiter annotation and the mock must be generated (not hand-written)
- Use `prometheus.MustRegister` in `init()` — never register in constructors
- All known label values must be pre-initialized with `.Add(0)` in `init()`
- Use `errors.Wrapf(ctx, err, "...")` from `github.com/bborbe/errors` — never `fmt.Errorf`
- The `prometheusMetrics` struct is unexported; only the `Metrics` interface and `NewMetrics()` constructor are exported
</constraints>

<verification>
cd watcher/github && grep -n "type Metrics interface" pkg/metrics.go
# Expected: one match

cd watcher/github && grep -n "IncPollCycle\|IncPRPublished" pkg/watcher.go
# Expected: multiple matches (instrumentation points)

cd watcher/github && ls pkg/mocks/metrics.go
# Expected: file exists

cd watcher/github && make precommit
</verification>
