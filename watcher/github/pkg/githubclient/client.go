// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package githubclient

import (
	"context"
	"fmt"
	"path"
	"time"

	"github.com/bborbe/errors"
	gogithub "github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
)

// PullRequest holds the fields the watcher needs from a GitHub PR.
type PullRequest struct {
	GlobalID    int64
	Number      int
	Owner       string
	Repo        string
	Title       string
	HTMLURL     string
	HeadSHA     string
	AuthorLogin string
	IsDraft     bool
	UpdatedAt   time.Time
}

// SearchResult is the result of a single paginated search call.
type SearchResult struct {
	PullRequests  []PullRequest
	HasNextPage   bool
	NextPage      int
	RateRemaining int
	RateResetAt   time.Time
}

//counterfeiter:generate -o mocks/github_client.go --fake-name GitHubClient . GitHubClient

// GitHubClient abstracts the GitHub API calls.
type GitHubClient interface {
	// SearchPRs issues a GitHub Search query for open PRs updated since cursor.
	// page=1 for the first call; use SearchResult.NextPage for subsequent calls.
	// PullRequest.HeadSHA in the result is empty — call GetHeadSHA to fetch it.
	SearchPRs(ctx context.Context, scope string, since time.Time, page int) (SearchResult, error)

	// GetHeadSHA fetches the head commit SHA for a single PR. The Search
	// API does NOT return head SHA, so the poll loop must call this for
	// any PR it needs head-SHA tracking for (force-push detection).
	GetHeadSHA(ctx context.Context, owner, repo string, number int) (string, error)
}

// NewGitHubClient returns a GitHubClient backed by the real GitHub API.
func NewGitHubClient(token string) GitHubClient {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &githubClient{
		client: gogithub.NewClient(httpClient),
	}
}

type githubClient struct {
	client *gogithub.Client
}

func (c *githubClient) SearchPRs(
	ctx context.Context,
	scope string,
	since time.Time,
	page int,
) (SearchResult, error) {
	query := fmt.Sprintf(
		"is:pr is:open archived:false user:%s updated:>=%s",
		scope,
		since.UTC().Format(time.RFC3339),
	)
	opts := &gogithub.SearchOptions{
		ListOptions: gogithub.ListOptions{
			Page:    page,
			PerPage: 100,
		},
	}

	result, resp, err := c.client.Search.Issues(ctx, query, opts)
	if err != nil {
		return SearchResult{}, errors.Wrapf(ctx, err, "search github prs scope=%s", scope)
	}

	prs := make([]PullRequest, 0, len(result.Issues))
	for _, issue := range result.Issues {
		repoURL := issue.GetRepositoryURL()
		owner, repo := parseOwnerRepo(repoURL)
		prs = append(prs, PullRequest{
			GlobalID:    issue.GetID(),
			Number:      issue.GetNumber(),
			Owner:       owner,
			Repo:        repo,
			Title:       issue.GetTitle(),
			HTMLURL:     issue.GetHTMLURL(),
			HeadSHA:     "",
			AuthorLogin: issue.GetUser().GetLogin(),
			IsDraft:     issue.GetDraft(),
			UpdatedAt:   issue.GetUpdatedAt().Time,
		})
	}

	return SearchResult{
		PullRequests:  prs,
		HasNextPage:   resp.NextPage > 0,
		NextPage:      resp.NextPage,
		RateRemaining: resp.Rate.Remaining,
		RateResetAt:   resp.Rate.Reset.Time,
	}, nil
}

func (c *githubClient) GetHeadSHA(
	ctx context.Context,
	owner, repo string,
	number int,
) (string, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "get pull request %s/%s#%d", owner, repo, number)
	}
	return pr.GetHead().GetSHA(), nil
}

// parseOwnerRepo extracts owner and repo from a GitHub API repository URL.
// Input format: https://api.github.com/repos/{owner}/{repo}
func parseOwnerRepo(repoURL string) (owner, repo string) {
	dir, repoName := path.Split(repoURL)
	_, ownerName := path.Split(path.Clean(dir))
	return ownerName, repoName
}
