---
status: completed
summary: Created comprehensive README.md with installation, usage, configuration, and workflow documentation
container: pr-reviewer-011-add-readme
dark-factory-version: v0.14.5
created: "2026-03-04T21:18:46Z"
queued: "2026-03-04T21:18:46Z"
started: "2026-03-04T21:18:46Z"
completed: "2026-03-04T21:21:53Z"
---
<objective>
Add a comprehensive README.md for the pr-reviewer project.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Read specs/001-mvp-review-single-pr.md for feature overview.
Read main.go for CLI usage and flags.
Read pkg/config/config.go for config structure.
Precondition: project has no README.md at all.
</context>

<requirements>
1. Create README.md with these sections:
   - Title and one-line description
   - What it does (2-3 sentences)
   - Installation (`go install`)
   - Usage: `pr-reviewer [-v] <pr-url>`
   - Configuration: `~/.pr-reviewer.yaml` with full example showing all fields (github.token, model, repos with url/path/reviewCommand)
   - How it works (numbered steps: parse URL → find repo → fetch branch → worktree → claude review → post comment → cleanup)
   - Requirements: Go 1.26+, claude CLI, gh CLI
   - License: BSD (link to LICENSE file)

2. Keep it concise — no badges, no contributing section, no changelog link.

3. Use the actual config structure from pkg/config/config.go for the YAML example.

4. Show both minimal and full config examples.
</requirements>

<constraints>
- Match existing coding-guidelines README patterns
- No emoji
- No badges
- BSD license reference
</constraints>

<verification>
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- README.md exists at project root
- Contains installation, usage, config, and how-it-works sections
- Config example matches actual struct fields
- make precommit passes
</success_criteria>
