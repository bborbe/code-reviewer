// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filter_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/code-reviewer/watcher/github/pkg/filter"
	"github.com/bborbe/code-reviewer/watcher/github/pkg/githubclient"
)

var _ = Describe("Filter", func() {
	Describe("ShouldSkip", func() {
		DescribeTable("skipping rules",
			func(pr githubclient.PullRequest, allowlist []string, expected bool) {
				Expect(filter.ShouldSkip(pr, allowlist)).To(Equal(expected))
			},
			Entry("draft PR, empty allowlist → skipped",
				githubclient.PullRequest{IsDraft: true, AuthorLogin: "alice"},
				[]string{},
				true,
			),
			Entry("non-draft PR, empty allowlist → not skipped",
				githubclient.PullRequest{IsDraft: false, AuthorLogin: "alice"},
				[]string{},
				false,
			),
			Entry("non-draft PR, author in allowlist → skipped",
				githubclient.PullRequest{IsDraft: false, AuthorLogin: "dependabot[bot]"},
				[]string{"dependabot[bot]"},
				true,
			),
			Entry("non-draft PR, author NOT in allowlist → not skipped",
				githubclient.PullRequest{IsDraft: false, AuthorLogin: "alice"},
				[]string{"dependabot[bot]", "renovate[bot]"},
				false,
			),
			Entry("draft PR, author in allowlist → skipped (both conditions true)",
				githubclient.PullRequest{IsDraft: true, AuthorLogin: "dependabot[bot]"},
				[]string{"dependabot[bot]"},
				true,
			),
			Entry("case sensitivity: Dependabot[bot] does NOT match dependabot[bot]",
				githubclient.PullRequest{IsDraft: false, AuthorLogin: "dependabot[bot]"},
				[]string{"Dependabot[bot]"},
				false,
			),
		)
	})

	Describe("IsBotAuthor", func() {
		It("returns false for empty allowlist", func() {
			pr := githubclient.PullRequest{AuthorLogin: "alice"}
			Expect(filter.IsBotAuthor(pr, nil)).To(BeFalse())
		})

		It("returns true for exact match", func() {
			pr := githubclient.PullRequest{AuthorLogin: "renovate[bot]"}
			Expect(
				filter.IsBotAuthor(pr, []string{"dependabot[bot]", "renovate[bot]"}),
			).To(BeTrue())
		})

		It("returns false when no entry matches", func() {
			pr := githubclient.PullRequest{AuthorLogin: "alice"}
			Expect(
				filter.IsBotAuthor(pr, []string{"dependabot[bot]", "renovate[bot]"}),
			).To(BeFalse())
		})
	})
})
