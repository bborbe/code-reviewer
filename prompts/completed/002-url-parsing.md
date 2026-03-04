---
status: completed
summary: Implemented GitHub PR URL parsing with PRInfo struct and ParsePRURL function
container: pr-reviewer-002-url-parsing
dark-factory-version: v0.14.5
created: "2026-03-04T14:03:58Z"
queued: "2026-03-04T14:03:58Z"
started: "2026-03-04T14:03:58Z"
completed: "2026-03-04T14:08:19Z"
---
<objective>
Implement GitHub PR URL parsing. The tool needs to extract owner, repo, and PR number from a GitHub PR URL, and construct the repo URL for config lookup.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Precondition: pkg/config exists with Loader and FindRepoPath.
Spec: specs/001-mvp-review-single-pr.md — URL Pattern section.
</context>

<requirements>
1. Create `pkg/github/url.go`:
   - Define `PRInfo` struct: `Owner string`, `Repo string`, `Number int`, `RepoURL string`
   - Define function `ParsePRURL(rawURL string) (*PRInfo, error)`
   - Parse URL pattern: `https://github.com/{owner}/{repo}/pull/{number}`
   - Extract owner, repo, PR number
   - Construct RepoURL: `https://github.com/{owner}/{repo}`
   - Return error for non-GitHub URLs: `"unsupported URL format: <url>"`
   - Return error for malformed GitHub URLs (missing parts, non-numeric PR number)
   - Accept URLs with or without trailing slash

2. Create `pkg/github/url_test.go`:
   - Ginkgo v2 + Gomega test suite
   - Test valid URL → correct Owner, Repo, Number, RepoURL
   - Test URL with trailing slash → works
   - Test non-GitHub URL → error "unsupported URL format"
   - Test GitHub URL but not a PR (e.g., /issues/123) → error
   - Test GitHub PR URL with non-numeric number → error
   - Test empty string → error
   - Test URL without scheme → error
</requirements>

<implementation>
Use net/url from stdlib for parsing. Use regexp or strings.Split for path extraction:

```go
func ParsePRURL(rawURL string) (*PRInfo, error) {
    u, err := url.Parse(rawURL)
    if err != nil {
        return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
    }
    if u.Host != "github.com" {
        return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
    }
    // split path, extract owner/repo/pull/number
}
```
</implementation>

<constraints>
- URL parsing: validate against GitHub pattern only, reject anything else
- Must use net/url from stdlib for URL parsing (no third-party URL libs)
- Error messages must match spec: "unsupported URL format: <url>"
- Package name must be `github_test` (external test package)
- Use Ginkgo v2 + Gomega for tests
- Do NOT touch pkg/config or main.go
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- pkg/github package exists with ParsePRURL function
- All valid GitHub PR URLs parse correctly
- All invalid URLs return "unsupported URL format" error
- Tests cover 7+ cases (valid, trailing slash, non-GitHub, not-PR, non-numeric, empty, no-scheme)
- make precommit passes
</success_criteria>
