<objective>
Add Bitbucket Server review verdict methods (approve, needs-work, get-profile) to the existing Bitbucket client (pkg/bitbucket/client.go). These methods are needed for spec 005 to submit structured review verdicts instead of plain comments only.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read specs/005-bitbucket-approve-request-changes.md for the full spec.
Read pkg/bitbucket/client.go for the existing Bitbucket client implementation.
Read pkg/bitbucket/client_test.go for existing test patterns.
</context>

<requirements>
1. Add `GetProfile(ctx, host) (Profile, error)` method to the Bitbucket client interface and implementation
   - Calls `GET https://{host}/rest/api/1.0/profile` with Bearer token auth
   - Returns a `Profile` struct with at minimum a `Slug` field (parsed from JSON response `slug` field)
   - Handles 401/403 errors same as existing methods

2. Add `Approve(ctx, host, project, repo string, number int) error` method
   - Calls `POST https://{host}/rest/api/1.0/projects/{project}/repos/{repo}/pull-requests/{number}/approve`
   - No request body needed
   - Bearer token auth (same as existing methods)
   - Handles 401/403/404/409 errors (409 = already approved, treat as success)

3. Add `NeedsWork(ctx, host, project, repo string, number int, userSlug string) error` method
   - Calls `PUT https://{host}/rest/api/1.0/projects/{project}/repos/{repo}/pull-requests/{number}/participants/{userSlug}`
   - Request body: `{"user":{"slug":"<userSlug>"},"approved":false,"status":"NEEDS_WORK"}`
   - Content-Type: application/json
   - Bearer token auth
   - Handles 401/403/404 errors same as existing methods

4. Add tests for all three methods following existing test patterns in client_test.go
   - Use httptest.NewServer to mock API responses
   - Test success cases and error cases (401, 403, 404)
   - Test that Approve treats 409 as success

5. Regenerate counterfeiter mock: `go generate ./...`
</requirements>

<constraints>
- Only Bitbucket Server/Data Center REST API — no Bitbucket Cloud
- Existing PostComment and GetPRBranch methods must not change
- Bearer token auth for all API calls (same pattern as existing methods)
- Token must not appear in error messages
- HTTPS enforced (same buildURL helper)
</constraints>

<verification>
Run `make precommit` -- must pass.
</verification>
