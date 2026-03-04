// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bborbe/errors"
	"gopkg.in/yaml.v3"
)

// Config holds the pr-reviewer configuration.
type Config struct {
	Repos []RepoConfig `yaml:"repos"`
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
	expandedPath := expandHome(l.configPath)

	// #nosec G304 -- path from constructor, typically ~/.pr-reviewer.yaml
	data, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrapf(ctx, err, "config not found: %s", expandedPath)
		}
		return nil, errors.Wrapf(ctx, err, "read config file failed")
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

func expandHome(path string) string {
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
