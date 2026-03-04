// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bborbe/pr-reviewer/pkg/verdict"
)

// Client interacts with GitHub via the gh CLI.
//
//counterfeiter:generate -o ../../mocks/github-client.go --fake-name GitHubClient . Client
type Client interface {
	GetPRBranch(ctx context.Context, owner, repo string, number int) (string, error)
	PostComment(ctx context.Context, owner, repo string, number int, body string) error
	SubmitReview(
		ctx context.Context,
		owner, repo string,
		number int,
		body string,
		verdict verdict.Verdict,
	) error
}

// NewGHClient creates a Client that uses the gh CLI.
// If token is non-empty, it will be passed to gh CLI via GH_TOKEN env var.
func NewGHClient(token string) Client {
	return &ghClient{token: token}
}

type ghClient struct {
	token string
}

// GetPRBranch fetches the source branch name (headRefName) for a pull request.
func (c *ghClient) GetPRBranch(
	ctx context.Context,
	owner, repo string,
	number int,
) (string, error) {
	repoArg := fmt.Sprintf("%s/%s", owner, repo)
	numberArg := strconv.Itoa(number)

	// #nosec G204 -- args are validated by caller, owner/repo from URL parsing
	cmd := exec.CommandContext(ctx, "gh", "pr", "view",
		numberArg,
		"--repo", repoArg,
		"--json", "headRefName",
		"--jq", ".headRefName",
	)

	// Set GH_TOKEN if configured
	if c.token != "" {
		cmd.Env = append(os.Environ(), "GH_TOKEN="+c.token)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh pr view failed: %s", strings.TrimSpace(stderr.String()))
	}

	branch := strings.TrimSpace(stdout.String())
	if branch == "" {
		return "", fmt.Errorf("gh pr view returned empty branch name")
	}

	return branch, nil
}

// PostComment posts a comment on a pull request.
func (c *ghClient) PostComment(
	ctx context.Context,
	owner, repo string,
	number int,
	body string,
) error {
	repoArg := fmt.Sprintf("%s/%s", owner, repo)
	numberArg := strconv.Itoa(number)

	// #nosec G204 -- args are validated by caller, owner/repo from URL parsing
	cmd := exec.CommandContext(ctx, "gh", "pr", "comment",
		numberArg,
		"--repo", repoArg,
		"--body", body,
	)

	// Set GH_TOKEN if configured
	if c.token != "" {
		cmd.Env = append(os.Environ(), "GH_TOKEN="+c.token)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr comment failed: %s", strings.TrimSpace(stderr.String()))
	}

	return nil
}

// SubmitReview submits a structured review (approve or request-changes) on a pull request.
func (c *ghClient) SubmitReview(
	ctx context.Context,
	owner, repo string,
	number int,
	body string,
	v verdict.Verdict,
) error {
	// Only approve and request-changes are supported
	// For comment verdict, caller should use PostComment instead
	if v != verdict.VerdictApprove && v != verdict.VerdictRequestChanges {
		return fmt.Errorf("unsupported verdict for SubmitReview: %s (use PostComment instead)", v)
	}

	repoArg := fmt.Sprintf("%s/%s", owner, repo)
	numberArg := strconv.Itoa(number)

	args := []string{"pr", "review", numberArg, "--repo", repoArg}

	if v == verdict.VerdictApprove {
		args = append(args, "--approve")
	} else {
		args = append(args, "--request-changes")
	}

	args = append(args, "--body", body)

	// #nosec G204 -- args are validated by caller, owner/repo from URL parsing
	cmd := exec.CommandContext(ctx, "gh", args...)

	// Set GH_TOKEN if configured
	if c.token != "" {
		cmd.Env = append(os.Environ(), "GH_TOKEN="+c.token)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr review failed: %s", strings.TrimSpace(stderr.String()))
	}

	return nil
}
