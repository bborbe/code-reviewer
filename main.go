// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/bborbe/errors"

	"github.com/bborbe/pr-reviewer/pkg/bitbucket"
	"github.com/bborbe/pr-reviewer/pkg/config"
	"github.com/bborbe/pr-reviewer/pkg/git"
	"github.com/bborbe/pr-reviewer/pkg/github"
	"github.com/bborbe/pr-reviewer/pkg/prurl"
	"github.com/bborbe/pr-reviewer/pkg/review"
	"github.com/bborbe/pr-reviewer/pkg/verdict"
	"github.com/bborbe/pr-reviewer/pkg/version"
)

func main() {
	// Parse flags
	verbose := flag.Bool("v", false, "enable verbose output")
	commentOnly := flag.Bool("comment-only", false, "skip verdict, post as plain comment")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: pr-reviewer [-v] [--comment-only] <pr-url>\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer cancel()

	if err := run(ctx, *verbose, *commentOnly); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, verbose bool, commentOnly bool) error {
	// Parse args
	if flag.NArg() < 1 {
		return fmt.Errorf("usage: pr-reviewer [-v] [--comment-only] <pr-url>")
	}
	rawURL := flag.Arg(0)

	// Log version
	logVerbose(verbose, "pr-reviewer %s", version.Version)

	// Parse PR URL
	logVerbose(verbose, "parsing URL: %s", rawURL)
	prInfo, err := prurl.Parse(rawURL)
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

	// Route based on platform
	switch prInfo.Platform {
	case prurl.PlatformGitHub:
		return runGitHub(ctx, verbose, commentOnly, cfg, prInfo, repoPath, repoInfo)
	case prurl.PlatformBitbucket:
		return runBitbucket(ctx, verbose, commentOnly, cfg, prInfo, repoPath, repoInfo)
	default:
		return fmt.Errorf("unsupported platform: %s", prInfo.Platform)
	}
}

// runGitHub handles the GitHub PR review workflow.
func runGitHub(
	ctx context.Context,
	verbose bool,
	commentOnly bool,
	cfg *config.Config,
	prInfo *prurl.PRInfo,
	repoPath string,
	repoInfo *config.RepoInfo,
) error {
	// Resolve token and create client
	resolvedToken := cfg.ResolvedGitHubToken()
	logTokenStatus(
		verbose,
		"github token",
		cfg.GitHub.Token,
		resolvedToken,
		config.DefaultGitHubToken,
	)
	ghClient := github.NewGHClient(resolvedToken)

	// Get PR branch names
	logAlways("fetching PR #%d metadata...", prInfo.Number)
	branches, err := ghClient.GetPRBranches(ctx, prInfo.Owner, prInfo.Repo, prInfo.Number)
	if err != nil {
		return errors.Wrap(ctx, err, "get PR branches failed")
	}
	logVerbose(verbose, "source branch: %s, target branch: %s", branches.Source, branches.Target)

	// Create worktree and run review
	worktreePath, cleanup, err := createWorktreeAndFetch(
		ctx,
		verbose,
		repoPath,
		branches.Source,
		prInfo.Number,
	)
	if err != nil {
		return err
	}
	defer cleanup()

	// Run review
	reviewer := review.NewClaudeReviewer()
	reviewCommand := buildReviewCommand(repoInfo.ReviewCommand, branches.Target)
	reviewText, result, err := runReview(
		ctx,
		reviewer,
		worktreePath,
		reviewCommand,
		cfg.ResolvedModel(),
		prInfo,
	)
	if err != nil {
		return err
	}

	// Submit review or post comment
	return submitGitHubReview(ctx, commentOnly, result, ghClient, prInfo, reviewText)
}

// runBitbucket handles the Bitbucket Server PR review workflow.
func runBitbucket(
	ctx context.Context,
	verbose bool,
	commentOnly bool,
	cfg *config.Config,
	prInfo *prurl.PRInfo,
	repoPath string,
	repoInfo *config.RepoInfo,
) error {
	// Resolve token and create client
	resolvedToken := cfg.ResolvedBitbucketToken()
	logTokenStatus(
		verbose,
		"bitbucket token",
		cfg.Bitbucket.Token,
		resolvedToken,
		config.DefaultBitbucketToken,
	)
	if resolvedToken == "" {
		return fmt.Errorf("BITBUCKET_TOKEN not set")
	}
	bbClient := bitbucket.NewClient(resolvedToken)

	// Get PR branch names
	logAlways("fetching PR #%d metadata...", prInfo.Number)
	branches, err := bbClient.GetPRBranches(
		ctx,
		prInfo.Host,
		prInfo.Project,
		prInfo.Repo,
		prInfo.Number,
	)
	if err != nil {
		return errors.Wrap(ctx, err, "get PR branches failed")
	}
	logVerbose(verbose, "source branch: %s, target branch: %s", branches.Source, branches.Target)

	// Create worktree and run review
	worktreePath, cleanup, err := createWorktreeAndFetch(
		ctx,
		verbose,
		repoPath,
		branches.Source,
		prInfo.Number,
	)
	if err != nil {
		return err
	}
	defer cleanup()

	// Run review
	reviewer := review.NewClaudeReviewer()
	reviewCommand := buildReviewCommand(repoInfo.ReviewCommand, branches.Target)
	reviewText, result, err := runReview(
		ctx,
		reviewer,
		worktreePath,
		reviewCommand,
		cfg.ResolvedModel(),
		prInfo,
	)
	if err != nil {
		return err
	}

	// Post comment and submit verdict
	return submitBitbucketReview(
		ctx,
		commentOnly,
		result,
		bbClient,
		prInfo,
		reviewText,
		cfg.Bitbucket.Username,
	)
}

// createWorktreeAndFetch creates a worktree and fetches latest changes.
// Returns worktree path and cleanup function.
func createWorktreeAndFetch(
	ctx context.Context,
	verbose bool,
	repoPath, branch string,
	prNumber int,
) (string, func(), error) {
	worktreeManager := git.NewWorktreeManager()

	// Fetch latest changes
	logAlways("fetching latest changes...")
	if err := worktreeManager.Fetch(ctx, repoPath); err != nil {
		return "", nil, errors.Wrap(ctx, err, "fetch failed")
	}

	// Create worktree
	logAlways("creating worktree for branch %s...", branch)
	worktreePath, err := worktreeManager.CreateWorktree(ctx, repoPath, branch, prNumber)
	if err != nil {
		return "", nil, errors.Wrap(ctx, err, "create worktree failed")
	}
	logVerbose(verbose, "created worktree: %s", worktreePath)

	cleanup := func() {
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
	}

	return worktreePath, cleanup, nil
}

// runReview executes the Claude review and returns the review text and verdict.
func runReview(
	ctx context.Context,
	reviewer review.Reviewer,
	worktreePath, reviewCommand, model string,
	prInfo *prurl.PRInfo,
) (string, verdict.Result, error) {
	// Run review
	logAlways(
		"reviewing PR #%d (%s/%s) (this may take a few minutes)...",
		prInfo.Number,
		prInfo.Owner,
		prInfo.Repo,
	)
	reviewText, err := reviewer.Review(ctx, worktreePath, reviewCommand, model)
	if err != nil {
		return "", verdict.Result{}, errors.Wrap(ctx, err, "review failed")
	}

	// Always print review to stdout
	fmt.Println(reviewText)

	// Parse verdict
	result := verdict.Parse(reviewText)
	logAlways("verdict: %s (%s)", result.Verdict, result.Reason)

	return reviewText, result, nil
}

// submitGitHubReview submits the review to GitHub using the appropriate method.
func submitGitHubReview(
	ctx context.Context,
	commentOnly bool,
	result verdict.Result,
	ghClient github.Client,
	prInfo *prurl.PRInfo,
	reviewText string,
) error {
	// --comment-only flag overrides verdict
	if commentOnly {
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

	// Submit structured review for approve/request-changes
	if result.Verdict == verdict.VerdictApprove ||
		result.Verdict == verdict.VerdictRequestChanges {
		logAlways("submitting review: %s...", result.Verdict)
		if err := ghClient.SubmitReview(
			ctx,
			prInfo.Owner,
			prInfo.Repo,
			prInfo.Number,
			reviewText,
			result.Verdict,
		); err != nil {
			return errors.Wrap(ctx, err, "submit review failed")
		}
		logAlways("done")
		return nil
	}

	// Fallback to plain comment for VerdictComment
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

// submitBitbucketReview submits the review to Bitbucket with comment and verdict.
func submitBitbucketReview(
	ctx context.Context,
	commentOnly bool,
	result verdict.Result,
	bbClient bitbucket.Client,
	prInfo *prurl.PRInfo,
	reviewText string,
	username string,
) error {
	// Always post comment first
	logAlways("posting comment...")
	if err := bbClient.PostComment(
		ctx,
		prInfo.Host,
		prInfo.Project,
		prInfo.Repo,
		prInfo.Number,
		reviewText,
	); err != nil {
		return errors.Wrap(ctx, err, "post comment failed")
	}

	// --comment-only flag skips verdict submission
	if commentOnly {
		logAlways("done")
		return nil
	}

	// Submit verdict for approve/request-changes
	if result.Verdict == verdict.VerdictApprove {
		logAlways("approving PR...")
		if err := bbClient.Approve(
			ctx,
			prInfo.Host,
			prInfo.Project,
			prInfo.Repo,
			prInfo.Number,
		); err != nil {
			return errors.Wrap(ctx, err, "approve failed")
		}
		logAlways("done")
		return nil
	}

	if result.Verdict == verdict.VerdictRequestChanges {
		if username == "" {
			logAlways("skipping needs-work verdict (bitbucket.username not configured)")
			logAlways("done")
			return nil
		}
		logAlways("marking PR as needs-work...")
		if err := bbClient.NeedsWork(
			ctx,
			prInfo.Host,
			prInfo.Project,
			prInfo.Repo,
			prInfo.Number,
			username,
		); err != nil {
			return errors.Wrap(ctx, err, "needs-work failed")
		}
		logAlways("done")
		return nil
	}

	// VerdictComment - no verdict action needed
	logAlways("done")
	return nil
}

// buildReviewCommand constructs the review command with the target branch.
// If reviewCommand is set (custom override), use it as-is.
// Otherwise, construct "/pr-review <targetBranch>".
func buildReviewCommand(reviewCommand, targetBranch string) string {
	if reviewCommand != "" {
		return reviewCommand
	}
	return fmt.Sprintf("/pr-review %s", targetBranch)
}

// logTokenStatus logs the token source and whether it resolved to a value.
func logTokenStatus(verbose bool, label, configToken, resolvedToken, defaultToken string) {
	source := configToken
	if source == "" {
		source = defaultToken
	}
	if resolvedToken == "" {
		logVerbose(verbose, "%s: %s (not set, using default auth)", label, source)
	} else {
		logVerbose(verbose, "%s: %s (set, %d chars)", label, source, len(resolvedToken))
	}
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
