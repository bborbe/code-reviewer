// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package review embeds the ai_review-phase prompt for agent-pr-reviewer.
package review

import (
	_ "embed"

	claudelib "github.com/bborbe/agent/lib/claude"
)

//go:embed workflow.md
var workflow string

//go:embed output-format.md
var outputFormat string

// BuildInstructions assembles the ai_review-phase prompt from embedded modules.
func BuildInstructions() claudelib.Instructions {
	return claudelib.Instructions{
		{Name: "workflow", Content: workflow},
		{Name: "output-format", Content: outputFormat},
	}
}
