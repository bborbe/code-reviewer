// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package factory wires concrete dependencies for the github-pr-watcher binary.
package factory

import (
	"context"
	"strings"
	"time"

	"github.com/bborbe/cqrs/base"
	"github.com/bborbe/cqrs/cdb"
	"github.com/bborbe/errors"
	libkafka "github.com/bborbe/kafka"
	"github.com/bborbe/log"

	"github.com/bborbe/code-reviewer/watcher/github/pkg"
)

// CreateKafkaPublisher constructs a CommandPublisher backed by a Kafka sync producer.
// The cleanup function closes the underlying sync producer on shutdown.
func CreateKafkaPublisher(
	ctx context.Context,
	brokers libkafka.Brokers,
	branch base.Branch,
) (pkg.CommandPublisher, func(), error) {
	syncProducer, err := libkafka.NewSyncProducerWithName(ctx, brokers, "github-pr-watcher")
	if err != nil {
		return nil, nil, errors.Wrap(ctx, err, "create sync producer")
	}
	sender := cdb.NewCommandObjectSender(syncProducer, branch, log.DefaultSamplerFactory)
	cleanup := func() {
		if err := syncProducer.Close(); err != nil {
			_ = err
		}
	}
	return pkg.NewCommandPublisher(sender), cleanup, nil
}

// CreateWatcher wires all dependencies and returns a ready-to-use Watcher.
func CreateWatcher(
	ctx context.Context,
	ghToken string,
	brokers libkafka.Brokers,
	stage string,
	repoScope string,
	botAllowlist []string,
	pollInterval time.Duration,
	startTime time.Time,
) (pkg.Watcher, func(), error) {
	branch := base.Branch(stage)
	pub, cleanup, err := CreateKafkaPublisher(ctx, brokers, branch)
	if err != nil {
		return nil, nil, errors.Wrap(ctx, err, "create kafka publisher")
	}

	ghClient := pkg.NewGitHubClient(ghToken)
	w := pkg.NewWatcher(
		ghClient,
		pub,
		pkg.DefaultCursorPath,
		startTime,
		repoScope,
		botAllowlist,
		stage,
	)
	return w, cleanup, nil
}

// ParseBotAllowlist splits a comma-separated allowlist string into a slice.
func ParseBotAllowlist(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
