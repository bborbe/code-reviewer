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

	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/prompts"
)

const serviceName = "agent-pr-reviewer"

// allowedTools pins the Claude tools pr-reviewer needs: read files, search,
// invoke git/gh for PR inspection, and fetch web content.
var allowedTools = claudelib.AllowedTools{
	"Read", "Grep", "Glob", "Bash(git:*)", "Bash(gh:*)", "WebFetch",
}

// CreateClaudeRunner constructs a ClaudeRunner pre-configured with tools,
// model, working directory, and CLI environment. ghToken is forwarded as
// GH_TOKEN into the Claude CLI subprocess env so the gh CLI can authenticate.
func CreateClaudeRunner(
	claudeConfigDir claudelib.ClaudeConfigDir,
	agentDir claudelib.AgentDir,
	model claudelib.ClaudeModel,
	ghToken string,
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

// CreateAgent assembles the full 3-phase pr-reviewer agent. Single Claude
// step shared across planning / in_progress / ai_review preserves the
// existing CRD trigger.phases behavior — every phase runs Claude once and
// emits done.
func CreateAgent(
	claudeConfigDir claudelib.ClaudeConfigDir,
	agentDir claudelib.AgentDir,
	model claudelib.ClaudeModel,
	ghToken string,
) *agentlib.Agent {
	runner := CreateClaudeRunner(claudeConfigDir, agentDir, model, ghToken)
	step := claudelib.NewAgentStep(claudelib.AgentStepConfig{
		Name:          "pr-review",
		Runner:        runner,
		Instructions:  prompts.BuildInstructions(),
		OutputSection: "## Review",
		NextPhase:     "done",
	})
	return agentlib.NewAgent(
		agentlib.NewPhase("planning", step),
		agentlib.NewPhase("in_progress", step),
		agentlib.NewPhase("ai_review", step),
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
