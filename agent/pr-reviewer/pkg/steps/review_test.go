// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package steps_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/steps"
)

var _ = Describe("ExtractVerdict", func() {
	DescribeTable("parses verdict from various LLM response shapes",
		func(input, wantVerdict, wantReason string, wantOK bool) {
			got, err := steps.ExtractVerdictForTest(input)
			if !wantOK {
				Expect(err).To(HaveOccurred())
				return
			}
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Verdict).To(Equal(wantVerdict))
			Expect(got.Reason).To(Equal(wantReason))
		},

		Entry("raw JSON object",
			`{"verdict":"pass","reason":"all good"}`,
			"pass", "all good", true),

		Entry("JSON with leading + trailing whitespace",
			"\n\n  {\"verdict\":\"fail\",\"reason\":\"bad\"}  \n",
			"fail", "bad", true),

		Entry("JSON wrapped in ```json fence",
			"```json\n{\"verdict\":\"pass\",\"reason\":\"x\"}\n```",
			"pass", "x", true),

		Entry("JSON wrapped in plain ``` fence",
			"```\n{\"verdict\":\"fail\",\"reason\":\"y\"}\n```",
			"fail", "y", true),

		Entry(
			"prose before JSON (Claude commentary)",
			"All three checks pass:\n\n1. Concerns addressed\n2. No hallucinations\n3. Consistent\n\n{\"verdict\":\"pass\",\"reason\":\"all good\"}",
			"pass",
			"all good",
			true,
		),

		Entry("prose before AND after JSON",
			"Reasoning here.\n\n{\"verdict\":\"pass\",\"reason\":\"ok\"}\n\nFurther explanation.",
			"pass", "ok", true),

		Entry("multiple JSON-like fragments — picks the last balanced block",
			"Ignored: {\"foo\":\"bar\"}\n\nFinal: {\"verdict\":\"fail\",\"reason\":\"z\"}",
			"fail", "z", true),

		Entry("nested objects in the verdict JSON are preserved",
			"```json\n{\"verdict\":\"fail\",\"reason\":\"nested\",\"detail\":{\"a\":1}}\n```",
			"fail", "nested", true),

		Entry("empty string fails",
			"", "", "", false),

		Entry("prose only without any JSON fails",
			"This is just prose with no braces.", "", "", false),

		Entry("malformed JSON with unbalanced braces fails",
			"oops {{{", "", "", false),
	)
})
