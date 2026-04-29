// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package git_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/code-reviewer/agent/pr-reviewer/pkg/git"
)

var _ = Describe("ParseCloneURL", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	DescribeTable(
		"valid URLs",
		func(rawURL, expected string) {
			result, err := git.ParseCloneURL(ctx, rawURL)
			Expect(err).To(BeNil())
			Expect(result).To(Equal(expected))
		},
		Entry(
			"URL with .git suffix",
			"https://github.com/bborbe/code-reviewer.git",
			"github.com/bborbe/code-reviewer.git",
		),
		Entry(
			"URL without .git suffix",
			"https://github.com/bborbe/code-reviewer",
			"github.com/bborbe/code-reviewer.git",
		),
	)

	DescribeTable("invalid URLs",
		func(rawURL string) {
			_, err := git.ParseCloneURL(ctx, rawURL)
			Expect(err).NotTo(BeNil())
		},
		Entry("empty string", ""),
		Entry("no host", "/owner/repo.git"),
		Entry("path traversal in owner (2-segment)", "https://github.com/../repo.git"),
		Entry("path traversal in owner (3-segment)", "https://github.com/../foo/bar.git"),
		Entry("dot in owner", "https://github.com/./repo.git"),
		Entry("empty segment from double slash", "https://github.com//repo.git"),
		Entry("only one path segment", "https://github.com/owner"),
		Entry("three path segments", "https://github.com/owner/repo/extra"),
		Entry("shell metacharacter in repo", "https://github.com/owner/repo;rm -rf /"),
	)
})
