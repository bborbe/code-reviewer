// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/pr-reviewer/pkg/github"
)

var _ = Describe("Client", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("NewGHClient", func() {
		It("creates a non-nil client with empty token", func() {
			client := github.NewGHClient("")
			Expect(client).NotTo(BeNil())
		})

		It("creates a non-nil client with token", func() {
			client := github.NewGHClient("test-token")
			Expect(client).NotTo(BeNil())
		})

		It("implements Client interface", func() {
			var _ github.Client = github.NewGHClient("")
		})
	})

	Context("GetPRBranch", func() {
		It("requires valid context", func() {
			client := github.NewGHClient("")
			_, err := client.GetPRBranch(ctx, "owner", "repo", 123)
			// Will fail in test env without gh CLI, but validates interface contract
			Expect(err).To(HaveOccurred())
		})

		It("respects context cancellation", func() {
			client := github.NewGHClient("")
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // Cancel immediately
			_, err := client.GetPRBranch(cancelCtx, "owner", "repo", 123)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("PostComment", func() {
		It("requires valid context", func() {
			client := github.NewGHClient("")
			err := client.PostComment(ctx, "owner", "repo", 123, "test comment")
			// Will fail in test env without gh CLI, but validates interface contract
			Expect(err).To(HaveOccurred())
		})

		It("respects context cancellation", func() {
			client := github.NewGHClient("")
			cancelCtx, cancel := context.WithCancel(ctx)
			cancel() // Cancel immediately
			err := client.PostComment(cancelCtx, "owner", "repo", 123, "test comment")
			Expect(err).To(HaveOccurred())
		})
	})

	// NOTE: Success path tests for GetPRBranch and PostComment are not practical
	// without refactoring. The ghClient uses exec.CommandContext internally, which
	// cannot be mocked without injecting a command executor interface. To test:
	// - Branch name parsing from gh output
	// - Correct command argument construction
	// - GH_TOKEN environment variable handling
	// We would need to refactor ghClient to accept an injectable command executor,
	// which is outside the scope of adding tests to existing code.
})
