// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package plugins provides a reusable Claude plugin installer library.
package plugins

import (
	"bytes"
	"context"
	"os/exec"
	"path"
	"strings"

	"github.com/bborbe/errors"
	"github.com/golang/glog"
)

// Spec identifies a Claude Code plugin to ensure is installed.
type Spec struct {
	Marketplace string // e.g. "bborbe/coding"
	Name        string // e.g. "coding"
}

// Commander runs an external command and returns its combined stdout.
//
//counterfeiter:generate -o mocks/commander.go --fake-name Commander . Commander
type Commander interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// Installer ensures a list of Claude plugins are installed or updated.
//
//counterfeiter:generate -o mocks/installer.go --fake-name Installer . Installer
type Installer interface {
	EnsureInstalled(ctx context.Context, specs []Spec) error
}

// NewExecCommander returns a Commander that uses os/exec to run real processes.
func NewExecCommander() Commander {
	return &execCommander{}
}

type execCommander struct{}

func (e *execCommander) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(
		ctx,
		name,
		args...) // #nosec G204 -- name and args are caller-controlled; Commander is an internal interface not exposed to untrusted input
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return "", errors.Wrapf(
			ctx,
			err,
			"run %s %s: %s",
			name,
			strings.Join(args, " "),
			errOut.String(),
		)
	}
	return out.String(), nil
}

// NewInstaller returns an Installer that uses the given Commander to manage Claude plugins.
func NewInstaller(commander Commander) Installer {
	return &installer{commander: commander}
}

type installer struct {
	commander Commander
}

func (i *installer) EnsureInstalled(ctx context.Context, specs []Spec) error {
	if len(specs) == 0 {
		return nil
	}
	for _, spec := range specs {
		if err := i.ensureOne(ctx, spec); err != nil {
			return err
		}
	}
	return nil
}

func (i *installer) ensureOne(ctx context.Context, spec Spec) error {
	alias := path.Base(spec.Marketplace)
	updateForm := spec.Name + "@" + alias

	output, err := i.commander.Run(ctx, "claude", "plugin", "list")
	if err != nil {
		return errors.Wrapf(ctx, err, "list plugins")
	}

	installed := false
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, spec.Name) {
			installed = true
			break
		}
	}

	if !installed {
		if err := i.runHard(ctx, "claude", "plugin", "marketplace", "add", spec.Marketplace); err != nil {
			return err
		}
		if err := i.runHard(ctx, "claude", "plugin", "install", spec.Name); err != nil {
			return err
		}
		return nil
	}

	if _, err := i.commander.Run(ctx, "claude", "plugin", "marketplace", "update", alias); err != nil {
		glog.Warningf(
			"marketplace update failed plugin=%s cmd=%s err=%v",
			spec.Name,
			"claude plugin marketplace update "+alias,
			err,
		)
	}
	if _, err := i.commander.Run(ctx, "claude", "plugin", "update", updateForm); err != nil {
		glog.Warningf(
			"plugin update failed plugin=%s cmd=%s err=%v",
			spec.Name,
			"claude plugin update "+updateForm,
			err,
		)
	}
	return nil
}

func (i *installer) runHard(ctx context.Context, name string, args ...string) error {
	_, err := i.commander.Run(ctx, name, args...)
	if err != nil {
		return errors.Wrapf(ctx, err, "run %s %s", name, strings.Join(args, " "))
	}
	return nil
}
