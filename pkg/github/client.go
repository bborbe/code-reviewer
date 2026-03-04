// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Client interacts with GitHub via the gh CLI.
//
//counterfeiter:generate -o ../../mocks/github-client.go --fake-name GitHubClient . Client
type Client interface {
	GetPRBranch(ctx context.Context, owner, repo string, number int) (string, error)
	PostComment(ctx context.Context, owner, repo string, number int, body string) error
}

// NewGHClient creates a Client that uses the gh CLI.
func NewGHClient() Client {
	return &ghClient{}
}

type ghClient struct{}

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

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh pr comment failed: %s", strings.TrimSpace(stderr.String()))
	}

	return nil
}
