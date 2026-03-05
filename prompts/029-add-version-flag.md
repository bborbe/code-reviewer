---
status: created
---
<objective>
Add a --version flag that prints the build-time version and exits.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read main.go for the current flag parsing.
Read pkg/version/version.go for the existing Version variable.
</context>

<requirements>
1. Add `--version` flag to main.go.

2. When `--version` is passed, print `pr-reviewer <version>` to stdout and exit 0.

3. Check `--version` before any other logic (before arg validation, config loading, etc.).

4. Update usage string to include `--version`.
</requirements>

<constraints>
- Use the existing `version.Version` variable (set at build time via ldflags)
- No other behavior changes
</constraints>

<verification>
Run `make precommit` — must pass.
</verification>
