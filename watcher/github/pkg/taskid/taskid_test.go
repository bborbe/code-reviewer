// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package taskid_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/code-reviewer/watcher/github/pkg/taskid"
)

var _ = Describe("TaskID", func() {
	var prWatcherNamespace = uuid.MustParse("7d4b3e5f-8a21-4c9d-b036-2e5f7a8c1d0e")

	Describe("Derive", func() {
		It("is deterministic — same inputs always produce the same UUID", func() {
			a := taskid.Derive("bborbe", "code-reviewer", 42)
			b := taskid.Derive("bborbe", "code-reviewer", 42)
			Expect(a).To(Equal(b))
		})

		It("produces different UUIDs for different owner/repo/number combos", func() {
			a := taskid.Derive("bborbe", "code-reviewer", 42)
			b := taskid.Derive("bborbe", "code-reviewer", 43)
			c := taskid.Derive("bborbe", "other-repo", 42)
			d := taskid.Derive("other-org", "code-reviewer", 42)
			Expect(a).NotTo(Equal(b))
			Expect(a).NotTo(Equal(c))
			Expect(a).NotTo(Equal(d))
		})

		It("produces the expected pinned UUID for bborbe/code-reviewer#42", func() {
			expected := uuid.NewSHA1(prWatcherNamespace, []byte("bborbe/code-reviewer#42"))
			Expect(taskid.Derive("bborbe", "code-reviewer", 42)).To(Equal(expected))
		})
	})
})
