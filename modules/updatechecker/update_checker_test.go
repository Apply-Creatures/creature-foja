// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package updatechecker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDNSUpdate(t *testing.T) {
	version, err := getVersionDNS("release.forgejo.org")
	assert.NoError(t, err)
	assert.NotEmpty(t, version)
}
