// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package prompts

import (
	_ "embed"

	claudelib "github.com/bborbe/agent/lib/claude"
)

//go:embed execution_workflow.md
var executionWorkflow string

//go:embed execution_output-format.md
var executionOutputFormat string

// BuildExecutionInstructions assembles the execution-phase prompt from embedded modules.
func BuildExecutionInstructions() claudelib.Instructions {
	return claudelib.Instructions{
		{Name: "workflow", Content: executionWorkflow},
		{Name: "output-format", Content: executionOutputFormat},
	}
}
