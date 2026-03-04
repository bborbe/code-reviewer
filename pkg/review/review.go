// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package review

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Reviewer runs a code review using the claude CLI.
//
//counterfeiter:generate -o ../../mocks/reviewer.go --fake-name Reviewer . Reviewer
type Reviewer interface {
	Review(ctx context.Context, worktreePath string) (string, error)
}

// NewClaudeReviewer creates a Reviewer that invokes the claude CLI.
func NewClaudeReviewer() Reviewer {
	return &claudeReviewer{}
}

type claudeReviewer struct{}

// Review runs 'claude --print "/code-review"' in the worktree directory.
// Returns the review text from stdout on success.
// Returns an error if claude is not in PATH or exits non-zero.
func (r *claudeReviewer) Review(ctx context.Context, worktreePath string) (string, error) {
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		return "", fmt.Errorf("claude not found in PATH")
	}

	// #nosec G204 -- claudePath verified by LookPath, args are hardcoded flags
	cmd := exec.CommandContext(ctx, claudePath, "--print", "/code-review")
	cmd.Dir = worktreePath

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude review failed: %s", strings.TrimSpace(stderr.String()))
	}

	return stdout.String(), nil
}
