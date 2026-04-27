# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## v0.15.2

- feat(watcher/github): add GitHub API layer — `GitHubClient` interface with `SearchPRs`/`GetHeadSHA`, `filter.ShouldSkip`, and `taskid.Derive` deterministic task ID derivation using UUID v5

## v0.15.1

- feat(watcher/github): add github-pr-watcher service scaffold (go.mod, Makefile, Dockerfile, main.go skeleton)

## v0.15.0

- feat: add plugin installer library (`pkg/plugins/`) ensuring Claude Code plugins are installed before task handling
- feat: wire plugin installer into agent-pr-reviewer startup — ensures `bborbe/coding` plugin is present on every pod boot
- docs: add `docs/claude-plugin-cli.md` documenting claude plugin CLI derivation rules

## v0.14.3

- feat(pr-reviewer): add `pkg/plugins` package with `Installer` interface and `NewExecCommander` for managing Claude Code plugins (install/update via `claude plugin` CLI)

## v0.14.2

- feat(pr-reviewer): preflight GH_TOKEN check as step 0 in every phase. New `pkg/steps/gh_token.go` hits GitHub's `/rate_limit` endpoint (free, doesn't count against the limit) before each Claude call. Routes failures explicitly:
  - empty token → `needs_input` → `human_review` (non-retryable)
  - HTTP 401 → `needs_input` → `human_review` (non-retryable, with truncated GH error body)
  - rate limit < 1000/hr → `needs_input` → `human_review` (token degraded to anonymous, e.g. revoked or scope-stripped)
  - remaining quota < 10 → `failed` → controller retries after backoff
  - network error / non-200 → `failed` → controller retries
  - healthy PAT → `done + ContinueToNext` (the actual Claude step runs next)
- Catches the exact failure mode that wasted 3 jobs in the v0.14.1 e2e smoke test: a teamvault-stored token that authenticates as user but rate-limits as anonymous. The agent now stops at preflight (~200ms, 1 HTTP call) with an actionable message instead of running through 3 phases of confusing "rate limit exceeded" errors from inside the LLM.
- 9 table-driven tests in `pkg/steps/gh_token_test.go` covering all branches via `httptest.Server`.
- New `make verify-gh-token` target in `agent/pr-reviewer/Makefile` — same check from the command line, useful before deploying.

## v0.14.1

- fix(pr-reviewer): tolerate prose around JSON in ai_review verdict. Caught during local smoke against PR #2: Claude prefixed the verdict JSON with explanatory prose despite the prompt asking for raw JSON only, causing `json.Unmarshal` to fail and incorrectly route to `human_review`. New `extractVerdict` walks the LLM response in 3 stages — direct unmarshal, fence-stripped unmarshal, last-balanced-`{...}`-block extraction — covered by 11 table-driven test cases in `pkg/steps/review_test.go`.

## v0.14.0

- feat(pr-reviewer): per-phase decomposition. Replace single shared Claude step with 3 distinct steps:
  - `planning` — read-only diff inspection (`git diff`, `gh pr view/diff`); writes `## Plan` JSON (files, scope, focus areas, concerns)
  - `in_progress` — read + cross-file inspection; reads `## Plan`, writes `## Review` JSON (verdict, summary, comments, concerns_addressed)
  - `ai_review` — minimal read-only fresh-context verifier; reads `## Plan` + `## Review`, writes `## Verdict` JSON; conditional next-phase routing — `verdict=pass` → `done`, anything else → `human_review`
- New per-phase prompt modules under `pkg/prompts/{planning,execution,review}/` with workflow.md + output-format.md; old generic prompt removed
- New `pkg/steps/review.go` — custom AgentStep that parses verdict JSON to drive conditional NextPhase
- Per-phase tool scopes in factory: planning + review are read-only; execution gets broader git/gh access; none can post comments (posting stays out-of-band after human approves verdict)

## v0.13.0

- refactor(pr-reviewer): migrate to agent framework (lib v0.54.0). Drop `claudelib.TaskRunner` / `NewResultDelivererAdapter` / `FallbackContentGenerator`; use `lib.NewAgent` + `claude.NewAgentStep` shared across 3 phases (planning, in_progress, ai_review) with `## Review` output section. Factory exposes `CreateAgent` + `CreateDeliverer` matching the canonical `agent/claude` shape. main.go gains typed `Phase` field; both entry points (Kafka main.go + cmd/run-task) updated.

## v0.12.3

- chore: bump github.com/bborbe/agent/lib to v0.53.1 (route-failures: failed status → phase: human_review + ## Failure section; UpdateFrontmatterCommand for spawn/failure notifications)

## v0.12.2

- Rework root README to document three modes (CLI, local task runner, k8s Job agent); add k8s deploy, prerequisites, trigger instructions, debug commands, full repo layout

## v0.12.1

- Replace os.Getenv passthrough in factory with typed GHToken argument on both main.go entries — factory receives ghToken string and wires GH_TOKEN into ClaudeRunnerConfig.Env only when non-empty

## v0.12.0

- Specialize pr-reviewer factory: hardcode AllowedTools + GH_TOKEN passthrough and move prompts.BuildInstructions() inside; drop AllowedToolsRaw/EnvContextRaw/ClaudeEnvRaw CLI args and parseKeyValuePairs helper from both main.go entries
- Type TaskID field as agentlib.TaskIdentifier directly (no string conversion)
- Rename Secret data key `PR_REVIEWER_GITHUB_TOKEN` → `GH_TOKEN` so gh CLI picks it up natively; drop `ALLOWED_TOOLS` from Config CRD env
- Add `make apply` target; `make buca` passes DOCKER_REGISTRY from env
- Add github-cli + git to container image; shrink image via `apk del npm` + `npm cache clean`
- Harden Makefile.env: error on invalid BRANCH, outdent conditional includes (Make parsed as recipe)

## v0.11.0

- feat: add k8s manifests for pr-reviewer (Config CRD, PVC, Secret, PriorityClass, ResourceQuota dev+prod, Makefile)

## v0.10.0

- Add `agent/pr-reviewer/main.go` + `main_test.go` — k8s job entry point mirroring `bborbe/agent/agent/claude/main.go` (task-content via env, optional Kafka result delivery on `TASK_ID`, configurable `AllowedTools`/`EnvContext`/`ClaudeEnv`)
- Add `agent/pr-reviewer/pkg/factory/factory.go` — wires `TaskRunner`, `ClaudeRunner`, `SyncProducer`, and `KafkaResultDeliverer` (verbatim claude factory, `serviceName = agent-pr-reviewer`)
- Add `agent/pr-reviewer/pkg/prompts/` with embedded `workflow.md` + `output-format.md` via `//go:embed`
- Rewrite `agent/pr-reviewer/cmd/run-task/main.go` as claude-style local runner: reads task file, uses `FileResultDeliverer` to write result back, configurable allowed-tools
- Update `cmd/run-task/Makefile` claude-style with `ALLOWED_TOOLS=Read,Grep,Glob,Bash(git:*),Bash(gh:*),WebFetch`; generates/runs dummy PR-review task
- Add `agent/pr-reviewer/agent/.claude/CLAUDE.md` — headless PR reviewer guardrails (no internal network, no state mutation, JSON-only output)
- Simplify `agent/pr-reviewer/Makefile` to use shared `Makefile.variables` + `Makefile.precommit` includes (reduced from ~100 to 14 lines); keeps own `install`/`run` targets with `VERSION`/`LDFLAGS`
- Fix `Makefile.precommit` `goimports-reviser` project-name from `github.com/bborbe/agent` to `github.com/bborbe/code-reviewer`
- Fix root `Makefile.folder` `DIRS` discovery to match bborbe/agent (`find */* -maxdepth 0`) so it targets service dirs (`agent/pr-reviewer`) instead of recursing into `cmd/run-task`
- Add `agent/pr-reviewer/Dockerfile` mirroring `bborbe/agent/agent/claude/Dockerfile` (multi-stage build → `/main` + `claude-code` CLI)
- Add `agent/pr-reviewer/.gitignore` (`.update-logs/`, `.claude/`, `CLAUDE.md`, `.mcp-*`, `cover.out`)
- Add `agent/pr-reviewer/README.md` describing service layout, entry points (local CLI vs planned k8s job), verdict contract
- Add root `common.env`, `default.env`, `dev.env`, `prod.env` (copied from bborbe/agent) so shared `Makefile.env`/`Makefile.docker` work
- Promote `bborbe/agent/lib`, `cqrs`, `kafka`, `sentry`, `service`, `time`, `golang/glog` to direct deps in `agent/pr-reviewer/go.mod`; bump `golang.org/x/vuln` to v1.2.0

## v0.9.0

- Transform repo to multi-module layout (bborbe/agent pattern): service at `agent/pr-reviewer/` with own `go.mod`, binary entry point at `cmd/run-task/main.go`; root has no `go.mod`
- Root `Makefile` delegates `precommit`/`test`/`lint` to service dirs via `Makefile.folder` (auto-discovers Makefiles at any depth)
- Copy shared Makefile includes from bborbe/agent: `Makefile.docker`, `Makefile.env`, `Makefile.k8s`, `Makefile.precommit`, `Makefile.variables`
- Update module path to `github.com/bborbe/code-reviewer/agent/pr-reviewer`; rewrite all imports and LDFLAGS
- Binary renamed `pr-reviewer` → `run-task` (matches Pattern B Job convention)
- `.golangci.yml`, `.osv-scanner.toml`, `.trivyignore` stay at repo root; service Makefile references via `../../`

## v0.8.0

- Rename module path from `github.com/bborbe/pr-reviewer` to `github.com/bborbe/code-reviewer` (repo renamed to cover broader scope)
- Update all imports, Makefile ldflags, prompts, specs, and docs to new module path
- Upgrade `github.com/go-git/go-git/v5` from v5.17.2 to v5.18.0 (security fix)
- Remove stale OSV ignore entries for GO-2026-4923, GHSA-6jwv-w5xf-7j27, GHSA-xmrv-pmrh-hhx2

## v0.7.4

- Update golangci-lint to v2.11.4
- Update osv-scanner to v2.3.5
- Update gosec to v2.25.0
- Update multiple indirect dependencies
- Bump Go toolchain to 1.26.2

## v0.7.3

- Update dependencies (docker, containerd, prometheus, otel, golang.org/x)
- Upgrade go-git to v5.17.2
- Upgrade moby/buildkit to v0.29.0
- Remove stale exclude and replace directives from go.mod

## v0.7.2

- upgrade golangci-lint from v1 to v2
- standardize Makefile: multiline trivy
- update .golangci.yml to v2 format
- setup dark-factory config

## v0.7.1

- go mod update

## v0.7.0

- Add --version flag to print build-time version and exit

## v0.6.0

- Use YOLO_OUTPUT=print for raw text output instead of stream-json extraction
- Update default container image to claude-yolo v0.2.0

## v0.5.9

- Fix Docker executor to use YOLO_PROMPT_FILE and YOLO_MODEL env vars (matching dark-factory pattern)
- Extract review result from stream-formatter output via --- DONE --- marker

## v0.5.8

- Replace git worktree with git clone --local for Docker-compatible standalone repos

## v0.5.7

- Remove useDocker toggle, always use Docker (claude-yolo) for reviews
- Remove host-based claudeReviewer

## v0.5.6

- Add configurable autoApprove field (default false) to guard approve API calls
- Refactor submitGitHubReview and submitBitbucketReview to respect autoApprove setting

## v0.5.5

- Add Docker-based review executor using claude-yolo container
- Add useDocker and containerImage config fields
- Mount ~/.claude-yolo as Claude config inside container

## v0.5.4

- Strip JSON verdict block from review text before posting as PR comment

## v0.5.3

- Add JSON verdict parser (parseJSONVerdict) with fallback to heuristic section scanning
- Add StripJSONVerdict to remove verdict block from review output
- Support JSON verdict inside markdown code fences

## v0.5.2

- Fix verdict parser treating markdown horizontal rules (---) as must-fix content

## v0.5.1

- Fix Bitbucket needs-work verdict by replacing broken /profile endpoint with configurable username
- Remove GetProfile from Bitbucket client (404 on Bitbucket Data Center)
- Add bitbucket.username config field for needs-work participant API
- Fix /pr-review vendor/node_modules exclusion for nested directories

## v0.5.0

- Use /pr-review command with target branch for diff-scoped reviews instead of /code-review
- Fetch both source and target branch from GitHub and Bitbucket PR APIs
- Rename GetPRBranch to GetPRBranches returning source and target branch pair

## v0.4.1

- Add progress logging for long-running operations (fetch, worktree, review, post)
- Move worktrees to /tmp to avoid polluting repo directory
- Add robust stale worktree cleanup with fallback to force-remove

## v0.4.0

- Add Bitbucket Server support: parse PR URLs, fetch branch, post comments via REST API
- Add platform-agnostic URL parser (pkg/prurl) supporting GitHub and Bitbucket Server
- Add Bitbucket API client (pkg/bitbucket) with Bearer token auth and error handling
- Add Bitbucket token configuration with BITBUCKET_TOKEN env var default
- Route GitHub and Bitbucket URLs to respective clients in main workflow

## v0.3.0

- Wire verdict-based review submission into main workflow
- Add --comment-only flag to skip verdict and post as plain comment
- Log detected verdict and reason to stderr

## v0.2.0

- Add verdict parser for review output analysis (approve/request-changes/comment)
- Add SubmitReview to GitHub Client for structured review submission via gh CLI
- Add verbose version display and token debug logging
- Add build-time version injection via pkg/version
- Clean up default GitHub token constant

## v0.1.1

- Fix LICENSE year from 2016 to 2025
- Fix README license type from BSD 3-Clause to BSD 2-Clause
- Update README token example to PR_REVIEWER_GITHUB_TOKEN
- Add CLAUDE.md to .gitignore
- Default github token to ${PR_REVIEWER_GITHUB_TOKEN} env var

## v0.1.0

- Initial project setup
