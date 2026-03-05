---
status: completed
summary: Added Bitbucket Server URL parsing alongside existing GitHub URL parsing
container: pr-reviewer-017-bitbucket-url-parser
dark-factory-version: v0.17.0
created: "2026-03-05T10:51:03Z"
queued: "2026-03-05T10:51:03Z"
started: "2026-03-05T10:51:03Z"
completed: "2026-03-05T10:56:17Z"
---
<objective>
Add Bitbucket Server URL parsing alongside existing GitHub URL parsing. Extract host, project, repo, and PR number from Bitbucket Server PR URLs.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Read pkg/github/url.go for the existing ParsePRURL function and PRInfo struct.
Read pkg/github/url_test.go for existing test patterns.
Read main.go to understand how ParsePRURL is called.
Precondition: PRInfo struct exists with Owner, Repo, Number, RepoURL fields.
Precondition: ParsePRURL only handles GitHub URLs (returns error for non-GitHub).
</context>

<requirements>
1. Create `pkg/prurl/prurl.go` with a platform-agnostic PR URL parser:
   - Define `Platform` type (string): `PlatformGitHub`, `PlatformBitbucket`
   - Define `PRInfo` struct: `Platform`, `Host`, `Owner` (GitHub) or `Project` (Bitbucket), `Repo`, `Number`, `RepoURL`
   - `Parse(rawURL string) (*PRInfo, error)` — detects platform and delegates
   - GitHub pattern: `https://github.com/{owner}/{repo}/pull/{number}`
   - Bitbucket Server pattern: `https://{host}/projects/{project}/repos/{repo}/pull-requests/{number}` (with optional `/overview` suffix)
   - RepoURL for Bitbucket: `https://{host}/projects/{project}/repos/{repo}`
   - For GitHub: `Owner` = owner, `Project` = "" (unused)
   - For Bitbucket: `Owner` = project (used as owner equivalent), `Project` = project
   - Return error for unrecognized URL patterns

2. Add comprehensive tests in `pkg/prurl/prurl_test.go`:
   - GitHub URLs: standard, trailing slash, missing parts, non-pull path
   - Bitbucket URLs: with /overview suffix, without suffix, trailing slash, missing parts
   - Empty URL, HTTP (no HTTPS), malformed URLs
   - Verify Platform field is set correctly
   - Verify RepoURL is constructed correctly for both platforms

3. Add `pkg/prurl/prurl_suite_test.go` for Ginkgo test suite.

4. Update `main.go`: replace `github.ParsePRURL(rawURL)` with `prurl.Parse(rawURL)`.
   - Update import from `github.ParsePRURL` to `prurl.Parse`
   - The returned PRInfo fields are used: `prInfo.Owner`, `prInfo.Repo`, `prInfo.Number`, `prInfo.RepoURL`
   - For now, all paths still use the GitHub client (Bitbucket client comes in a later prompt)

5. Keep `pkg/github/url.go` and `pkg/github/url_test.go` — do NOT delete them yet (other code may reference PRInfo type). Mark ParsePRURL as deprecated with a comment.
</requirements>

<constraints>
- Existing GitHub support must continue to work unchanged
- HTTPS required — reject HTTP URLs for Bitbucket
- Use Ginkgo v2 + Gomega for tests
- Do NOT add Bitbucket API client — only URL parsing in this prompt
- Do NOT modify pkg/github/client.go
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- Bitbucket Server URLs parsed correctly (host, project, repo, number)
- GitHub URLs still parsed correctly
- Platform field distinguishes GitHub from Bitbucket
- RepoURL constructed correctly for both platforms
- main.go uses new prurl.Parse
- make precommit passes
</success_criteria>
