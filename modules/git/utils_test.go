// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import "testing"

// This file contains utility functions that are used across multiple tests,
// but not in production code.

func skipIfSHA256NotSupported(t *testing.T) {
	if isGogit || CheckGitVersionAtLeast("2.42") != nil {
		t.Skip("skipping because installed Git version doesn't support SHA256")
	}
}
