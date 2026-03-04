// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/pr-reviewer/pkg/github"
)

var _ = Describe("ParsePRURL", func() {
	var (
		rawURL string
		info   *github.PRInfo
		err    error
	)

	JustBeforeEach(func() {
		info, err = github.ParsePRURL(rawURL)
	})

	Context("with valid GitHub PR URL", func() {
		BeforeEach(func() {
			rawURL = "https://github.com/bborbe/teamvault-docker/pull/4"
		})

		It("returns no error", func() {
			Expect(err).To(BeNil())
		})

		It("extracts owner correctly", func() {
			Expect(info.Owner).To(Equal("bborbe"))
		})

		It("extracts repo correctly", func() {
			Expect(info.Repo).To(Equal("teamvault-docker"))
		})

		It("extracts PR number correctly", func() {
			Expect(info.Number).To(Equal(4))
		})

		It("constructs repo URL correctly", func() {
			Expect(info.RepoURL).To(Equal("https://github.com/bborbe/teamvault-docker"))
		})
	})

	Context("with valid GitHub PR URL with trailing slash", func() {
		BeforeEach(func() {
			rawURL = "https://github.com/bborbe/teamvault-docker/pull/4/"
		})

		It("returns no error", func() {
			Expect(err).To(BeNil())
		})

		It("extracts owner correctly", func() {
			Expect(info.Owner).To(Equal("bborbe"))
		})

		It("extracts repo correctly", func() {
			Expect(info.Repo).To(Equal("teamvault-docker"))
		})

		It("extracts PR number correctly", func() {
			Expect(info.Number).To(Equal(4))
		})

		It("constructs repo URL correctly", func() {
			Expect(info.RepoURL).To(Equal("https://github.com/bborbe/teamvault-docker"))
		})
	})

	Context("with non-GitHub URL", func() {
		BeforeEach(func() {
			rawURL = "https://gitlab.com/user/repo/merge_requests/1"
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})

		It("error contains the URL", func() {
			Expect(err.Error()).To(ContainSubstring(rawURL))
		})
	})

	Context("with GitHub URL but not a PR (issue)", func() {
		BeforeEach(func() {
			rawURL = "https://github.com/bborbe/pr-reviewer/issues/123"
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})
	})

	Context("with GitHub PR URL with non-numeric number", func() {
		BeforeEach(func() {
			rawURL = "https://github.com/bborbe/pr-reviewer/pull/abc"
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})
	})

	Context("with empty string", func() {
		BeforeEach(func() {
			rawURL = ""
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})
	})

	Context("with URL without scheme", func() {
		BeforeEach(func() {
			rawURL = "github.com/bborbe/pr-reviewer/pull/4"
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})
	})

	Context("with malformed GitHub URL (missing parts)", func() {
		BeforeEach(func() {
			rawURL = "https://github.com/bborbe/pull/4"
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})
	})

	Context("with malformed GitHub URL (too many parts)", func() {
		BeforeEach(func() {
			rawURL = "https://github.com/bborbe/repo/extra/pull/4"
		})

		It("returns error", func() {
			Expect(err).NotTo(BeNil())
		})

		It("error contains 'unsupported URL format'", func() {
			Expect(err.Error()).To(ContainSubstring("unsupported URL format"))
		})
	})
})
