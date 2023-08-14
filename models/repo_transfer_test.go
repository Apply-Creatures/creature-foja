// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestGetPendingTransferIDs(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	reciepient := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	pendingTransfer := unittest.AssertExistsAndLoadBean(t, &RepoTransfer{RecipientID: reciepient.ID, DoerID: doer.ID})

	pendingTransferIDs, err := GetPendingTransferIDs(db.DefaultContext, reciepient.ID, doer.ID)
	assert.NoError(t, err)
	if assert.Len(t, pendingTransferIDs, 1) {
		assert.EqualValues(t, pendingTransfer.ID, pendingTransferIDs[0])
	}
}
