// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command agent-pr-reviewer is the Kafka entry point for the PR-review
// agent — spawned as a K8s Job by task/executor with TASK_CONTENT +
// TASK_ID + PHASE + KAFKA_BROKERS env. For local CLI mode (file-based),
// see cmd/run-task/main.go.
package main

import (
	"context"
	"os"

	agentlib "github.com/bborbe/agent/lib"
	claudelib "github.com/bborbe/agent/lib/claude"
	"github.com/bborbe/cqrs/base"
	"github.com/bborbe/errors"
	libkafka "github.com/bborbe/kafka"
	libsentry "github.com/bborbe/sentry"
	"github.com/bborbe/service"
	"github.com/bborbe/vault-cli/pkg/domain"
	"github.com/golang/glog"

	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/factory"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN   string `required:"false" arg:"sentry-dsn"   env:"SENTRY_DSN"   usage:"SentryDSN"    display:"length"`
	SentryProxy string `required:"false" arg:"sentry-proxy" env:"SENTRY_PROXY" usage:"Sentry Proxy"`

	// Claude Code CLI configuration
	ClaudeConfigDir claudelib.ClaudeConfigDir `required:"false" arg:"claude-config-dir" env:"CLAUDE_CONFIG_DIR" usage:"Claude Code config directory"`

	// Agent directory (contains .claude/ with CLAUDE.md and commands)
	AgentDir claudelib.AgentDir `required:"false" arg:"agent-dir" env:"AGENT_DIR" usage:"Agent directory with .claude/ config" default:"agent"`

	// Model selection
	Model claudelib.ClaudeModel `required:"false" arg:"model" env:"MODEL" usage:"Claude model to use (sonnet, opus)" default:"sonnet"`

	// Task content from agent pipeline
	TaskContent string `required:"true" arg:"task-content" env:"TASK_CONTENT" usage:"Raw task markdown from vault"`

	// Branch for Kafka result delivery
	Branch base.Branch `required:"true" arg:"branch" env:"BRANCH" usage:"branch"`

	// Phase to run (framework requires explicit phase)
	Phase domain.TaskPhase `required:"false" arg:"phase" env:"PHASE" usage:"Agent phase: planning | in_progress | ai_review" default:"in_progress"`

	// Kafka delivery (optional — only active when TASK_ID is set)
	KafkaBrokers libkafka.Brokers        `required:"false" arg:"kafka-brokers" env:"KAFKA_BROKERS" usage:"Comma separated list of Kafka brokers"`
	TaskID       agentlib.TaskIdentifier `required:"false" arg:"task-id"       env:"TASK_ID"       usage:"Agent task identifier for publishing results back to task controller"`

	// GitHub token forwarded to the Claude CLI subprocess as GH_TOKEN for gh auth.
	GHToken string `required:"false" arg:"gh-token" env:"GH_TOKEN" usage:"GitHub token for gh CLI auth" display:"length"`
}

func (a *application) Run(ctx context.Context, _ libsentry.Client) error {
	glog.V(2).Infof("agent-pr-reviewer started phase=%s", a.Phase)

	deliverer, cleanup, err := factory.CreateDeliverer(
		ctx, a.TaskID, a.KafkaBrokers, a.Branch, a.TaskContent,
	)
	if err != nil {
		return errors.Wrap(ctx, err, "create deliverer")
	}
	defer cleanup()

	agent := factory.CreateAgent(a.ClaudeConfigDir, a.AgentDir, a.Model, a.GHToken)

	result, err := agent.Run(ctx, a.Phase, a.TaskContent, deliverer)
	if err != nil {
		return errors.Wrap(ctx, err, "agent run failed")
	}
	return agentlib.PrintResult(result)
}
