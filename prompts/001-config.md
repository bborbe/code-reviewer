<objective>
Implement config loading for pr-reviewer. The tool needs a YAML config file (~/.pr-reviewer.yaml) that maps repo URLs to local checkout paths. This is the foundation — all other prompts depend on config lookup.
</objective>

<context>
Read CLAUDE.md for project conventions.
Read ALL markdown files in ~/Documents/workspaces/coding-guidelines/ for Go patterns.
Current state: stub main.go with usage message, no packages yet.
Spec: specs/001-mvp-review-single-pr.md
</context>

<requirements>
1. Create `pkg/config/config.go`:
   - Define `Config` struct with `Repos []RepoConfig`
   - Define `RepoConfig` struct with `URL string` and `Path string` (yaml tags: `url`, `path`)
   - Define `Loader` interface: `Load(ctx context.Context) (*Config, error)`
   - Implement `fileLoader` struct that reads from a file path
   - Constructor: `NewFileLoader(configPath string) Loader`
   - `Load` reads the YAML file, unmarshals into Config, returns it
   - Default config path: `~/.pr-reviewer.yaml` (expand `~` to home dir)

2. Add method `Config.FindRepoPath(repoURL string) (string, error)`:
   - Looks up repoURL in Repos list (case-insensitive match, normalize trailing slashes)
   - Returns the local path if found
   - Returns error if not found: `"repo not found in config, add to ~/.pr-reviewer.yaml: <url>"`

3. Add validation in `Load`:
   - Config file doesn't exist → return error: `"config not found: <path>"`
   - Config file invalid YAML → return error with parse details
   - Repo entry with empty URL or empty Path → return error: `"invalid repo entry: url and path required"`

4. Create `pkg/config/config_test.go`:
   - Ginkgo v2 + Gomega test suite
   - Test Load with valid YAML (use os.MkdirTemp for temp config file)
   - Test Load with missing file → error contains "config not found"
   - Test Load with invalid YAML → error
   - Test Load with empty URL → validation error
   - Test FindRepoPath with matching URL → returns path
   - Test FindRepoPath with non-matching URL → error contains "repo not found"
   - Test FindRepoPath case-insensitive matching
   - Test FindRepoPath with/without trailing slash normalization

5. Add `gopkg.in/yaml.v3` to go.mod (run `go get gopkg.in/yaml.v3`)
</requirements>

<implementation>
Follow the Interface → Constructor → Struct → Method pattern from coding guidelines:

```go
// Interface
type Loader interface {
    Load(ctx context.Context) (*Config, error)
}

// Constructor
func NewFileLoader(configPath string) Loader {
    return &fileLoader{configPath: configPath}
}

// Struct
type fileLoader struct {
    configPath string
}

// Method
func (l *fileLoader) Load(ctx context.Context) (*Config, error) {
    // expand ~, read file, unmarshal, validate
}
```

For tests, follow this pattern from coding-guidelines:
```go
var _ = Describe("Config", func() {
    var (
        ctx    context.Context
        loader config.Loader
        tmpDir string
    )
    BeforeEach(func() {
        ctx = context.Background()
        tmpDir = GinkgoT().TempDir()
    })
    // ...
})
```
</implementation>

<constraints>
- Config file location: ~/.pr-reviewer.yaml
- Use gopkg.in/yaml.v3 for YAML parsing
- Package name must be `config_test` (external test package)
- Never hand-write mocks — use counterfeiter only (go generate)
- Use Ginkgo v2 + Gomega for tests
- Do NOT modify existing main.go yet (that's a later prompt)
- Do NOT add any CLI flag parsing yet
</constraints>

<verification>
Run `make test` -- must pass.
Run `make precommit` -- must pass.
</verification>

<success_criteria>
- pkg/config package exists with Loader interface and fileLoader implementation
- Config.FindRepoPath works with case-insensitive URL matching
- All error messages match spec failure modes
- Tests cover happy path + all 4 error cases
- make precommit passes
</success_criteria>
