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
	"time"

	"github.com/bborbe/errors"
	libhttp "github.com/bborbe/http"
	libkafka "github.com/bborbe/kafka"
	"github.com/bborbe/run"
	libsentry "github.com/bborbe/sentry"
	"github.com/bborbe/service"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/bborbe/code-reviewer/watcher/github/pkg"
	"github.com/bborbe/code-reviewer/watcher/github/pkg/factory"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN   string `required:"false" arg:"sentry-dsn"   env:"SENTRY_DSN"   usage:"SentryDSN"    display:"length"`
	SentryProxy string `required:"false" arg:"sentry-proxy" env:"SENTRY_PROXY" usage:"Sentry Proxy"`

	Listen       string           `required:"false" arg:"listen"        env:"LISTEN"        usage:"HTTP listen address (healthz/readiness/metrics)" default:":9090"`
	GHToken      string           `required:"true"  arg:"gh-token"      env:"GH_TOKEN"      usage:"GitHub token (read scope sufficient)"                                                    display:"length"`
	KafkaBrokers libkafka.Brokers `required:"true"  arg:"kafka-brokers" env:"KAFKA_BROKERS" usage:"Comma-separated Kafka broker list"`
	Stage        string           `required:"true"  arg:"stage"         env:"STAGE"         usage:"Deployment stage (dev|prod)"`
	PollInterval string           `required:"false" arg:"poll-interval" env:"POLL_INTERVAL" usage:"Poll interval (Go duration)"                     default:"5m"`
	RepoScope    string           `required:"false" arg:"repo-scope"    env:"REPO_SCOPE"    usage:"GitHub user/org scope"                           default:"bborbe"`
	BotAllowlist string           `required:"false" arg:"bot-allowlist" env:"BOT_ALLOWLIST" usage:"Comma-separated bot author allowlist"            default:"dependabot[bot],renovate[bot]"`
}

func (a *application) Run(ctx context.Context, _ libsentry.Client) error {
	pollInterval, err := time.ParseDuration(a.PollInterval)
	if err != nil {
		return errors.Wrapf(ctx, err, "parse poll interval %q", a.PollInterval)
	}

	botAllowlist := factory.ParseBotAllowlist(a.BotAllowlist)
	startTime := time.Now().UTC()

	w, cleanup, err := factory.CreateWatcher(
		ctx,
		a.GHToken,
		a.KafkaBrokers,
		a.Stage,
		a.RepoScope,
		botAllowlist,
		pollInterval,
		startTime,
	)
	if err != nil {
		return errors.Wrap(ctx, err, "create watcher")
	}
	defer cleanup()

	glog.V(2).
		Infof("github-pr-watcher starting stage=%s scope=%s interval=%s listen=%s", a.Stage, a.RepoScope, a.PollInterval, a.Listen)

	return run.CancelOnFirstFinish(ctx,
		a.runPollLoop(w, pollInterval),
		a.runHTTPServer(),
	)
}

func (a *application) runPollLoop(w pkg.Watcher, interval time.Duration) run.Func {
	return func(ctx context.Context) error {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				glog.V(2).Infof("poll loop: context cancelled, exiting cleanly")
				return nil
			case <-ticker.C:
				glog.V(2).Infof("poll cycle start stage=%s", a.Stage)
				if err := w.Poll(ctx); err != nil {
					glog.Errorf("poll cycle error: %v", err)
				}
			}
		}
	}
}

func (a *application) runHTTPServer() run.Func {
	return func(ctx context.Context) error {
		router := mux.NewRouter()
		router.Path("/healthz").Handler(libhttp.NewPrintHandler("OK"))
		router.Path("/readiness").Handler(libhttp.NewPrintHandler("OK"))
		router.Path("/metrics").Handler(promhttp.Handler())
		glog.V(2).Infof("http server listening on %s", a.Listen)
		return libhttp.NewServer(a.Listen, router).Run(ctx)
	}
}
