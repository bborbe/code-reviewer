---
status: created
---
<objective>
Remove the useDocker toggle and always run reviews inside the claude-yolo Docker container. Remove the host-based claudeReviewer entirely.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read pkg/review/review.go for both claudeReviewer and dockerReviewer implementations.
Read pkg/config/config.go for the Config struct (UseDocker, ContainerImage fields).
Read main.go for the reviewer selection logic.
</context>

<requirements>
1. Remove `UseDocker bool` from Config struct and all references.

2. Keep `ContainerImage string` in Config (yaml: `containerImage`) with default `docker.io/bborbe/claude-yolo:v0.0.9` via `ResolvedContainerImage()`.

3. Remove `claudeReviewer` struct and `NewClaudeReviewer()` from pkg/review/review.go.

4. Remove `filterEnv` / `FilterEnv` helper if only used by claudeReviewer.

5. In main.go, replace the if/else reviewer selection with direct `review.NewDockerReviewer(cfg.ResolvedContainerImage())` — no conditional.

6. Update tests: remove any tests specific to claudeReviewer or UseDocker toggle.

7. Update README.md: remove `useDocker` from config docs, keep `containerImage`.
</requirements>

<constraints>
- dockerReviewer implementation must remain unchanged
- ContainerImage config field and ResolvedContainerImage() must remain unchanged
- All existing dockerReviewer tests must continue to pass
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
