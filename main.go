// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/bborbe/errors"

	"github.com/bborbe/pr-reviewer/pkg/config"
	"github.com/bborbe/pr-reviewer/pkg/git"
	"github.com/bborbe/pr-reviewer/pkg/github"
	"github.com/bborbe/pr-reviewer/pkg/review"
)

func main() {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Parse args
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: pr-reviewer <pr-url>")
	}
	rawURL := os.Args[1]

	// Parse PR URL
	prInfo, err := github.ParsePRURL(rawURL)
	if err != nil {
		return err
	}

	// Load config
	configPath := "~/.pr-reviewer.yaml"
	loader := config.NewFileLoader(configPath)
	cfg, err := loader.Load(ctx)
	if err != nil {
		return err
	}

	// Find local repo information
	repoInfo, err := cfg.FindRepo(prInfo.RepoURL)
	if err != nil {
		return err
	}

	// Expand home directory in path
	repoPath := expandHome(repoInfo.Path)

	// Initialize components
	ghClient := github.NewGHClient()
	worktreeManager := git.NewWorktreeManager()
	reviewer := review.NewClaudeReviewer()

	// Get PR branch name
	branch, err := ghClient.GetPRBranch(ctx, prInfo.Owner, prInfo.Repo, prInfo.Number)
	if err != nil {
		return errors.Wrap(ctx, err, "get PR branch failed")
	}

	// Fetch latest changes
	if err := worktreeManager.Fetch(ctx, repoPath); err != nil {
		return errors.Wrap(ctx, err, "fetch failed")
	}

	// Create worktree
	worktreePath, err := worktreeManager.CreateWorktree(ctx, repoPath, branch, prInfo.Number)
	if err != nil {
		return errors.Wrap(ctx, err, "create worktree failed")
	}

	// Ensure cleanup on exit
	defer func() {
		cleanupCtx := context.Background()
		if cleanupErr := worktreeManager.RemoveWorktree(
			cleanupCtx,
			repoPath,
			worktreePath,
		); cleanupErr != nil {
			fmt.Fprintf(
				os.Stderr,
				"warning: cleanup failed: %v\n",
				cleanupErr,
			)
		}
	}()

	// Run review
	reviewText, err := reviewer.Review(ctx, worktreePath, repoInfo.ReviewCommand)
	if err != nil {
		return errors.Wrap(ctx, err, "review failed")
	}

	// Always print review to stdout
	fmt.Println(reviewText)

	// Post comment
	if err := ghClient.PostComment(
		ctx,
		prInfo.Owner,
		prInfo.Repo,
		prInfo.Number,
		reviewText,
	); err != nil {
		return errors.Wrap(ctx, err, "post comment failed")
	}

	return nil
}

// expandHome expands ~ to the user's home directory.
func expandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	if len(path) == 1 {
		return home
	}

	return filepath.Join(home, path[1:])
}
