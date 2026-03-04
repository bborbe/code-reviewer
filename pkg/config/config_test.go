// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package config_test

import (
	"context"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/pr-reviewer/pkg/config"
)

var _ = Describe("Config", func() {
	var (
		ctx    context.Context
		loader config.Loader
		tmpDir string
		cfg    *config.Config
		err    error
	)

	BeforeEach(func() {
		ctx = context.Background()
		tmpDir = GinkgoT().TempDir()
	})

	Describe("Load", func() {
		var configPath string

		JustBeforeEach(func() {
			loader = config.NewFileLoader(configPath)
			cfg, err = loader.Load(ctx)
		})

		Context("with valid YAML", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "config.yaml")
				validYAML := `repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /home/user/teamvault-docker
  - url: https://github.com/bborbe/pr-reviewer
    path: /home/user/pr-reviewer
`
				err := os.WriteFile(configPath, []byte(validYAML), 0600)
				Expect(err).To(BeNil())
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("loads repos correctly", func() {
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Repos).To(HaveLen(2))
				Expect(cfg.Repos[0].URL).To(Equal("https://github.com/bborbe/teamvault-docker"))
				Expect(cfg.Repos[0].Path).To(Equal("/home/user/teamvault-docker"))
				Expect(cfg.Repos[1].URL).To(Equal("https://github.com/bborbe/pr-reviewer"))
				Expect(cfg.Repos[1].Path).To(Equal("/home/user/pr-reviewer"))
			})
		})

		Context("with reviewCommand in YAML", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "config.yaml")
				yamlWithReviewCmd := `repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /home/user/teamvault-docker
    reviewCommand: /custom-review
  - url: https://github.com/bborbe/pr-reviewer
    path: /home/user/pr-reviewer
`
				err := os.WriteFile(configPath, []byte(yamlWithReviewCmd), 0600)
				Expect(err).To(BeNil())
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("loads reviewCommand correctly", func() {
				Expect(cfg).NotTo(BeNil())
				Expect(cfg.Repos).To(HaveLen(2))
				Expect(cfg.Repos[0].ReviewCommand).To(Equal("/custom-review"))
				Expect(cfg.Repos[1].ReviewCommand).To(Equal(""))
			})
		})

		Context("without github section", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "config.yaml")
				yamlWithoutGithub := `repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /home/user/teamvault-docker
`
				err := os.WriteFile(configPath, []byte(yamlWithoutGithub), 0600)
				Expect(err).To(BeNil())
				// Ensure default env var is unset so token resolves to empty
				err = os.Unsetenv("PR_REVIEWER_GITHUB_TOKEN")
				Expect(err).To(BeNil())
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("ResolvedGitHubToken falls back to default env var and returns empty when unset", func() {
				Expect(cfg.ResolvedGitHubToken()).To(Equal(""))
			})
		})

		Context("without github section but default env var set", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "config.yaml")
				yamlWithoutGithub := `repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /home/user/teamvault-docker
`
				err := os.WriteFile(configPath, []byte(yamlWithoutGithub), 0600)
				Expect(err).To(BeNil())
				err = os.Setenv("PR_REVIEWER_GITHUB_TOKEN", "default-token-value")
				Expect(err).To(BeNil())
			})

			AfterEach(func() {
				err := os.Unsetenv("PR_REVIEWER_GITHUB_TOKEN")
				Expect(err).To(BeNil())
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("ResolvedGitHubToken returns the default env var value", func() {
				Expect(cfg.ResolvedGitHubToken()).To(Equal("default-token-value"))
			})
		})

		Context("with github.token as env var reference", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "config.yaml")
				yamlWithEnvVar := `github:
  token: ${TEST_PR_REVIEWER_TOKEN}
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /home/user/teamvault-docker
`
				err := os.WriteFile(configPath, []byte(yamlWithEnvVar), 0600)
				Expect(err).To(BeNil())
			})

			Context("when env var is set", func() {
				BeforeEach(func() {
					err := os.Setenv("TEST_PR_REVIEWER_TOKEN", "test-token-value")
					Expect(err).To(BeNil())
				})

				AfterEach(func() {
					err := os.Unsetenv("TEST_PR_REVIEWER_TOKEN")
					Expect(err).To(BeNil())
				})

				It("returns no error", func() {
					Expect(err).To(BeNil())
				})

				It("ResolvedGitHubToken returns the env var value", func() {
					Expect(cfg.ResolvedGitHubToken()).To(Equal("test-token-value"))
				})
			})

			Context("when env var is not set", func() {
				BeforeEach(func() {
					err := os.Unsetenv("TEST_PR_REVIEWER_TOKEN")
					Expect(err).To(BeNil())
				})

				It("returns no error", func() {
					Expect(err).To(BeNil())
				})

				It("ResolvedGitHubToken returns empty string", func() {
					Expect(cfg.ResolvedGitHubToken()).To(Equal(""))
				})
			})
		})

		Context("with github.token as literal value", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "config.yaml")
				yamlWithLiteral := `github:
  token: literal-token-value
repos:
  - url: https://github.com/bborbe/teamvault-docker
    path: /home/user/teamvault-docker
`
				err := os.WriteFile(configPath, []byte(yamlWithLiteral), 0600)
				Expect(err).To(BeNil())
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("ResolvedGitHubToken returns the literal value", func() {
				Expect(cfg.ResolvedGitHubToken()).To(Equal("literal-token-value"))
			})
		})

		Context("with missing file", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "nonexistent.yaml")
			})

			It("returns error", func() {
				Expect(err).NotTo(BeNil())
			})

			It("error contains 'config not found'", func() {
				Expect(err.Error()).To(ContainSubstring("config not found"))
			})

			It("error contains the path", func() {
				Expect(err.Error()).To(ContainSubstring(configPath))
			})
		})

		Context("with invalid YAML", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "invalid.yaml")
				invalidYAML := `repos:
  - url: https://github.com/example/repo
    path: /home/user/repo
    invalid_indent
`
				err := os.WriteFile(configPath, []byte(invalidYAML), 0600)
				Expect(err).To(BeNil())
			})

			It("returns error", func() {
				Expect(err).NotTo(BeNil())
			})

			It("error contains parse failure message", func() {
				Expect(err.Error()).To(ContainSubstring("parse config failed"))
			})
		})

		Context("with empty URL", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "empty-url.yaml")
				emptyURLYAML := `repos:
  - url: ""
    path: /home/user/repo
`
				err := os.WriteFile(configPath, []byte(emptyURLYAML), 0600)
				Expect(err).To(BeNil())
			})

			It("returns error", func() {
				Expect(err).NotTo(BeNil())
			})

			It("error contains validation message", func() {
				Expect(
					err.Error(),
				).To(ContainSubstring("invalid repo entry: url and path required"))
			})
		})

		Context("with empty path", func() {
			BeforeEach(func() {
				configPath = filepath.Join(tmpDir, "empty-path.yaml")
				emptyPathYAML := `repos:
  - url: https://github.com/example/repo
    path: ""
`
				err := os.WriteFile(configPath, []byte(emptyPathYAML), 0600)
				Expect(err).To(BeNil())
			})

			It("returns error", func() {
				Expect(err).NotTo(BeNil())
			})

			It("error contains validation message", func() {
				Expect(
					err.Error(),
				).To(ContainSubstring("invalid repo entry: url and path required"))
			})
		})
	})

	Describe("FindRepoPath", func() {
		var (
			repoURL    string
			resultPath string
		)

		BeforeEach(func() {
			cfg = &config.Config{
				Repos: []config.RepoConfig{
					{
						URL:  "https://github.com/bborbe/teamvault-docker",
						Path: "/home/user/teamvault-docker",
					},
					{
						URL:  "https://github.com/bborbe/pr-reviewer",
						Path: "/home/user/pr-reviewer",
					},
				},
			}
		})

		JustBeforeEach(func() {
			resultPath, err = cfg.FindRepoPath(repoURL)
		})

		Context("with matching URL", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/bborbe/teamvault-docker"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(resultPath).To(Equal("/home/user/teamvault-docker"))
			})
		})

		Context("with non-matching URL", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/unknown/repo"
			})

			It("returns error", func() {
				Expect(err).NotTo(BeNil())
			})

			It("error contains 'repo not found'", func() {
				Expect(err.Error()).To(ContainSubstring("repo not found in config"))
			})

			It("error contains the URL", func() {
				Expect(err.Error()).To(ContainSubstring(repoURL))
			})
		})

		Context("with case-insensitive matching", func() {
			BeforeEach(func() {
				repoURL = "HTTPS://GITHUB.COM/BBORBE/TEAMVAULT-DOCKER"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(resultPath).To(Equal("/home/user/teamvault-docker"))
			})
		})

		Context("with trailing slash normalization (URL has slash)", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/bborbe/teamvault-docker/"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(resultPath).To(Equal("/home/user/teamvault-docker"))
			})
		})

		Context("with trailing slash normalization (config has slash)", func() {
			BeforeEach(func() {
				cfg.Repos[0].URL = "https://github.com/bborbe/teamvault-docker/"
				repoURL = "https://github.com/bborbe/teamvault-docker"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(resultPath).To(Equal("/home/user/teamvault-docker"))
			})
		})
	})

	Describe("FindRepo", func() {
		var (
			repoURL  string
			repoInfo *config.RepoInfo
		)

		BeforeEach(func() {
			cfg = &config.Config{
				Repos: []config.RepoConfig{
					{
						URL:           "https://github.com/bborbe/teamvault-docker",
						Path:          "/home/user/teamvault-docker",
						ReviewCommand: "/custom-review",
					},
					{
						URL:  "https://github.com/bborbe/pr-reviewer",
						Path: "/home/user/pr-reviewer",
						// ReviewCommand not set - should default to "/code-review"
					},
				},
			}
		})

		JustBeforeEach(func() {
			repoInfo, err = cfg.FindRepo(repoURL)
		})

		Context("with matching URL and custom reviewCommand", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/bborbe/teamvault-docker"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(repoInfo.Path).To(Equal("/home/user/teamvault-docker"))
			})

			It("returns the custom reviewCommand", func() {
				Expect(repoInfo.ReviewCommand).To(Equal("/custom-review"))
			})
		})

		Context("with matching URL and empty reviewCommand", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/bborbe/pr-reviewer"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(repoInfo.Path).To(Equal("/home/user/pr-reviewer"))
			})

			It("defaults reviewCommand to /code-review", func() {
				Expect(repoInfo.ReviewCommand).To(Equal("/code-review"))
			})
		})

		Context("with non-matching URL", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/unknown/repo"
			})

			It("returns error", func() {
				Expect(err).NotTo(BeNil())
			})

			It("error contains 'repo not found'", func() {
				Expect(err.Error()).To(ContainSubstring("repo not found in config"))
			})

			It("error contains the URL", func() {
				Expect(err.Error()).To(ContainSubstring(repoURL))
			})
		})

		Context("with case-insensitive matching", func() {
			BeforeEach(func() {
				repoURL = "HTTPS://GITHUB.COM/BBORBE/TEAMVAULT-DOCKER"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(repoInfo.Path).To(Equal("/home/user/teamvault-docker"))
			})

			It("returns the custom reviewCommand", func() {
				Expect(repoInfo.ReviewCommand).To(Equal("/custom-review"))
			})
		})

		Context("with trailing slash normalization", func() {
			BeforeEach(func() {
				repoURL = "https://github.com/bborbe/teamvault-docker/"
			})

			It("returns no error", func() {
				Expect(err).To(BeNil())
			})

			It("returns the correct path", func() {
				Expect(repoInfo.Path).To(Equal("/home/user/teamvault-docker"))
			})
		})
	})

	Describe("ResolvedModel", func() {
		Context("when model field is not set", func() {
			BeforeEach(func() {
				cfg = &config.Config{
					Repos: []config.RepoConfig{
						{
							URL:  "https://github.com/bborbe/teamvault-docker",
							Path: "/home/user/teamvault-docker",
						},
					},
				}
			})

			It("returns default model (sonnet)", func() {
				Expect(cfg.ResolvedModel()).To(Equal("sonnet"))
			})
		})

		Context("when model field is set to custom value", func() {
			BeforeEach(func() {
				cfg = &config.Config{
					Model: "opus",
					Repos: []config.RepoConfig{
						{
							URL:  "https://github.com/bborbe/teamvault-docker",
							Path: "/home/user/teamvault-docker",
						},
					},
				}
			})

			It("returns configured model", func() {
				Expect(cfg.ResolvedModel()).To(Equal("opus"))
			})
		})

		Context("when model field is empty string", func() {
			BeforeEach(func() {
				cfg = &config.Config{
					Model: "",
					Repos: []config.RepoConfig{
						{
							URL:  "https://github.com/bborbe/teamvault-docker",
							Path: "/home/user/teamvault-docker",
						},
					},
				}
			})

			It("returns default model (sonnet)", func() {
				Expect(cfg.ResolvedModel()).To(Equal("sonnet"))
			})
		})
	})
})
