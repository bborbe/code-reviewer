// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// PRInfo contains parsed information from a GitHub pull request URL.
type PRInfo struct {
	Owner   string
	Repo    string
	Number  int
	RepoURL string
}

// ParsePRURL parses a GitHub PR URL and extracts owner, repo, PR number, and repo URL.
// Expected format: https://github.com/{owner}/{repo}/pull/{number}
// Returns error for non-GitHub URLs or malformed GitHub URLs.
//
// Deprecated: Use prurl.Parse instead, which supports both GitHub and Bitbucket Server URLs.
func ParsePRURL(rawURL string) (*PRInfo, error) {
	if rawURL == "" {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	if u.Scheme == "" {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	if u.Host != "github.com" {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	// Parse path: /{owner}/{repo}/pull/{number}
	path := strings.TrimSuffix(u.Path, "/")
	parts := strings.Split(strings.TrimPrefix(path, "/"), "/")

	if len(parts) != 4 {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	owner := parts[0]
	repo := parts[1]
	pullKeyword := parts[2]
	numberStr := parts[3]

	if owner == "" || repo == "" {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	if pullKeyword != "pull" {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return nil, fmt.Errorf("unsupported URL format: %s", rawURL)
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s", owner, repo)

	return &PRInfo{
		Owner:   owner,
		Repo:    repo,
		Number:  number,
		RepoURL: repoURL,
	}, nil
}
