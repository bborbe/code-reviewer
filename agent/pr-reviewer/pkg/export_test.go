// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pkg

import "context"

// VerdictPayloadForTest re-exports the unexported verdictPayload so
// review_test.go (in the pkg_test package) can assert on the parsed
// values without exposing the type to production callers.
type VerdictPayloadForTest = verdictPayload

// ExtractVerdictForTest re-exports the unexported extractVerdict so
// review_test.go (in the pkg_test package) can table-test parsing
// across the various LLM response shapes Claude produces.
func ExtractVerdictForTest(raw string) (VerdictPayloadForTest, error) {
	return extractVerdict(context.Background(), raw)
}

// NewGHTokenCheckStepWithURLForTest constructs a ghTokenCheckStep
// pointed at a custom URL (httptest.Server in tests). Production code
// should use NewGHTokenCheckStep which hardcodes the GitHub URL.
func NewGHTokenCheckStepWithURLForTest(token, url string) *ghTokenCheckStep {
	return newGHTokenCheckStep(token, url)
}
