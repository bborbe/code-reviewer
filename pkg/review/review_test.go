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

				_, err := reviewer.Review(ctx, tempDir, "/code-review", "sonnet")
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(Equal("claude not found in PATH"))
			})
		})

		Context("with non-existent worktree path", func() {
			It("returns error", func() {
				// If claude exists in PATH, it will fail with a different error
				// If claude doesn't exist, it will fail with "claude not found in PATH"
				_, err := reviewer.Review(
					ctx,
					"/nonexistent/path/to/worktree",
					"/code-review",
					"sonnet",
				)
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

					result, err := reviewer.Review(ctx, worktreeDir, "/code-review", "sonnet")
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

					_, err := reviewer.Review(ctx, worktreeDir, "/code-review", "sonnet")
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

					result, err := reviewer.Review(ctx, worktreeDir, "/code-review", "sonnet")
					Expect(err).To(BeNil())
					Expect(result).To(Equal(""))
				})
			})

			Context("with custom review command", func() {
				BeforeEach(func() {
					// Create mock claude script that verifies the command parameter
					scriptContent := `#!/bin/sh
if [ "$4" = "/custom-review" ]; then
  echo "Custom review output"
  exit 0
else
  echo "Expected /custom-review but got $4" >&2
  exit 1
fi
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

				It("passes custom command correctly", func() {
					worktreeDir := GinkgoT().TempDir()

					result, err := reviewer.Review(ctx, worktreeDir, "/custom-review", "sonnet")
					Expect(err).To(BeNil())
					Expect(result).To(Equal("Custom review output\n"))
				})
			})

			Context("with model parameter", func() {
				BeforeEach(func() {
					// Create mock claude script that verifies the --model flag
					scriptContent := `#!/bin/sh
if [ "$1" = "--print" ] && [ "$2" = "--model" ] && [ "$3" = "opus" ] && [ "$4" = "/code-review" ]; then
  echo "Review with opus model"
  exit 0
else
  echo "Expected: --print --model opus /code-review, got: $@" >&2
  exit 1
fi
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

				It("passes --model flag correctly", func() {
					worktreeDir := GinkgoT().TempDir()

					result, err := reviewer.Review(ctx, worktreeDir, "/code-review", "opus")
					Expect(err).To(BeNil())
					Expect(result).To(Equal("Review with opus model\n"))
				})
			})
		})
	})
})

var _ = Describe("DockerReviewer", func() {
	var (
		ctx      context.Context
		reviewer review.Reviewer
	)

	BeforeEach(func() {
		ctx = context.Background()
		reviewer = review.NewDockerReviewer("test-image:latest")
	})

	Describe("Review", func() {
		Context("integration test with mock docker script", func() {
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

				mockScript = filepath.Join(binDir, "docker")

				// Save original PATH
				originalPATH = os.Getenv("PATH")
			})

			AfterEach(func() {
				// Restore original PATH
				err := os.Setenv("PATH", originalPATH)
				Expect(err).To(BeNil())
			})

			Context("with successful docker execution", func() {
				BeforeEach(func() {
					// Create mock docker script that prints review text
					scriptContent := `#!/bin/sh
echo "Docker review output"
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

					result, err := reviewer.Review(ctx, worktreeDir, "/code-review", "sonnet")
					Expect(err).To(BeNil())
					Expect(result).To(Equal("Docker review output\n"))
				})
			})

			Context("with docker returning non-zero exit code", func() {
				BeforeEach(func() {
					// Create mock docker script that fails
					scriptContent := `#!/bin/sh
echo "Error: container failed" >&2
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

					_, err := reviewer.Review(ctx, worktreeDir, "/code-review", "sonnet")
					Expect(err).NotTo(BeNil())
					Expect(err.Error()).To(ContainSubstring("claude review failed"))
					Expect(err.Error()).To(ContainSubstring("Error: container failed"))
				})
			})

			Context("with custom model parameter", func() {
				BeforeEach(func() {
					// Create mock docker script that verifies the model is passed correctly
					scriptContent := `#!/bin/sh
# Check that model parameter is present in the arguments
for arg in "$@"; do
  if [ "$arg" = "opus" ]; then
    echo "Model parameter found"
    exit 0
  fi
done
echo "Model parameter not found: $@" >&2
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

				It("passes model parameter correctly", func() {
					worktreeDir := GinkgoT().TempDir()

					result, err := reviewer.Review(ctx, worktreeDir, "/code-review", "opus")
					Expect(err).To(BeNil())
					Expect(result).To(Equal("Model parameter found\n"))
				})
			})

			Context("with volume mounts", func() {
				BeforeEach(func() {
					// Create mock docker script that verifies volume mounts
					scriptContent := `#!/bin/sh
# Check for -v flags (volume mounts)
for arg in "$@"; do
  case "$arg" in
    -v)
      echo "Volume mount flag found"
      exit 0
      ;;
  esac
done
echo "No volume mounts found" >&2
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

				It("includes volume mounts", func() {
					worktreeDir := GinkgoT().TempDir()

					result, err := reviewer.Review(ctx, worktreeDir, "/code-review", "sonnet")
					Expect(err).To(BeNil())
					Expect(result).To(Equal("Volume mount flag found\n"))
				})
			})
		})
	})
})
