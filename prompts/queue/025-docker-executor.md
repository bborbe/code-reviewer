---
status: approved
---
<objective>
Add a Docker-based review executor that runs Claude inside the claude-yolo container, matching dark-factory's execution pattern.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read pkg/review/review.go for the current Reviewer interface and claudeReviewer implementation.
Read specs/007-sandboxed-review-execution.md for the full spec.

Reference (do not import): ~/Documents/workspaces/dark-factory/pkg/executor/executor.go — this shows the Docker volume mount pattern used by dark-factory:
- `-v projectRoot:/workspace`
- `-v home/.claude-yolo:/home/node/.claude`
- `-v home/go/pkg:/home/node/go/pkg`
</context>

<requirements>
1. Add a new `dockerReviewer` struct implementing the existing `Reviewer` interface.

2. `NewDockerReviewer(containerImage string) Reviewer` constructor.

3. The `Review(ctx, worktreePath, command, model string) (string, error)` method:
   - Gets user home directory
   - Runs: `docker run --rm --cap-add=NET_ADMIN --cap-add=NET_RAW -w /workspace -v <worktreePath>:/workspace -v <home>/.claude-yolo:/home/node/.claude -v <home>/go/pkg:/home/node/go/pkg <containerImage> claude --print --model <model> <command>`
   - `-w /workspace` sets the working directory inside container
   - No env var forwarding — Claude auth comes from the mounted `~/.claude-yolo` config dir (same as dark-factory)
   - Do NOT pass host env vars into container (no `-e` flags for API keys)
   - Captures stdout as review text, stderr for errors
   - Returns review text on success
   - Export the existing `filterEnv` function as `FilterEnv` so dockerReviewer can reuse it (or duplicate the logic in the same package)

4. The container image should be configurable. Add `containerImage` field to the top-level Config struct (yaml: `containerImage`), defaulting to `docker.io/bborbe/claude-yolo:v0.0.9`.

5. Add `useDocker` bool to Config (yaml: `useDocker`, default: false). When true, use `NewDockerReviewer`; when false, use existing `NewClaudeReviewer`.

6. Update main.go to select reviewer based on `cfg.UseDocker`:
   ```go
   var reviewer review.Reviewer
   if cfg.UseDocker {
       reviewer = review.NewDockerReviewer(cfg.ContainerImage)
   } else {
       reviewer = review.NewClaudeReviewer()
   }
   ```

7. Add tests for dockerReviewer (unit tests with mock command runner, not integration tests requiring Docker).
</requirements>

<constraints>
- Default behavior unchanged (useDocker: false → existing claudeReviewer)
- Docker executor must handle container cleanup (--rm flag)
- No Docker-in-Docker — this runs on the host
- Container image must be configurable (not hardcoded)
- Existing claudeReviewer must remain unchanged
- Do NOT import dark-factory packages — reimplement the pattern
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
