// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

// IsBotAuthor returns true if the PR author login matches any allowlist entry (exact match).
func IsBotAuthor(pr PullRequest, allowlist []string) bool {
	for _, entry := range allowlist {
		if pr.AuthorLogin == entry {
			return true
		}
	}
	return false
}

// ShouldSkip returns true if the PR should be filtered out (draft or bot-authored).
func ShouldSkipPR(pr PullRequest, botAllowlist []string) bool {
	return pr.IsDraft || IsBotAuthor(pr, botAllowlist)
}
