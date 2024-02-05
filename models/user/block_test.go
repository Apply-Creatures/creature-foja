// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func TestIsBlocked(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	assert.True(t, user_model.IsBlocked(db.DefaultContext, 4, 1))

	// Simple test cases to ensure the function can also respond with false.
	assert.False(t, user_model.IsBlocked(db.DefaultContext, 1, 1))
	assert.False(t, user_model.IsBlocked(db.DefaultContext, 3, 2))
}

func TestIsBlockedMultiple(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	assert.True(t, user_model.IsBlockedMultiple(db.DefaultContext, []int64{4}, 1))
	assert.True(t, user_model.IsBlockedMultiple(db.DefaultContext, []int64{4, 3, 4, 5}, 1))

	// Simple test cases to ensure the function can also respond with false.
	assert.False(t, user_model.IsBlockedMultiple(db.DefaultContext, []int64{1}, 1))
	assert.False(t, user_model.IsBlockedMultiple(db.DefaultContext, []int64{3, 4, 1}, 2))
}

func TestUnblockUser(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	assert.True(t, user_model.IsBlocked(db.DefaultContext, 4, 1))

	assert.NoError(t, user_model.UnblockUser(db.DefaultContext, 4, 1))

	// Simple test cases to ensure the function can also respond with false.
	assert.False(t, user_model.IsBlocked(db.DefaultContext, 4, 1))
}

func TestListBlockedUsers(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	blockedUsers, err := user_model.ListBlockedUsers(db.DefaultContext, 4, db.ListOptions{})
	assert.NoError(t, err)
	if assert.Len(t, blockedUsers, 1) {
		assert.EqualValues(t, 1, blockedUsers[0].ID)
		// The function returns the created Unix of the block, not that of the user.
		assert.EqualValues(t, 1671607299, blockedUsers[0].CreatedUnix)
	}
}

func TestListBlockedByUsersID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	blockedByUserIDs, err := user_model.ListBlockedByUsersID(db.DefaultContext, 1)
	assert.NoError(t, err)
	if assert.Len(t, blockedByUserIDs, 1) {
		assert.EqualValues(t, 4, blockedByUserIDs[0])
	}
}

func TestCountBlockedUsers(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	count, err := user_model.CountBlockedUsers(db.DefaultContext, 4)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, count)

	count, err = user_model.CountBlockedUsers(db.DefaultContext, 1)
	assert.NoError(t, err)
	assert.EqualValues(t, 0, count)
}
