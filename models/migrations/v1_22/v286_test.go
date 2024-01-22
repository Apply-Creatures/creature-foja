// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1_22 //nolint

import (
	"testing"

	"code.gitea.io/gitea/models/migrations/base"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm"
)

func PrepareOldRepository(t *testing.T) (*xorm.Engine, func()) {
	type Repository struct { // old struct
		ID               int64  `xorm:"pk autoincr"`
		ObjectFormatName string `xorm:"VARCHAR(6) NOT NULL DEFAULT 'sha1'"`
	}

	type CommitStatus struct { // old struct
		ID          int64  `xorm:"pk autoincr"`
		ContextHash string `xorm:"char(40)"`
	}

	type Comment struct { // old struct
		ID        int64  `xorm:"pk autoincr"`
		CommitSHA string `xorm:"VARCHAR(40)"`
	}

	type PullRequest struct { // old struct
		ID             int64  `xorm:"pk autoincr"`
		MergeBase      string `xorm:"VARCHAR(40)"`
		MergedCommitID string `xorm:"VARCHAR(40)"`
	}

	type Review struct { // old struct
		ID       int64  `xorm:"pk autoincr"`
		CommitID string `xorm:"VARCHAR(40)"`
	}

	type ReviewState struct { // old struct
		ID        int64  `xorm:"pk autoincr"`
		CommitSHA string `xorm:"VARCHAR(40)"`
	}

	type RepoArchiver struct { // old struct
		ID       int64  `xorm:"pk autoincr"`
		CommitID string `xorm:"VARCHAR(40)"`
	}

	type Release struct { // old struct
		ID   int64  `xorm:"pk autoincr"`
		Sha1 string `xorm:"VARCHAR(40)"`
	}

	type RepoIndexerStatus struct { // old struct
		ID        int64  `xorm:"pk autoincr"`
		CommitSha string `xorm:"VARCHAR(40)"`
	}

	// Prepare and load the testing database
	return base.PrepareTestEnv(t, 0, new(Repository), new(CommitStatus), new(Comment), new(PullRequest), new(Review), new(ReviewState), new(RepoArchiver), new(Release), new(RepoIndexerStatus))
}

func Test_RepositoryFormat(t *testing.T) {
	x, deferable := PrepareOldRepository(t)
	defer deferable()

	type Repository struct {
		ID               int64  `xorm:"pk autoincr"`
		ObjectFormatName string `xorg:"not null default('sha1')"`
	}

	repo := new(Repository)

	// check we have some records to migrate
	count, err := x.Count(new(Repository))
	assert.NoError(t, err)
	assert.EqualValues(t, 4, count)

	assert.NoError(t, AdjustDBForSha256(x))

	repo.ID = 20
	repo.ObjectFormatName = "sha256"
	_, err = x.Insert(repo)
	assert.NoError(t, err)

	count, err = x.Count(new(Repository))
	assert.NoError(t, err)
	assert.EqualValues(t, 5, count)

	repo = new(Repository)
	ok, err := x.ID(2).Get(repo)
	assert.NoError(t, err)
	assert.EqualValues(t, true, ok)
	assert.EqualValues(t, "sha1", repo.ObjectFormatName)

	repo = new(Repository)
	ok, err = x.ID(20).Get(repo)
	assert.NoError(t, err)
	assert.EqualValues(t, true, ok)
	assert.EqualValues(t, "sha256", repo.ObjectFormatName)
}
