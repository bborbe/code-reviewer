// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bborbe/errors"

	"github.com/bborbe/pr-reviewer/pkg/config"
	"github.com/bborbe/pr-reviewer/pkg/git"
	"github.com/bborbe/pr-reviewer/pkg/github"
	"github.com/bborbe/pr-reviewer/pkg/review"
)

func main() {
	// Parse flags
	verbose := flag.Bool("v", false, "enable verbose output")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: pr-reviewer [-v] <pr-url>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := run(ctx, *verbose); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, verbose bool) error {
	// Parse args
	if flag.NArg() < 1 {
		return fmt.Errorf("usage: pr-reviewer [-v] <pr-url>")
	}
	rawURL := flag.Arg(0)

	// Parse PR URL
	logVerbose(verbose, "parsing URL: %s", rawURL)
	prInfo, err := github.ParsePRURL(rawURL)
	if err != nil {
		return err
	}

	// Load config
	configPath := "~/.pr-reviewer.yaml"
	logVerbose(verbose, "loading config: %s", configPath)
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
	repoPath := config.ExpandHome(repoInfo.Path)
	logVerbose(verbose, "repo: %s", repoPath)

	// Check if token is configured but env var is empty
	if cfg.GitHub.Token != "" && cfg.ResolvedGitHubToken() == "" {
		log.Println("warning: github.token configured but env var is empty, using default gh auth")
	}

	// Initialize components
	ghClient := github.NewGHClient(cfg.ResolvedGitHubToken())
	worktreeManager := git.NewWorktreeManager()
	reviewer := review.NewClaudeReviewer()

	// Get PR branch name
	branch, err := ghClient.GetPRBranch(ctx, prInfo.Owner, prInfo.Repo, prInfo.Number)
	if err != nil {
		return errors.Wrap(ctx, err, "get PR branch failed")
	}
	logVerbose(verbose, "fetching branch: %s", branch)

	// Fetch latest changes
	if err := worktreeManager.Fetch(ctx, repoPath); err != nil {
		return errors.Wrap(ctx, err, "fetch failed")
	}

	// Create worktree
	worktreePath, err := worktreeManager.CreateWorktree(ctx, repoPath, branch, prInfo.Number)
	if err != nil {
		return errors.Wrap(ctx, err, "create worktree failed")
	}
	logVerbose(verbose, "created worktree: %s", worktreePath)

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

	// Run review and post comment
	return runReviewAndPost(
		ctx,
		verbose,
		reviewer,
		ghClient,
		worktreePath,
		repoInfo.ReviewCommand,
		cfg.ResolvedModel(),
		prInfo,
	)
}

// runReviewAndPost executes the review and posts the comment.
func runReviewAndPost(
	ctx context.Context,
	verbose bool,
	reviewer review.Reviewer,
	ghClient github.Client,
	worktreePath string,
	reviewCommand string,
	model string,
	prInfo *github.PRInfo,
) error {
	// Run review
	logAlways("reviewing PR #%d (%s/%s)...", prInfo.Number, prInfo.Owner, prInfo.Repo)
	logVerbose(verbose, "running review... (this may take a few minutes)")
	reviewText, err := reviewer.Review(ctx, worktreePath, reviewCommand, model)
	if err != nil {
		return errors.Wrap(ctx, err, "review failed")
	}

	// Always print review to stdout
	fmt.Println(reviewText)

	// Post comment
	logAlways("posting comment...")
	if err := ghClient.PostComment(
		ctx,
		prInfo.Owner,
		prInfo.Repo,
		prInfo.Number,
		reviewText,
	); err != nil {
		return errors.Wrap(ctx, err, "post comment failed")
	}

	logAlways("done")
	return nil
}

// logAlways logs a message to stderr in both normal and verbose mode.
func logAlways(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// logVerbose logs a message to stderr only in verbose mode.
func logVerbose(verbose bool, format string, args ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}
