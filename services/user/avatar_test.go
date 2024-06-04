// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package user

import (
	"bytes"
	"image"
	"image/png"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/storage"
	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestUserDeleteAvatar(t *testing.T) {
	myImage := image.NewRGBA(image.Rect(0, 0, 1, 1))
	var buff bytes.Buffer
	png.Encode(&buff, myImage)

	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	err := UploadAvatar(db.DefaultContext, user, buff.Bytes())
	assert.NoError(t, err)
	verification := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	assert.NotEqual(t, "", verification.Avatar)

	t.Run("AtomicStorageFailure", func(t *testing.T) {
		defer test.MockVariableValue[storage.ObjectStorage](&storage.Avatars, storage.UninitializedStorage)()
		err = DeleteAvatar(db.DefaultContext, user)
		assert.Error(t, err)
		verification := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		assert.True(t, verification.UseCustomAvatar)
	})

	err = DeleteAvatar(db.DefaultContext, user)
	assert.NoError(t, err)

	verification = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	assert.Equal(t, "", verification.Avatar)
}
