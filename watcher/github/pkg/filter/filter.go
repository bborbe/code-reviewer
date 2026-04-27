// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filter

import "github.com/bborbe/code-reviewer/watcher/github/pkg/githubclient"

// IsBotAuthor returns true if the PR author login matches any allowlist entry (exact match).
func IsBotAuthor(pr githubclient.PullRequest, allowlist []string) bool {
	for _, entry := range allowlist {
		if pr.AuthorLogin == entry {
			return true
		}
	}
	return false
}

// ShouldSkip returns true if the PR should be filtered out (draft or bot-authored).
func ShouldSkip(pr githubclient.PullRequest, botAllowlist []string) bool {
	return pr.IsDraft || IsBotAuthor(pr, botAllowlist)
}
