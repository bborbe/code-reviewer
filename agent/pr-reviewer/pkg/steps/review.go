// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package steps holds custom agent steps for agent-pr-reviewer.
//
// claude.NewAgentStep covers the planning + execution phases (single
// Claude call per phase, fixed NextPhase). The ai_review phase needs a
// conditional NextPhase based on the parsed verdict ("pass" → done,
// anything else → human_review), so it lives here as a custom step.
package steps

import (
	"context"
	"encoding/json"
	"fmt"

	agentlib "github.com/bborbe/agent/lib"
	claudelib "github.com/bborbe/agent/lib/claude"
	"github.com/bborbe/errors"
)

// verdictPayload is the parsed shape of the ## Verdict JSON the ai_review
// step writes. Only the fields needed for next-phase routing are typed
// here; the full payload stays in the markdown body for humans.
type verdictPayload struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// reviewStep runs Claude on the task with the review-phase prompt, writes
// the LLM's response under ## Verdict, parses verdict, and routes the
// next phase: pass → done, fail (or unparseable) → human_review.
type reviewStep struct {
	runner       claudelib.ClaudeRunner
	instructions claudelib.Instructions
}

// NewReviewStep constructs the ai_review-phase step.
func NewReviewStep(
	runner claudelib.ClaudeRunner,
	instructions claudelib.Instructions,
) agentlib.Step {
	return &reviewStep{runner: runner, instructions: instructions}
}

// Name implements agentlib.Step.
func (s *reviewStep) Name() string { return "pr-ai-review" }

// ShouldRun returns false if ## Verdict already exists (idempotent).
func (s *reviewStep) ShouldRun(_ context.Context, md *agentlib.Markdown) (bool, error) {
	_, exists := md.FindSection("## Verdict")
	return !exists, nil
}

// Run calls Claude with the task body (which includes ## Plan + ## Review
// from earlier phases), writes ## Verdict, parses the verdict, and
// returns Done with conditional NextPhase.
func (s *reviewStep) Run(ctx context.Context, md *agentlib.Markdown) (*agentlib.Result, error) {
	taskContent, err := md.Marshal(ctx)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "ai-review marshal task")
	}

	prompt := claudelib.BuildPrompt(s.instructions.String(), nil, taskContent)

	runResult, runErr := s.runner.Run(ctx, prompt)
	if runErr != nil {
		return &agentlib.Result{
			Status:  agentlib.AgentStatusFailed,
			Message: fmt.Sprintf("ai-review claude run failed: %v", runErr),
		}, nil
	}

	md.ReplaceSection(agentlib.Section{
		Heading: "## Verdict",
		Body:    runResult.Result,
	})

	var verdict verdictPayload
	if err := json.Unmarshal([]byte(runResult.Result), &verdict); err != nil {
		return &agentlib.Result{
			Status:    agentlib.AgentStatusDone,
			NextPhase: "human_review",
			Message:   fmt.Sprintf("ai-review wrote ## Verdict but JSON unparseable: %v", err),
		}, nil
	}

	if verdict.Verdict == "pass" {
		return &agentlib.Result{
			Status:    agentlib.AgentStatusDone,
			NextPhase: "done",
			Message:   verdict.Reason,
		}, nil
	}

	return &agentlib.Result{
		Status:    agentlib.AgentStatusDone,
		NextPhase: "human_review",
		Message:   fmt.Sprintf("ai-review verdict=%s: %s", verdict.Verdict, verdict.Reason),
	}, nil
}
