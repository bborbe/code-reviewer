// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import (
	"context"
	"fmt"
	"time"

	agentlib "github.com/bborbe/agent/lib"
	"github.com/bborbe/errors"
	"github.com/golang/glog"
)

//counterfeiter:generate -o mocks/watcher.go --fake-name Watcher . Watcher

// Watcher polls GitHub and publishes task commands to Kafka.
type Watcher interface {
	Poll(ctx context.Context) error
}

// NewWatcher returns a Watcher that polls GitHub and publishes commands.
func NewWatcher(
	ghClient GitHubClient,
	pub CommandPublisher,
	cursorPath string,
	startTime time.Time,
	scope string,
	botAllowlist []string,
	stage string,
) Watcher {
	return &watcher{
		ghClient:     ghClient,
		publisher:    pub,
		cursorPath:   cursorPath,
		startTime:    startTime,
		scope:        scope,
		botAllowlist: botAllowlist,
		stage:        stage,
	}
}

type watcher struct {
	ghClient     GitHubClient
	publisher    CommandPublisher
	cursorPath   string
	startTime    time.Time
	scope        string
	botAllowlist []string
	stage        string
}

func (w *watcher) Poll(ctx context.Context) error {
	cursorState, err := LoadCursor(ctx, w.cursorPath, w.startTime)
	if err != nil {
		glog.Errorf("failed to load cursor err=%v", err)
		return nil
	}

	allPRs, ok := w.fetchAllPRs(ctx, cursorState.LastUpdatedAt)
	if !ok {
		return nil
	}

	maxUpdatedAt := w.processPRs(ctx, cursorState, allPRs)

	if maxUpdatedAt.After(cursorState.LastUpdatedAt) {
		cursorState.LastUpdatedAt = maxUpdatedAt
	}

	if err := SaveCursor(ctx, w.cursorPath, cursorState); err != nil {
		glog.Errorf("failed to save cursor err=%v", err)
	}
	return nil
}

// fetchAllPRs paginates GitHub search results. Returns (prs, true) on success,
// (nil, false) if the caller should abort the poll cycle.
func (w *watcher) fetchAllPRs(
	ctx context.Context,
	since time.Time,
) ([]PullRequest, bool) {
	page := 1
	var allPRs []PullRequest

	for {
		result, err := w.ghClient.SearchPRs(ctx, w.scope, since, page)
		if err != nil {
			glog.Errorf("github search failed err=%v", err)
			return nil, false
		}

		allPRs = append(allPRs, result.PullRequests...)

		if !result.HasNextPage {
			break
		}
		page = result.NextPage

		select {
		case <-ctx.Done():
			return nil, false
		default:
		}
	}
	return allPRs, true
}

// processPRs iterates over fetched PRs, publishes commands, and returns the max updated-at seen.
func (w *watcher) processPRs(
	ctx context.Context,
	cursorState Cursor,
	allPRs []PullRequest,
) time.Time {
	since := cursorState.LastUpdatedAt
	maxUpdatedAt := since
	headSHACache := make(map[string]string)

	for _, pr := range allPRs {
		if ShouldSkipPR(pr, w.botAllowlist) {
			glog.V(3).Infof("skipping pr=%s/%s#%d reason=filtered", pr.Owner, pr.Repo, pr.Number)
			continue
		}

		taskIDStr := DeriveTaskID(pr.Owner, pr.Repo, pr.Number).String()

		headSHA, err := w.fetchHeadSHA(ctx, pr, taskIDStr, headSHACache)
		if err != nil {
			glog.Errorf("get head sha failed pr=%s/%s#%d err=%v", pr.Owner, pr.Repo, pr.Number, err)
			continue
		}

		if w.handlePR(ctx, cursorState, pr, taskIDStr, headSHA) {
			if pr.UpdatedAt.After(maxUpdatedAt) {
				maxUpdatedAt = pr.UpdatedAt
			}
		}
	}
	return maxUpdatedAt
}

// handlePR processes a single PR: publishes commands, updates cursor SHA.
// Returns true if the PR was processed successfully (cursor should advance).
func (w *watcher) handlePR(
	ctx context.Context,
	cursorState Cursor,
	pr PullRequest,
	taskIDStr, headSHA string,
) bool {
	knownSHA, exists := cursorState.HeadSHAs[taskIDStr]
	switch {
	case !exists:
		return w.publishCreate(ctx, cursorState, pr, taskIDStr, headSHA)
	case knownSHA != headSHA:
		return w.publishForcePush(ctx, cursorState, pr, taskIDStr, knownSHA, headSHA)
	default:
		glog.V(3).
			Infof("no change, skipping pr=%s/%s#%d taskID=%s", pr.Owner, pr.Repo, pr.Number, taskIDStr)
		return true
	}
}

func (w *watcher) publishCreate(
	ctx context.Context,
	cursorState Cursor,
	pr PullRequest,
	taskIDStr, headSHA string,
) bool {
	cmd := agentlib.CreateTaskCommand{
		TaskIdentifier: agentlib.TaskIdentifier(taskIDStr),
		Frontmatter:    buildFrontmatter(pr, taskIDStr, w.stage),
		Body:           buildTaskBody(pr),
	}
	if err := w.publisher.PublishCreate(ctx, cmd); err != nil {
		glog.Errorf("publish create-task failed pr=%s err=%v", pr.HTMLURL, err)
		return false
	}
	cursorState.HeadSHAs[taskIDStr] = headSHA
	glog.V(2).
		Infof("published CreateTaskCommand pr=%s/%s#%d taskID=%s", pr.Owner, pr.Repo, pr.Number, taskIDStr)
	return true
}

func (w *watcher) publishForcePush(
	ctx context.Context,
	cursorState Cursor,
	pr PullRequest,
	taskIDStr, oldSHA, newSHA string,
) bool {
	heading := fmt.Sprintf("## Outdated by force-push %s", oldSHA)
	cmd := agentlib.UpdateFrontmatterCommand{
		TaskIdentifier: agentlib.TaskIdentifier(taskIDStr),
		Updates: agentlib.TaskFrontmatter{
			"phase":         "planning",
			"status":        "in_progress",
			"trigger_count": 0,
		},
		Body: &agentlib.BodySection{Heading: heading, Section: heading + "\n"},
	}
	if err := w.publisher.PublishUpdateFrontmatter(ctx, cmd); err != nil {
		glog.Errorf("publish update-frontmatter failed pr=%s err=%v", pr.HTMLURL, err)
		return false
	}
	cursorState.HeadSHAs[taskIDStr] = newSHA
	glog.V(2).
		Infof("published UpdateFrontmatterCommand pr=%s/%s#%d taskID=%s", pr.Owner, pr.Repo, pr.Number, taskIDStr)
	return true
}

func (w *watcher) fetchHeadSHA(
	ctx context.Context,
	pr PullRequest,
	taskIDStr string,
	cache map[string]string,
) (string, error) {
	if sha, ok := cache[taskIDStr]; ok {
		return sha, nil
	}
	sha, err := w.ghClient.GetHeadSHA(ctx, pr.Owner, pr.Repo, pr.Number)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "get head sha pr=%s/%s#%d", pr.Owner, pr.Repo, pr.Number)
	}
	cache[taskIDStr] = sha
	return sha, nil
}

func buildTaskBody(pr PullRequest) string {
	return fmt.Sprintf("# PR Review: %s\n\n%s\n", pr.Title, pr.HTMLURL)
}

func buildFrontmatter(
	pr PullRequest,
	taskIDStr, stage string,
) agentlib.TaskFrontmatter {
	return agentlib.TaskFrontmatter{
		"assignee":        "pr-reviewer-agent",
		"phase":           "planning",
		"status":          "in_progress",
		"stage":           stage,
		"task_identifier": taskIDStr,
		"title":           pr.Title,
	}
}
