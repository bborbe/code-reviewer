// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package review_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/pr-reviewer/pkg/review"
)

var _ = Describe("ClaudeReviewer", func() {
	var (
		ctx      context.Context
		reviewer review.Reviewer
	)

	BeforeEach(func() {
		ctx = context.Background()
		reviewer = review.NewClaudeReviewer()
	})

	Describe("Review", func() {
		Context("with claude not in PATH", func() {
			var originalPATH string

			BeforeEach(func() {
				// Save and clear PATH to ensure claude is not found
				originalPATH = os.Getenv("PATH")
				err := os.Setenv("PATH", "")
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				// Restore original PATH
				err := os.Setenv("PATH", originalPATH)
				Expect(err).To(BeNil())
			})

			It("returns error", func() {
				// Create empty temp directory
				tempDir := GinkgoT().TempDir()

				_, err := reviewer.Review(ctx, tempDir)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("claude not found in PATH"))
			})
		})

		Context("with non-existent worktree path", func() {
			It("returns error", func() {
				// If claude exists in PATH, it will fail with a different error
				// If claude doesn't exist, it will fail with "claude not found in PATH"
				_, err := reviewer.Review(ctx, "/nonexistent/path/to/worktree")
				Expect(err).NotTo(BeNil())
			})
		})

		Context("integration test with mock claude script", func() {
			var (
				tempDir      string
				mockScript   string
				originalPATH string
			)

			BeforeEach(func() {
				tempDir = GinkgoT().TempDir()
				binDir := filepath.Join(tempDir, "bin")
				err := os.Mkdir(binDir, 0750)
				Expect(err).To(BeNil())

				mockScript = filepath.Join(binDir, "claude")

				// Save original PATH
				originalPATH = os.Getenv("PATH")
			})

			AfterEach(func() {
				// Restore original PATH
				err := os.Setenv("PATH", originalPATH)
				Expect(err).To(BeNil())
			})

			Context("with successful claude execution", func() {
				BeforeEach(func() {
					// Create mock claude script that prints review text
					scriptContent := `#!/bin/sh
echo "Code review output"
exit 0
`
					// #nosec G306 -- test file: mock executable script needs execute permissions
					err := os.WriteFile(mockScript, []byte(scriptContent), 0750)
					Expect(err).To(BeNil())

					// Prepend bin dir to PATH
					err = os.Setenv(
						"PATH",
						filepath.Join(tempDir, "bin")+string(os.PathListSeparator)+originalPATH,
					)
					Expect(err).To(BeNil())
				})

				It("returns review text from stdout", func() {
					worktreeDir := GinkgoT().TempDir()

					result, err := reviewer.Review(ctx, worktreeDir)
					Expect(err).To(BeNil())
					Expect(result).To(Equal("Code review output\n"))
				})
			})

			Context("with claude returning non-zero exit code", func() {
				BeforeEach(func() {
					// Create mock claude script that fails
					scriptContent := `#!/bin/sh
echo "Error: something went wrong" >&2
exit 1
`
					// #nosec G306 -- test file: mock executable script needs execute permissions
					err := os.WriteFile(mockScript, []byte(scriptContent), 0750)
					Expect(err).To(BeNil())

					// Prepend bin dir to PATH
					err = os.Setenv(
						"PATH",
						filepath.Join(tempDir, "bin")+string(os.PathListSeparator)+originalPATH,
					)
					Expect(err).To(BeNil())
				})

				It("returns error with stderr content", func() {
					worktreeDir := GinkgoT().TempDir()

					_, err := reviewer.Review(ctx, worktreeDir)
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("claude review failed"))
					Expect(err.Error()).To(ContainSubstring("Error: something went wrong"))
				})
			})

			Context("with empty claude output", func() {
				BeforeEach(func() {
					// Create mock claude script with no output
					scriptContent := `#!/bin/sh
exit 0
`
					// #nosec G306 -- test file: mock executable script needs execute permissions
					err := os.WriteFile(mockScript, []byte(scriptContent), 0750)
					Expect(err).To(BeNil())

					// Prepend bin dir to PATH
					err = os.Setenv(
						"PATH",
						filepath.Join(tempDir, "bin")+string(os.PathListSeparator)+originalPATH,
					)
					Expect(err).To(BeNil())
				})

				It("returns empty string", func() {
					worktreeDir := GinkgoT().TempDir()

					result, err := reviewer.Review(ctx, worktreeDir)
					Expect(err).To(BeNil())
					Expect(result).To(Equal(""))
				})
			})
		})
	})
})
