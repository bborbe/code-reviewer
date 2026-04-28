// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("validateRepoScope",
	func(scope string, expectError bool) {
		ctx := context.Background()
		err := validateRepoScope(ctx, scope)
		if expectError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}
	},
	Entry("simple username", "bborbe", false),
	Entry("org with hyphen", "my-org", false),
	Entry("org with dot", "org.name", false),
	Entry("org with underscore", "org_name", false),
	Entry("mixed case and digits", "Org123", false),
	Entry("space injection", "user is:issue", true),
	Entry("semicolon injection", "user;drop", true),
	Entry("empty string", "", true),
	Entry("plus injection", "user+more", true),
)
