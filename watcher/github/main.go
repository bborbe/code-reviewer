// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command github-pr-watcher polls GitHub for open pull requests in
// configured repos and publishes a CreateTaskCommand to Kafka per new
// PR so the existing pr-reviewer agent picks it up automatically.
package main

import (
	"context"
	"os"

	"github.com/bborbe/errors"
	libkafka "github.com/bborbe/kafka"
	libsentry "github.com/bborbe/sentry"
	"github.com/bborbe/service"
	"github.com/golang/glog"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN   string `required:"false" arg:"sentry-dsn"   env:"SENTRY_DSN"   usage:"SentryDSN"    display:"length"`
	SentryProxy string `required:"false" arg:"sentry-proxy" env:"SENTRY_PROXY" usage:"Sentry Proxy"`

	GHToken      string           `required:"true"  arg:"gh-token"      env:"GH_TOKEN"      usage:"GitHub token (read scope sufficient)" display:"length"`
	KafkaBrokers libkafka.Brokers `required:"true"  arg:"kafka-brokers" env:"KAFKA_BROKERS" usage:"Comma-separated Kafka broker list"`
	Stage        string           `required:"true"  arg:"stage"         env:"STAGE"         usage:"Deployment stage (dev|prod)"`
	PollInterval string           `required:"false" arg:"poll-interval" env:"POLL_INTERVAL" usage:"Poll interval (Go duration)"                           default:"5m"`
	RepoScope    string           `required:"false" arg:"repo-scope"    env:"REPO_SCOPE"    usage:"GitHub user/org scope"                                 default:"bborbe"`
	BotAllowlist string           `required:"false" arg:"bot-allowlist" env:"BOT_ALLOWLIST" usage:"Comma-separated bot author allowlist"                  default:"dependabot[bot],renovate[bot]"`
}

func (a *application) Run(ctx context.Context, _ libsentry.Client) error {
	glog.V(2).
		Infof("github-pr-watcher started stage=%s scope=%s interval=%s", a.Stage, a.RepoScope, a.PollInterval)

	// TODO(spec-009 prompt 3): replace with real poll loop wiring
	// For the scaffold prompt, we just block on ctx so the binary
	// shape exists for sibling prompts to extend.
	<-ctx.Done()
	return errors.Wrap(ctx, ctx.Err(), "context cancelled")
}
