// Copyright (c) 2025 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package git

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"github.com/bborbe/errors"
)

var cloneURLSegmentRegexp = regexp.MustCompile(`^[a-zA-Z0-9._\-]+$`)

// ParseCloneURL converts a git clone URL to a relative bare-repo path:
// "<host>/<owner>/<repo>.git". Returns an error for malformed or unsafe URLs.
func ParseCloneURL(ctx context.Context, rawURL string) (string, error) {
	if rawURL == "" {
		return "", errors.Errorf(ctx, "clone URL must not be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.Wrapf(ctx, err, "parse clone URL")
	}

	if parsed.Host == "" {
		return "", errors.Errorf(ctx, "clone URL missing host: %s", rawURL)
	}

	// Strip leading '/' and trailing '.git', then split into segments.
	path := strings.TrimPrefix(parsed.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	segments := strings.Split(path, "/")
	if len(segments) != 2 {
		return "", errors.Errorf(
			ctx,
			"clone URL path must have exactly 2 segments (<owner>/<repo>), got %d: %s",
			len(segments),
			rawURL,
		)
	}

	for _, seg := range segments {
		if err := validateCloneURLSegment(ctx, seg); err != nil {
			return "", err
		}
	}

	return parsed.Host + "/" + segments[0] + "/" + segments[1] + ".git", nil
}

func validateCloneURLSegment(ctx context.Context, seg string) error {
	if seg == "" {
		return errors.Errorf(ctx, "clone URL segment must not be empty")
	}
	if seg == "." || seg == ".." {
		return errors.Errorf(ctx, "clone URL segment must not be '.' or '..': %s", seg)
	}
	if !cloneURLSegmentRegexp.MatchString(seg) {
		return errors.Errorf(ctx, "clone URL segment contains invalid characters: %s", seg)
	}
	return nil
}
