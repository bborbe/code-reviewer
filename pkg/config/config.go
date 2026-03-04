// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bborbe/errors"
	"gopkg.in/yaml.v3"
)

// GitHubConfig holds GitHub-specific configuration.
type GitHubConfig struct {
	Token string `yaml:"token"`
}

// DefaultModel is the default Claude model to use when not specified in config.
const DefaultModel = "sonnet"

// DefaultGitHubToken is the default env var reference for the GitHub token.
const DefaultGitHubToken = "${PR_REVIEWER_GITHUB_TOKEN}"

// Config holds the pr-reviewer configuration.
type Config struct {
	GitHub GitHubConfig `yaml:"github"`
	Model  string       `yaml:"model"`
	Repos  []RepoConfig `yaml:"repos"`
}

// RepoConfig maps a repository URL to a local path.
type RepoConfig struct {
	URL           string `yaml:"url"`
	Path          string `yaml:"path"`
	ReviewCommand string `yaml:"reviewCommand"`
}

// RepoInfo holds repository information including path and review command.
type RepoInfo struct {
	Path          string
	ReviewCommand string
}

// resolveEnvVar resolves environment variable references in the format ${VAR_NAME}.
// If the value matches this pattern, it returns the value of the env var.
// Otherwise, it returns the value as-is.
func resolveEnvVar(value string) string {
	// Match ${VAR_NAME} pattern
	re := regexp.MustCompile(`^\$\{([A-Z_][A-Z0-9_]*)\}$`)
	matches := re.FindStringSubmatch(value)
	if len(matches) == 2 {
		return os.Getenv(matches[1])
	}
	return value
}

// ResolvedGitHubToken returns the GitHub token with environment variable resolution.
// If no token is configured, falls back to DefaultGitHubToken.
func (c *Config) ResolvedGitHubToken() string {
	token := c.GitHub.Token
	if token == "" {
		token = DefaultGitHubToken
	}
	return resolveEnvVar(token)
}

// ResolvedModel returns the configured model if non-empty, otherwise returns DefaultModel.
func (c *Config) ResolvedModel() string {
	if c.Model != "" {
		return c.Model
	}
	return DefaultModel
}

// Loader loads configuration from a source.
//
//counterfeiter:generate -o ../../mocks/config-loader.go --fake-name ConfigLoader . Loader
type Loader interface {
	Load(ctx context.Context) (*Config, error)
}

// NewFileLoader creates a Loader that reads from a YAML file.
func NewFileLoader(configPath string) Loader {
	return &fileLoader{configPath: configPath}
}

type fileLoader struct {
	configPath string
}

func (l *fileLoader) Load(ctx context.Context) (*Config, error) {
	expandedPath := ExpandHome(l.configPath)

	// #nosec G304 -- path from constructor, typically ~/.pr-reviewer.yaml
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrapf(ctx, err, "config not found: %s", expandedPath)
		}
		return nil, errors.Wrapf(ctx, err, "read config file failed")
	}

	// Check file permissions - warn if world-readable
	fileInfo, err := os.Stat(expandedPath)
	if err == nil {
		mode := fileInfo.Mode()
		if mode&0004 != 0 {
			log.Printf(
				"warning: config file is world-readable, consider: chmod 600 %s",
				expandedPath,
			)
		}
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrapf(ctx, err, "parse config failed")
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, errors.Wrapf(ctx, err, "validate config failed")
	}

	return &cfg, nil
}

// ExpandHome expands ~ to the user's home directory.
func ExpandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}

func validateConfig(cfg *Config) error {
	for _, repo := range cfg.Repos {
		if repo.URL == "" || repo.Path == "" {
			return fmt.Errorf("invalid repo entry: url and path required")
		}
	}
	return nil
}

// FindRepoPath looks up the local path for a given repository URL.
func (c *Config) FindRepoPath(repoURL string) (string, error) {
	normalizedURL := normalizeURL(repoURL)

	for _, repo := range c.Repos {
		if normalizeURL(repo.URL) == normalizedURL {
			return repo.Path, nil
		}
	}

	return "", fmt.Errorf("repo not found in config, add to ~/.pr-reviewer.yaml: %s", repoURL)
}

// FindRepo looks up the repository information including path and review command.
// Returns RepoInfo with path and reviewCommand. If reviewCommand is not specified,
// defaults to "/code-review".
func (c *Config) FindRepo(repoURL string) (*RepoInfo, error) {
	normalizedURL := normalizeURL(repoURL)

	for _, repo := range c.Repos {
		if normalizeURL(repo.URL) == normalizedURL {
			reviewCmd := repo.ReviewCommand
			if reviewCmd == "" {
				reviewCmd = "/code-review"
			}
			return &RepoInfo{
				Path:          repo.Path,
				ReviewCommand: reviewCmd,
			}, nil
		}
	}

	return nil, fmt.Errorf("repo not found in config, add to ~/.pr-reviewer.yaml: %s", repoURL)
}

func normalizeURL(url string) string {
	normalized := strings.ToLower(strings.TrimSpace(url))
	return strings.TrimSuffix(normalized, "/")
}
