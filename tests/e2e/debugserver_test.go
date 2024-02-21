// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// This "test" is meant to be run with `make test-e2e-debugserver` and will just
// keep open a gitea instance in a test environment (with the data from
// `models/fixtures`) on port 3000. This is useful for debugging e2e tests, for
// example with the playwright vscode extension.

//nolint:forbidigo
package e2e

import (
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"testing"

	"code.gitea.io/gitea/modules/setting"
)

func TestDebugserver(t *testing.T) {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	onGiteaRun(t, func(*testing.T, *url.URL) {
		println(setting.AppURL)
		<-done
	})
}
