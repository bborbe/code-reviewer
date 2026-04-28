// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	agentlib "github.com/bborbe/agent/lib"
	libtime "github.com/bborbe/time"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/code-reviewer/watcher/github/pkg"
	"github.com/bborbe/code-reviewer/watcher/github/pkg/mocks"
)

func newTestWatcher(
	ghClient pkg.GitHubClient,
	pub *mocks.CommandPublisher,
	cursorPath string,
	startTime libtime.DateTime,
) pkg.Watcher {
	return pkg.NewWatcher(
		ghClient,
		pub,
		cursorPath,
		startTime,
		"bborbe",
		[]string{"dependabot[bot]"},
		"dev",
	)
}

var _ = Describe("pkg.Watcher", func() {
	var (
		ctx        context.Context
		cancel     context.CancelFunc
		ghClient   *mocks.GitHubClient
		pub        *mocks.CommandPublisher
		tmpDir     string
		cursorPath string
		startTime  libtime.DateTime
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		ghClient = new(mocks.GitHubClient)
		pub = new(mocks.CommandPublisher)
		startTime = libtime.DateTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		var err error
		tmpDir, err = os.MkdirTemp("", "watcher-test-*")
		Expect(err).NotTo(HaveOccurred())
		cursorPath = filepath.Join(tmpDir, "cursor.json")
	})

	AfterEach(func() {
		cancel()
		_ = os.RemoveAll(tmpDir) // #nosec G104 -- best-effort temp dir cleanup
	})

	Describe("No PRs returned", func() {
		It("returns nil, cursor saved", func() {
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  nil,
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			err := w.Poll(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(0))
			Expect(pub.PublishUpdateFrontmatterCallCount()).To(Equal(0))
			_, err = os.Stat(cursorPath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("New PR (no existing cursor entry)", func() {
		It("publishes CreateTaskCommand", func() {
			pr := pkg.PullRequest{
				Number:      42,
				Owner:       "bborbe",
				Repo:        "code-reviewer",
				Title:       "feat: new feature",
				HTMLURL:     "https://github.com/bborbe/code-reviewer/pull/42",
				AuthorLogin: "alice",
				IsDraft:     false,
				UpdatedAt:   libtime.DateTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("abc123", nil)
			pub.PublishCreateReturns(nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			err := w.Poll(ctx)
			Expect(err).NotTo(HaveOccurred())

			Expect(pub.PublishCreateCallCount()).To(Equal(1))
			Expect(pub.PublishUpdateFrontmatterCallCount()).To(Equal(0))

			_, cmd := pub.PublishCreateArgsForCall(0)
			Expect(string(cmd.TaskIdentifier)).NotTo(BeEmpty())
			Expect(cmd.Frontmatter["assignee"]).To(Equal("pr-reviewer-agent"))
		})
	})

	Describe("Existing PR, same SHA", func() {
		It("publishes nothing", func() {
			pr := pkg.PullRequest{
				Number:      42,
				Owner:       "bborbe",
				Repo:        "code-reviewer",
				Title:       "existing pr",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("sha-existing", nil)

			// Pre-populate cursor with the same SHA
			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			// First poll: creates the entry
			pub.PublishCreateReturns(nil)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(1))

			// Second poll: same SHA, no publish
			pub = new(mocks.CommandPublisher)
			w = newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(0))
			Expect(pub.PublishUpdateFrontmatterCallCount()).To(Equal(0))
		})
	})

	Describe("Force-push (existing entry, different SHA)", func() {
		It("publishes UpdateFrontmatterCommand with correct body section", func() {
			pr := pkg.PullRequest{
				Number:      42,
				Owner:       "bborbe",
				Repo:        "code-reviewer",
				Title:       "force pushed pr",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
			}

			// First poll: register initial SHA
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("old-sha", nil)
			pub.PublishCreateReturns(nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(1))

			// Second poll: new SHA (force-push)
			pub = new(mocks.CommandPublisher)
			ghClient.GetHeadSHAReturns("new-sha", nil)
			pub.PublishUpdateFrontmatterReturns(nil)

			w = newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())

			Expect(pub.PublishUpdateFrontmatterCallCount()).To(Equal(1))
			_, cmd := pub.PublishUpdateFrontmatterArgsForCall(0)
			Expect(cmd.Body).NotTo(BeNil())
			Expect(cmd.Body.Heading).To(ContainSubstring("## Outdated by force-push old-sha"))
			Expect(cmd.Updates["phase"]).To(Equal("planning"))
		})
	})

	Describe("Draft PR", func() {
		It("is skipped, no publish calls", func() {
			pr := pkg.PullRequest{
				Number:      1,
				Owner:       "bborbe",
				Repo:        "repo",
				AuthorLogin: "alice",
				IsDraft:     true,
				UpdatedAt:   libtime.DateTime(fixedNow),
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(0))
			Expect(pub.PublishUpdateFrontmatterCallCount()).To(Equal(0))
		})
	})

	Describe("Bot-authored PR", func() {
		It("is skipped, no publish calls", func() {
			pr := pkg.PullRequest{
				Number:      2,
				Owner:       "bborbe",
				Repo:        "repo",
				AuthorLogin: "dependabot[bot]",
				IsDraft:     false,
				UpdatedAt:   libtime.DateTime(fixedNow),
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(0))
		})
	})

	Describe("GitHub error", func() {
		It("Poll returns nil, cursor unchanged, no publish calls", func() {
			ghClient.SearchPRsReturns(pkg.SearchResult{}, errors.New("network timeout"))

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			err := w.Poll(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(0))
			// pkg.Cursor file should not exist (no save after error)
			_, statErr := os.Stat(cursorPath)
			Expect(os.IsNotExist(statErr)).To(BeTrue())
		})
	})

	Describe("Kafka publish fails (CreateTaskCommand)", func() {
		It("Poll returns nil, cursor not updated for that PR", func() {
			pr := pkg.PullRequest{
				Number:      99,
				Owner:       "bborbe",
				Repo:        "repo",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)),
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("sha123", nil)
			pub.PublishCreateReturns(errors.New("kafka unavailable"))

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			err := w.Poll(ctx)
			Expect(err).NotTo(HaveOccurred())

			// HeadSHAs should not include this PR since publish failed
			// Verify by doing a second poll and checking PublishCreate is called again
			pub2 := new(mocks.CommandPublisher)
			pub2.PublishCreateReturns(nil)
			w2 := newTestWatcher(ghClient, pub2, cursorPath, startTime)
			Expect(w2.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub2.PublishCreateCallCount()).To(Equal(1))
		})
	})

	Describe("Context cancellation between pages", func() {
		It("Poll returns nil cleanly", func() {
			cancelCtx, cancelFn := context.WithCancel(context.Background())

			callCount := 0
			ghClient.SearchPRsStub = func(c context.Context, scope string, since libtime.DateTime, page int) (pkg.SearchResult, error) {
				callCount++
				if callCount == 1 {
					cancelFn() // cancel context after first page
					return pkg.SearchResult{
						PullRequests:  nil,
						HasNextPage:   true,
						NextPage:      2,
						RateRemaining: 100,
					}, nil
				}
				return pkg.SearchResult{}, nil
			}

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			err := w.Poll(cancelCtx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("pkg.Cursor file missing on first poll", func() {
		It("uses startTime as since, calls SearchPRs correctly", func() {
			ghClient.SearchPRsReturns(pkg.SearchResult{
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())

			Expect(ghClient.SearchPRsCallCount()).To(Equal(1))
			_, _, since, _ := ghClient.SearchPRsArgsForCall(0)
			Expect(since).To(Equal(startTime))
		})
	})

	Describe("LoadCursor error (unreadable cursor file)", func() {
		It("Poll returns a non-nil error", func() {
			if os.Getuid() == 0 {
				Skip("running as root, skipping permission test")
			}
			Expect(os.WriteFile(cursorPath, []byte("{}"), 0600)).To(Succeed())
			Expect(os.Chmod(cursorPath, 0000)).To(Succeed())
			defer func() { _ = os.Chmod(cursorPath, 0600) }()

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			err := w.Poll(ctx)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Closed PR pruned from cursor after poll", func() {
		It("removes closed PR's task ID from HeadSHAs after second poll", func() {
			prA := pkg.PullRequest{
				Number:      42,
				Owner:       "bborbe",
				Repo:        "code-reviewer",
				Title:       "pr A",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
			}
			prB := pkg.PullRequest{
				Number:      43,
				Owner:       "bborbe",
				Repo:        "code-reviewer",
				Title:       "pr B",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
			}

			// First poll: both PRs open
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{prA, prB},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("sha-initial", nil)
			pub.PublishCreateReturns(nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(pub.PublishCreateCallCount()).To(Equal(2))

			// Second poll: only PR A returned (PR B closed/merged)
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{prA},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			pub2 := new(mocks.CommandPublisher)
			pub2.PublishCreateReturns(nil)
			w2 := newTestWatcher(ghClient, pub2, cursorPath, startTime)
			Expect(w2.Poll(ctx)).NotTo(HaveOccurred())

			cursor, err := pkg.LoadCursor(ctx, cursorPath, startTime)
			Expect(err).NotTo(HaveOccurred())

			taskIDA := pkg.DeriveTaskID(prA.Owner, prA.Repo, prA.Number).String()
			taskIDB := pkg.DeriveTaskID(prB.Owner, prB.Repo, prB.Number).String()
			Expect(cursor.HeadSHAs).To(HaveKey(taskIDA))
			Expect(cursor.HeadSHAs).NotTo(HaveKey(taskIDB))
		})
	})

	Describe("pkg.Cursor save fails", func() {
		It("Poll returns nil (non-crash)", func() {
			ghClient.SearchPRsReturns(pkg.SearchResult{
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)

			// Use an unwritable path
			w := newTestWatcher(ghClient, pub, "/nonexistent/path/cursor.json", startTime)
			err := w.Poll(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("GetHeadSHA caches result", func() {
		It("calls GetHeadSHA once per unique PR", func() {
			prs := []pkg.PullRequest{
				{
					Number:      1,
					Owner:       "bborbe",
					Repo:        "repo",
					AuthorLogin: "alice",
					UpdatedAt:   libtime.DateTime(fixedNow),
				},
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  prs,
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("sha1", nil)
			pub.PublishCreateReturns(nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())
			Expect(ghClient.GetHeadSHACallCount()).To(Equal(1))
		})
	})

	Describe("buildFrontmatter fields", func() {
		It("includes required keys", func() {
			pr := pkg.PullRequest{
				Number:      5,
				Owner:       "bborbe",
				Repo:        "repo",
				Title:       "my title",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(fixedNow),
			}
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("sha1", nil)
			pub.PublishCreateReturns(nil)

			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())

			Expect(pub.PublishCreateCallCount()).To(Equal(1))
			_, cmd := pub.PublishCreateArgsForCall(0)
			Expect(cmd.Frontmatter["assignee"]).To(Equal("pr-reviewer-agent"))
			Expect(cmd.Frontmatter["phase"]).To(Equal("planning"))
			Expect(cmd.Frontmatter["status"]).To(Equal("in_progress"))
			Expect(cmd.Frontmatter["stage"]).To(Equal("dev"))
			Expect(cmd.Frontmatter["title"]).To(Equal("my title"))
			Expect(cmd.Frontmatter["task_identifier"]).NotTo(BeEmpty())
		})
	})

	Describe("UpdateFrontmatterCommand fields", func() {
		It("body section heading matches exact format", func() {
			pr := pkg.PullRequest{
				Number:      7,
				Owner:       "bborbe",
				Repo:        "repo",
				AuthorLogin: "alice",
				UpdatedAt:   libtime.DateTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)),
			}

			// First poll: create
			ghClient.SearchPRsReturns(pkg.SearchResult{
				PullRequests:  []pkg.PullRequest{pr},
				HasNextPage:   false,
				RateRemaining: 100,
			}, nil)
			ghClient.GetHeadSHAReturns("sha-v1", nil)
			pub.PublishCreateReturns(nil)
			w := newTestWatcher(ghClient, pub, cursorPath, startTime)
			Expect(w.Poll(ctx)).NotTo(HaveOccurred())

			// Second poll: force-push
			pub2 := new(mocks.CommandPublisher)
			ghClient.GetHeadSHAReturns("sha-v2", nil)
			pub2.PublishUpdateFrontmatterReturns(nil)
			w2 := newTestWatcher(ghClient, pub2, cursorPath, startTime)
			Expect(w2.Poll(ctx)).NotTo(HaveOccurred())

			Expect(pub2.PublishUpdateFrontmatterCallCount()).To(Equal(1))
			_, cmd := pub2.PublishUpdateFrontmatterArgsForCall(0)
			Expect(cmd.Body.Heading).To(Equal("## Outdated by force-push sha-v1"))
			Expect(cmd.Body.Section).To(Equal("## Outdated by force-push sha-v1\n"))
			Expect(cmd.Updates["phase"]).To(Equal("planning"))
			Expect(cmd.Updates["status"]).To(Equal("in_progress"))
			Expect(cmd.Updates["trigger_count"]).To(Equal(0))
			taskID := agentlib.TaskIdentifier("")
			_, cmd2 := pub2.PublishUpdateFrontmatterArgsForCall(0)
			taskID = cmd2.TaskIdentifier
			Expect(string(taskID)).NotTo(BeEmpty())
		})
	})
})
