// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package review

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Reviewer runs a code review using the claude CLI.
//
//counterfeiter:generate -o ../../mocks/reviewer.go --fake-name Reviewer . Reviewer
type Reviewer interface {
	Review(ctx context.Context, worktreePath string, command string, model string) (string, error)
}

// NewDockerReviewer creates a Reviewer that invokes claude inside a Docker container.
func NewDockerReviewer(containerImage string) Reviewer {
	return &dockerReviewer{containerImage: containerImage}
}

type dockerReviewer struct {
	containerImage string
}

// Review runs claude inside a Docker container using the claude-yolo image.
// The container mounts the worktree, Claude config, and Go module cache.
// Uses YOLO_PROMPT_FILE pattern from dark-factory to avoid shell escaping issues.
func (r *dockerReviewer) Review(
	ctx context.Context,
	worktreePath string,
	command string,
	model string,
) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	// Write prompt to temp file (same pattern as dark-factory)
	promptFile, err := os.CreateTemp("", "pr-reviewer-prompt-*.md")
	if err != nil {
		return "", fmt.Errorf("create prompt file: %w", err)
	}
	defer func() { _ = os.Remove(promptFile.Name()) }()

	if _, err := promptFile.WriteString(command); err != nil {
		_ = promptFile.Close()
		return "", fmt.Errorf("write prompt file: %w", err)
	}
	if err := promptFile.Close(); err != nil {
		return "", fmt.Errorf("close prompt file: %w", err)
	}

	// Build docker command
	// Uses entrypoint.sh which reads YOLO_PROMPT_FILE, sets up firewall/proxy,
	// and runs claude. Output is stream-json piped through stream-formatter.py.
	// The formatter extracts text content from assistant messages and prints the
	// final result after "--- DONE ---".
	// #nosec G204 -- paths from context, containerImage from config
	cmd := exec.CommandContext(
		ctx,
		"docker",
		"run",
		"--rm",
		"--cap-add=NET_ADMIN",
		"--cap-add=NET_RAW",
		"-w", "/workspace",
		"-e", "YOLO_PROMPT_FILE=/tmp/prompt.md",
		"-e", "YOLO_MODEL="+model,
		"-v", promptFile.Name()+":/tmp/prompt.md:ro",
		"-v", fmt.Sprintf("%s:/workspace", worktreePath),
		"-v", fmt.Sprintf("%s/.claude-yolo:/home/node/.claude", home),
		"-v", fmt.Sprintf("%s/go/pkg:/home/node/go/pkg", home),
		r.containerImage,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude review failed: %s", strings.TrimSpace(stderr.String()))
	}

	// Extract the final result from stream-formatter output.
	// The formatter prints "--- DONE ---\n<result>" at the end.
	output := stdout.String()
	return extractResult(output), nil
}

// extractResult extracts the final result text from stream-formatter output.
// The formatter outputs "--- DONE ---\n<result>" when claude finishes.
// If no marker found, returns the full output as fallback.
func extractResult(output string) string {
	const marker = "\n--- DONE ---\n"
	if idx := strings.LastIndex(output, marker); idx != -1 {
		return strings.TrimSpace(output[idx+len(marker):])
	}
	return strings.TrimSpace(output)
}
