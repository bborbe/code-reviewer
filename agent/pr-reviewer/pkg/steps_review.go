// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

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

	verdict, err := extractVerdict(ctx, runResult.Result)
	if err != nil {
		return &agentlib.Result{
			Status:    agentlib.AgentStatusDone,
			NextPhase: "human_review",
			Message:   fmt.Sprintf("ai-review wrote ## Verdict but verdict unparseable: %v", err),
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

// extractVerdict parses the verdict from the LLM's response. The prompt
// asks for raw JSON only, but Claude sometimes prefixes the JSON with
// prose explanation. To be tolerant, we (1) try direct unmarshal of the
// trimmed response, then (2) strip ```json fences if present, then
// (3) extract the last balanced {...} block from the response.
func extractVerdict(ctx context.Context, raw string) (verdictPayload, error) {
	trimmed := strings.TrimSpace(raw)

	// 1. Direct attempt.
	var v verdictPayload
	if err := json.Unmarshal([]byte(trimmed), &v); err == nil {
		return v, nil
	}

	// 2. Strip code fences.
	stripped := strings.TrimSpace(strings.TrimSuffix(
		strings.TrimPrefix(strings.TrimPrefix(trimmed, "```json"), "```"),
		"```",
	))
	if err := json.Unmarshal([]byte(stripped), &v); err == nil {
		return v, nil
	}

	// 3. Extract last balanced {...} block.
	block, ok := lastJSONBlock(ctx, trimmed)
	if !ok {
		return verdictPayload{}, errors.Errorf(ctx, "no JSON object found in response")
	}
	if err := json.Unmarshal([]byte(block), &v); err != nil {
		return verdictPayload{}, errors.Wrapf(ctx, err, "extract last JSON block")
	}
	return v, nil
}

// lastJSONBlock returns the last balanced {...} substring in s, or
// "", false if none exists. Walks from the end finding the closing
// brace, then walks back tracking brace depth to find the matching open.
func lastJSONBlock(_ context.Context, s string) (string, bool) {
	end := strings.LastIndex(s, "}")
	if end < 0 {
		return "", false
	}
	depth := 0
	for i := end; i >= 0; i-- {
		switch s[i] {
		case '}':
			depth++
		case '{':
			depth--
			if depth == 0 {
				return s[i : end+1], true
			}
		}
	}
	return "", false
}
