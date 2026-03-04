// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bborbe/errors"
)

// WorktreeManager manages git worktrees for isolated PR review.
//
//counterfeiter:generate -o ../../mocks/worktree-manager.go --fake-name WorktreeManager . WorktreeManager
type WorktreeManager interface {
	Fetch(ctx context.Context, repoPath string) error
	CreateWorktree(
		ctx context.Context,
		repoPath string,
		branch string,
		prNumber int,
	) (worktreePath string, err error)
	RemoveWorktree(ctx context.Context, repoPath string, worktreePath string) error
}

// NewWorktreeManager creates a WorktreeManager that uses git CLI.
func NewWorktreeManager() WorktreeManager {
	return &worktreeManager{}
}

type worktreeManager struct{}

// Fetch runs git fetch --all --prune in the repository.
func (m *worktreeManager) Fetch(ctx context.Context, repoPath string) error {
	if err := m.validateRepoPath(ctx, repoPath); err != nil {
		return errors.Wrap(ctx, err, "validate repo path failed")
	}

	if err := m.runGit(ctx, repoPath, "fetch", "--all", "--prune"); err != nil {
		return errors.Wrap(ctx, err, "git fetch failed")
	}

	return nil
}

// CreateWorktree creates a git worktree for the given branch.
// Worktree path is deterministic: <repoPath>/.worktrees/pr-<number>
// If a stale worktree exists, it is removed first.
func (m *worktreeManager) CreateWorktree(
	ctx context.Context,
	repoPath string,
	branch string,
	prNumber int,
) (string, error) {
	if err := m.validateRepoPath(ctx, repoPath); err != nil {
		return "", errors.Wrap(ctx, err, "validate repo path failed")
	}

	worktreePath := m.worktreePath(repoPath, prNumber)

	// Remove stale worktree if it exists
	if _, err := os.Stat(worktreePath); err == nil {
		if err := m.RemoveWorktree(ctx, repoPath, worktreePath); err != nil {
			return "", errors.Wrap(ctx, err, "remove stale worktree failed")
		}
	}

	// Create worktree
	err := m.runGit(ctx, repoPath, "worktree", "add", worktreePath, branch)
	if err != nil {
		if strings.Contains(err.Error(), "invalid reference") {
			return "", fmt.Errorf("branch not found: %s", branch)
		}
		return "", errors.Wrap(ctx, err, "git worktree add failed")
	}

	return worktreePath, nil
}

// RemoveWorktree removes a git worktree.
// Idempotent: returns nil if worktree doesn't exist.
func (m *worktreeManager) RemoveWorktree(
	ctx context.Context,
	repoPath string,
	worktreePath string,
) error {
	if err := m.validateRepoPath(ctx, repoPath); err != nil {
		return errors.Wrap(ctx, err, "validate repo path failed")
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return nil // idempotent
	}

	err := m.runGit(ctx, repoPath, "worktree", "remove", worktreePath, "--force")
	if err != nil {
		return errors.Wrap(ctx, err, "git worktree remove failed")
	}

	return nil
}

// validateRepoPath checks if the path exists and is a git repository.
func (m *worktreeManager) validateRepoPath(ctx context.Context, repoPath string) error {
	// Check path exists
	info, err := os.Stat(repoPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("local path not found: %s", repoPath)
	}
	if err != nil {
		return errors.Wrap(ctx, err, "stat path failed")
	}

	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", repoPath)
	}

	// Check if it's a git repo
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repo: %s", repoPath)
	}

	return nil
}

// worktreePath generates the deterministic worktree path.
func (m *worktreeManager) worktreePath(repoPath string, prNumber int) string {
	return filepath.Join(repoPath, ".worktrees", fmt.Sprintf("pr-%d", prNumber))
}

// runGit executes a git command and returns an error with stderr.
func (m *worktreeManager) runGit(
	ctx context.Context,
	repoPath string,
	args ...string,
) error {
	// #nosec G204 -- repoPath is validated by validateRepoPath, args are hardcoded git commands
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repoPath}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}

	return nil
}
