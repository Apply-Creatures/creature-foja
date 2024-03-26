// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"testing"

	"code.gitea.io/gitea/models/migrations/base"

	"github.com/stretchr/testify/assert"
)

func Test_RemoveSSHSignaturesFromReleaseNotes(t *testing.T) {
	// A reduced mock of the `repo_model.Release` struct.
	type Release struct {
		ID   int64  `xorm:"pk autoincr"`
		Note string `xorm:"TEXT"`
	}

	x, deferable := base.PrepareTestEnv(t, 0, new(Release))
	defer deferable()

	assert.NoError(t, RemoveSSHSignaturesFromReleaseNotes(x))

	var releases []Release
	err := x.Table("release").OrderBy("id ASC").Find(&releases)
	assert.NoError(t, err)
	assert.Len(t, releases, 3)

	assert.Equal(t, "", releases[0].Note)
	assert.Equal(t, "A message.\n", releases[1].Note)
	assert.Equal(t, "no signature present here", releases[2].Note)
}
