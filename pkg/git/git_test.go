// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/pr-reviewer/pkg/git"
)

var _ = Describe("WorktreeManager", func() {
	var (
		ctx     context.Context
		manager git.WorktreeManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		manager = git.NewWorktreeManager()
	})

	Describe("Fetch", func() {
		Context("with non-existent path", func() {
			It("returns error", func() {
				err := manager.Fetch(ctx, "/nonexistent/path")
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("local path not found: /nonexistent/path"))
			})
		})

		Context("with non-git directory", func() {
			var tmpDir string

			BeforeEach(func() {
				tmpDir = GinkgoT().TempDir()
			})

			It("returns error", func() {
				err := manager.Fetch(ctx, tmpDir)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("not a git repo: " + tmpDir))
			})
		})

		Context("with valid git repository", func() {
			var repoPath string

			BeforeEach(func() {
				repoPath = initGitRepo()
			})

			It("succeeds", func() {
				err := manager.Fetch(ctx, repoPath)
				Expect(err).To(BeNil())
			})
		})
	})

	Describe("CreateWorktree", func() {
		var repoPath string

		BeforeEach(func() {
			repoPath = initGitRepo()
			createBranch(repoPath, "feature-branch")
		})

		Context("with valid branch", func() {
			It("returns deterministic worktree path", func() {
				worktreePath, err := manager.CreateWorktree(ctx, repoPath, "feature-branch", 123)
				Expect(err).To(BeNil())
				Expect(worktreePath).To(Equal(filepath.Join(repoPath, ".worktrees", "pr-123")))

				// Verify worktree directory exists
				info, err := os.Stat(worktreePath)
				Expect(err).To(BeNil())
				Expect(info.IsDir()).To(BeTrue())
			})
		})

		Context("with stale worktree from previous run", func() {
			BeforeEach(func() {
				// Create initial worktree
				_, err := manager.CreateWorktree(ctx, repoPath, "feature-branch", 456)
				Expect(err).To(BeNil())
			})

			It("removes stale worktree and creates fresh one", func() {
				worktreePath, err := manager.CreateWorktree(ctx, repoPath, "feature-branch", 456)
				Expect(err).To(BeNil())
				Expect(worktreePath).To(Equal(filepath.Join(repoPath, ".worktrees", "pr-456")))

				// Verify worktree exists
				_, err = os.Stat(worktreePath)
				Expect(err).To(BeNil())
			})
		})

		Context("with non-existent branch", func() {
			It("returns error", func() {
				_, err := manager.CreateWorktree(ctx, repoPath, "nonexistent-branch", 789)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("branch not found: nonexistent-branch"))
			})
		})

		Context("with non-existent repo path", func() {
			It("returns error", func() {
				_, err := manager.CreateWorktree(ctx, "/nonexistent/path", "feature-branch", 999)
				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("local path not found: /nonexistent/path"))
			})
		})
	})

	Describe("RemoveWorktree", func() {
		var repoPath string

		BeforeEach(func() {
			repoPath = initGitRepo()
			createBranch(repoPath, "test-branch")
		})

		Context("with non-existent worktree", func() {
			It("returns nil (idempotent)", func() {
				nonExistentPath := filepath.Join(repoPath, ".worktrees", "pr-nonexistent")
				err := manager.RemoveWorktree(ctx, repoPath, nonExistentPath)
				Expect(err).To(BeNil())
			})
		})

		Context("with existing worktree", func() {
			var worktreePath string

			BeforeEach(func() {
				var err error
				worktreePath, err = manager.CreateWorktree(ctx, repoPath, "test-branch", 111)
				Expect(err).To(BeNil())
			})

			It("removes worktree successfully", func() {
				// Verify worktree exists
				_, err := os.Stat(worktreePath)
				Expect(err).To(BeNil())

				// Remove worktree
				err = manager.RemoveWorktree(ctx, repoPath, worktreePath)
				Expect(err).To(BeNil())

				// Verify worktree is gone
				_, err = os.Stat(worktreePath)
				Expect(os.IsNotExist(err)).To(BeTrue())
			})
		})
	})

	Describe("Full integration cycle", func() {
		var repoPath string

		BeforeEach(func() {
			repoPath = initGitRepo()
			createBranch(repoPath, "pr-branch")
		})

		It("fetch → create → verify → remove → verify gone", func() {
			// Fetch (simulates updating branches)
			err := manager.Fetch(ctx, repoPath)
			Expect(err).To(BeNil())

			// Create worktree
			worktreePath, err := manager.CreateWorktree(ctx, repoPath, "pr-branch", 42)
			Expect(err).To(BeNil())
			Expect(worktreePath).To(Equal(filepath.Join(repoPath, ".worktrees", "pr-42")))

			// Verify worktree exists and has git repo
			gitFile := filepath.Join(worktreePath, ".git")
			_, err = os.Stat(gitFile)
			Expect(err).To(BeNil())

			// Remove worktree
			err = manager.RemoveWorktree(ctx, repoPath, worktreePath)
			Expect(err).To(BeNil())

			// Verify worktree is gone
			_, err = os.Stat(worktreePath)
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})
})

// Helper functions for test setup

func initGitRepo() string {
	tmpDir := GinkgoT().TempDir()

	// Initialize git repo
	runCmd(tmpDir, "git", "init")
	runCmd(tmpDir, "git", "config", "user.email", "test@example.com")
	runCmd(tmpDir, "git", "config", "user.name", "Test User")

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	err := os.WriteFile(testFile, []byte("# Test Repo\n"), 0600)
	Expect(err).To(BeNil())

	runCmd(tmpDir, "git", "add", "README.md")
	runCmd(tmpDir, "git", "commit", "-m", "Initial commit")

	return tmpDir
}

func createBranch(repoPath, branchName string) {
	runCmd(repoPath, "git", "branch", branchName)
}

func runCmd(dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	err := cmd.Run()
	Expect(err).To(BeNil())
}
