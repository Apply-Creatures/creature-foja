// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"testing"
	"time"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/git"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToCommitMeta(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	headRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	sha1 := git.Sha1ObjectFormat
	signature := &git.Signature{Name: "Test Signature", Email: "test@email.com", When: time.Unix(0, 0)}
	tag := &git.Tag{
		Name:    "Test Tag",
		ID:      sha1.EmptyObjectID(),
		Object:  sha1.EmptyObjectID(),
		Type:    "Test Type",
		Tagger:  signature,
		Message: "Test Message",
	}

	commitMeta := ToCommitMeta(headRepo, tag)

	assert.NotNil(t, commitMeta)
	assert.EqualValues(t, &api.CommitMeta{
		SHA:     sha1.EmptyObjectID().String(),
		URL:     util.URLJoin(headRepo.APIURL(), "git/commits", sha1.EmptyObjectID().String()),
		Created: time.Unix(0, 0),
	}, commitMeta)
}
