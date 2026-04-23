// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package verdict

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Verdict represents the review verdict type
type Verdict string

const (
	VerdictApprove        Verdict = "approve"
	VerdictRequestChanges Verdict = "request-changes"
	VerdictComment        Verdict = "comment"
)

// Result holds the verdict and reason
type Result struct {
	Verdict Verdict
	Reason  string
}

// jsonVerdict is used for unmarshaling JSON verdict blocks
type jsonVerdict struct {
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// tryParseJSONLine attempts to parse a JSON verdict from a single line
// Returns the parsed Result and true if valid, or zero Result and false otherwise
func tryParseJSONLine(line string) (Result, bool) {
	// Extract JSON (remove code fence markers if present)
	jsonStr := strings.TrimSpace(line)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	// Try to parse JSON
	var jv jsonVerdict
	if err := json.Unmarshal([]byte(jsonStr), &jv); err != nil {
		return Result{}, false
	}

	// Validate verdict value
	var v Verdict
	switch jv.Verdict {
	case "approve":
		v = VerdictApprove
	case "request-changes":
		v = VerdictRequestChanges
	case "comment":
		v = VerdictComment
	default:
		// Unknown verdict value - fall back to heuristic
		return Result{}, false
	}

	return Result{
		Verdict: v,
		Reason:  jv.Reason,
	}, true
}

// parseJSONVerdict scans the last 50 lines of review text for a JSON verdict block
// Returns the parsed Result and true if found, or zero Result and false if not found
func parseJSONVerdict(reviewText string) (Result, bool) {
	lines := strings.Split(reviewText, "\n")

	// Search only the last 50 lines to avoid false matches in code examples
	startIdx := 0
	if len(lines) > 50 {
		startIdx = len(lines) - 50
	}

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Skip code fence markers
		if line == "```json" || line == "```" {
			continue
		}

		// Try to find JSON on this line
		if strings.Contains(line, `"verdict"`) && strings.Contains(line, `"reason"`) {
			if result, ok := tryParseJSONLine(line); ok {
				return result, true
			}
		}
	}

	return Result{}, false
}

// Parse analyzes Claude review output and determines the appropriate verdict
func Parse(reviewText string) Result {
	// First try to extract JSON verdict
	if result, ok := parseJSONVerdict(reviewText); ok {
		return result
	}

	// Fall back to heuristic section scanning
	if reviewText == "" {
		return Result{
			Verdict: VerdictComment,
			Reason:  "empty review text",
		}
	}

	// Check if review has expected sections
	hasReviewSections := hasExpectedReviewSections(reviewText)

	// Look for Must Fix section (case-insensitive, h2 or h3)
	mustFixPattern := regexp.MustCompile(`(?i)^##+ Must Fix`)
	lines := strings.Split(reviewText, "\n")

	mustFixIndex := -1
	for i, line := range lines {
		if mustFixPattern.MatchString(strings.TrimSpace(line)) {
			mustFixIndex = i
			break
		}
	}

	// No Must Fix section found
	if mustFixIndex == -1 {
		if hasReviewSections {
			return Result{
				Verdict: VerdictApprove,
				Reason:  "no must-fix section",
			}
		}
		return Result{
			Verdict: VerdictComment,
			Reason:  "unparseable review format",
		}
	}

	// Must Fix section found - check if it has content
	hasMustFixContent := checkMustFixContent(lines, mustFixIndex)

	if hasMustFixContent {
		return Result{
			Verdict: VerdictRequestChanges,
			Reason:  "must-fix items found",
		}
	}

	return Result{
		Verdict: VerdictApprove,
		Reason:  "no must-fix items",
	}
}

// hasExpectedReviewSections checks if the review has expected sections like Should Fix or Nice to Have
func hasExpectedReviewSections(reviewText string) bool {
	text := strings.ToLower(reviewText)
	return strings.Contains(text, "should fix") || strings.Contains(text, "nice to have")
}

// checkMustFixContent determines if Must Fix section has actual content (not just "None")
func checkMustFixContent(lines []string, mustFixIndex int) bool {
	nextHeadingPattern := regexp.MustCompile(`^##+ `)
	hasNonEmptyContent := false

	// Look at lines after the Must Fix heading
	for i := mustFixIndex + 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Stop at next heading
		if nextHeadingPattern.MatchString(line) {
			break
		}

		// Skip empty lines and horizontal rules
		if line == "" || isHorizontalRule(line) {
			continue
		}

		// Check if line is a "None" indicator (case-insensitive)
		if isNoneIndicator(line) {
			continue
		}

		// Found non-empty, non-"None" content
		hasNonEmptyContent = true
		break
	}

	return hasNonEmptyContent
}

// isHorizontalRule checks if a line is a markdown horizontal rule (---, ***, ___).
func isHorizontalRule(line string) bool {
	cleaned := strings.TrimSpace(line)
	if len(cleaned) < 3 {
		return false
	}
	for _, ch := range cleaned {
		if ch != '-' && ch != '*' && ch != '_' {
			return false
		}
	}
	return true
}

// isNoneIndicator checks if a line indicates "no issues" (e.g., "None", "*None*", "None identified.")
func isNoneIndicator(line string) bool {
	// Remove markdown formatting and punctuation
	cleaned := strings.Trim(line, "*_~` .")
	cleaned = strings.ToLower(strings.TrimSpace(cleaned))

	// Check for common "none" patterns
	if cleaned == "none" {
		return true
	}
	if strings.HasPrefix(cleaned, "none ") {
		return true
	}
	if strings.Contains(cleaned, "no issues") {
		return true
	}

	return false
}

// StripJSONVerdict removes the JSON verdict line (and surrounding code fence if present)
// from the review text. Returns the cleaned review text for posting as a PR comment.
// If no JSON verdict found, returns the text unchanged.
func StripJSONVerdict(reviewText string) string {
	lines := strings.Split(reviewText, "\n")
	linesToRemove := findVerdictLinesToRemove(lines)

	if len(linesToRemove) == 0 {
		return reviewText
	}

	return buildCleanedText(lines, linesToRemove)
}

// findVerdictLinesToRemove scans lines and returns a map of line indices to remove
func findVerdictLinesToRemove(lines []string) map[int]bool {
	startIdx := calculateStartIndex(lines)
	linesToRemove := make(map[int]bool)
	inCodeFence := false
	fenceStartIdx := -1

	for i := startIdx; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		if handleCodeFenceStart(line, &inCodeFence, &fenceStartIdx, i) {
			continue
		}

		if handleCodeFenceEnd(line, &inCodeFence, &fenceStartIdx) {
			continue
		}

		if containsVerdictJSON(line) {
			processVerdictLine(lines, i, line, inCodeFence, fenceStartIdx, linesToRemove)
		}
	}

	return linesToRemove
}

// calculateStartIndex returns the index to start searching (last 50 lines)
func calculateStartIndex(lines []string) int {
	if len(lines) > 50 {
		return len(lines) - 50
	}
	return 0
}

// handleCodeFenceStart checks for code fence start and updates state
func handleCodeFenceStart(line string, inCodeFence *bool, fenceStartIdx *int, i int) bool {
	if line == "```json" && !*inCodeFence {
		*inCodeFence = true
		*fenceStartIdx = i
		return true
	}
	return false
}

// handleCodeFenceEnd checks for code fence end and updates state
func handleCodeFenceEnd(line string, inCodeFence *bool, fenceStartIdx *int) bool {
	if line == "```" && *inCodeFence {
		*inCodeFence = false
		*fenceStartIdx = -1
		return true
	}
	return false
}

// containsVerdictJSON checks if a line contains verdict JSON markers
func containsVerdictJSON(line string) bool {
	return strings.Contains(line, `"verdict"`) && strings.Contains(line, `"reason"`)
}

// processVerdictLine validates and marks lines for removal if valid verdict found
func processVerdictLine(
	lines []string,
	i int,
	line string,
	inCodeFence bool,
	fenceStartIdx int,
	linesToRemove map[int]bool,
) {
	if !isValidVerdictJSON(line) {
		return
	}

	// Mark verdict line for removal
	linesToRemove[i] = true

	// If inside code fence, mark fence lines too
	if inCodeFence && fenceStartIdx >= 0 {
		markCodeFenceLinesForRemoval(lines, i, fenceStartIdx, linesToRemove)
	}
}

// isValidVerdictJSON checks if the line contains a valid verdict JSON
func isValidVerdictJSON(line string) bool {
	jsonStr := strings.TrimSpace(line)
	jsonStr = strings.TrimPrefix(jsonStr, "```json")
	jsonStr = strings.TrimSuffix(jsonStr, "```")
	jsonStr = strings.TrimSpace(jsonStr)

	var jv jsonVerdict
	if err := json.Unmarshal([]byte(jsonStr), &jv); err != nil {
		return false
	}

	return jv.Verdict != ""
}

// markCodeFenceLinesForRemoval marks fence start and end lines for removal
func markCodeFenceLinesForRemoval(
	lines []string,
	currentIdx int,
	fenceStartIdx int,
	linesToRemove map[int]bool,
) {
	linesToRemove[fenceStartIdx] = true

	// Find and mark the closing fence
	for j := currentIdx + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "```" {
			linesToRemove[j] = true
			break
		}
	}
}

// buildCleanedText constructs the final text without removed lines
func buildCleanedText(lines []string, linesToRemove map[int]bool) string {
	var cleaned []string
	for i, line := range lines {
		if !linesToRemove[i] {
			cleaned = append(cleaned, line)
		}
	}

	result := strings.Join(cleaned, "\n")
	return strings.TrimRight(result, "\n")
}
