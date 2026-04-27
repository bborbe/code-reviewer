// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package factory wires concrete dependencies for the agent-pr-reviewer binary.
//
// All factory functions follow the Create* prefix convention and contain
// zero business logic — they compose constructors with config.
package factory

import (
	"context"

	agentlib "github.com/bborbe/agent/lib"
	claudelib "github.com/bborbe/agent/lib/claude"
	delivery "github.com/bborbe/agent/lib/delivery"
	"github.com/bborbe/cqrs/base"
	"github.com/bborbe/errors"
	libkafka "github.com/bborbe/kafka"
	libtime "github.com/bborbe/time"
	"github.com/golang/glog"

	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/prompts/execution"
	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/prompts/planning"
	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/prompts/review"
	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/steps"
)

const serviceName = "agent-pr-reviewer"

// Per-phase tool scopes. Principle: each phase gets the smallest set that
// lets it do its job. Planning + Review are read-only inspection. Execution
// gets broader git/gh access for cross-file reads but still cannot post
// (no `gh pr comment` / `gh pr review`) — posting happens out-of-band
// after the human approves the verdict.
var (
	planningTools = claudelib.AllowedTools{
		"Read", "Grep", "Glob",
		"Bash(git diff:*)", "Bash(git log:*)", "Bash(git show:*)",
		"Bash(gh pr view:*)", "Bash(gh pr diff:*)", "Bash(gh pr list:*)",
	}
	executionTools = claudelib.AllowedTools{
		"Read", "Grep", "Glob",
		"Bash(git:*)",
		"Bash(gh pr view:*)", "Bash(gh pr diff:*)", "Bash(gh pr list:*)",
		"WebFetch",
	}
	reviewTools = claudelib.AllowedTools{
		"Read", "Grep",
		"Bash(gh pr view:*)", "Bash(gh pr diff:*)",
	}
)

// CreateClaudeRunner constructs a ClaudeRunner pre-configured with tools,
// model, working directory, and CLI environment. ghToken is forwarded as
// GH_TOKEN into the Claude CLI subprocess env so the gh CLI can authenticate.
func CreateClaudeRunner(
	claudeConfigDir claudelib.ClaudeConfigDir,
	agentDir claudelib.AgentDir,
	model claudelib.ClaudeModel,
	ghToken string,
	allowedTools claudelib.AllowedTools,
) claudelib.ClaudeRunner {
	env := map[string]string{}
	if ghToken != "" {
		env["GH_TOKEN"] = ghToken
	}
	return claudelib.NewClaudeRunner(claudelib.ClaudeRunnerConfig{
		ClaudeConfigDir:  claudeConfigDir,
		AllowedTools:     allowedTools,
		Model:            model,
		WorkingDirectory: agentDir,
		Env:              env,
	})
}

// CreateSyncProducer creates a Kafka sync producer.
func CreateSyncProducer(
	ctx context.Context,
	brokers libkafka.Brokers,
) (libkafka.SyncProducer, error) {
	producer, err := libkafka.NewSyncProducerWithName(ctx, brokers, serviceName)
	if err != nil {
		return nil, errors.Wrap(ctx, err, "create sync producer failed")
	}
	return producer, nil
}

// CreateKafkaResultDeliverer creates a ResultDeliverer that publishes task
// updates to Kafka via CQRS commands. Uses the passthrough content generator
// — the agent framework's StepRunner already produces the full marshaled
// task in result.Output; the deliverer publishes it as-is.
func CreateKafkaResultDeliverer(
	syncProducer libkafka.SyncProducer,
	branch base.Branch,
	taskID agentlib.TaskIdentifier,
	originalContent string,
	currentDateTime libtime.CurrentDateTimeGetter,
) agentlib.ResultDeliverer {
	return delivery.NewKafkaResultDeliverer(
		syncProducer,
		branch,
		taskID,
		originalContent,
		delivery.NewPassthroughContentGenerator(),
		currentDateTime,
	)
}

// CreateFileResultDeliverer creates a ResultDeliverer that writes the agent's
// output back to a markdown file (local CLI mode).
func CreateFileResultDeliverer(filePath string) agentlib.ResultDeliverer {
	return delivery.NewFileResultDeliverer(
		delivery.NewPassthroughContentGenerator(),
		filePath,
	)
}

// CreateAgent assembles the full 3-phase pr-reviewer agent with per-phase
// tool scopes and per-phase prompts:
//
//   - planning: read-only diff inspection → ## Plan (JSON)
//   - in_progress: read + cross-file inspection → ## Review (JSON)
//   - ai_review: minimal read-only fresh-context verifier → ## Verdict (JSON);
//     verdict=pass → done, otherwise → human_review
func CreateAgent(
	claudeConfigDir claudelib.ClaudeConfigDir,
	agentDir claudelib.AgentDir,
	model claudelib.ClaudeModel,
	ghToken string,
) *agentlib.Agent {
	planningStep := claudelib.NewAgentStep(claudelib.AgentStepConfig{
		Name:          "pr-plan",
		Runner:        CreateClaudeRunner(claudeConfigDir, agentDir, model, ghToken, planningTools),
		Instructions:  planning.BuildInstructions(),
		OutputSection: "## Plan",
		NextPhase:     "in_progress",
	})
	executionStep := claudelib.NewAgentStep(claudelib.AgentStepConfig{
		Name: "pr-execute",
		Runner: CreateClaudeRunner(
			claudeConfigDir,
			agentDir,
			model,
			ghToken,
			executionTools,
		),
		Instructions:  execution.BuildInstructions(),
		OutputSection: "## Review",
		NextPhase:     "ai_review",
	})
	reviewStep := steps.NewReviewStep(
		CreateClaudeRunner(claudeConfigDir, agentDir, model, ghToken, reviewTools),
		review.BuildInstructions(),
	)
	return agentlib.NewAgent(
		agentlib.NewPhase("planning", planningStep),
		agentlib.NewPhase("in_progress", executionStep),
		agentlib.NewPhase("ai_review", reviewStep),
	)
}

// CreateDeliverer builds the Kafka-or-Noop deliverer used by the Kafka
// entry point. Empty taskID means "no Kafka" — returns a noop deliverer
// and an empty cleanup. Non-empty taskID requires non-empty brokers; the
// returned cleanup closes the underlying SyncProducer (logged-and-ignored
// on error).
func CreateDeliverer(
	ctx context.Context,
	taskID agentlib.TaskIdentifier,
	brokers libkafka.Brokers,
	branch base.Branch,
	originalContent string,
) (agentlib.ResultDeliverer, func(), error) {
	if taskID == "" {
		glog.V(2).Infof("TASK_ID not set, skipping task result publishing")
		return delivery.NewNoopResultDeliverer(), func() {}, nil
	}
	if len(brokers) == 0 {
		return nil, nil, errors.Errorf(ctx, "KAFKA_BROKERS must be set when TASK_ID is set")
	}
	syncProducer, err := CreateSyncProducer(ctx, brokers)
	if err != nil {
		return nil, nil, errors.Wrap(ctx, err, "create sync producer failed")
	}
	deliverer := CreateKafkaResultDeliverer(
		syncProducer,
		branch,
		taskID,
		originalContent,
		libtime.NewCurrentDateTime(),
	)
	cleanup := func() {
		if err := syncProducer.Close(); err != nil {
			glog.Warningf("close sync producer failed: %v", err)
		}
	}
	return deliverer, cleanup, nil
}
