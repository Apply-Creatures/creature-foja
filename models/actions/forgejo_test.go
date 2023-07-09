// SPDX-License-Identifier: MIT

package actions

import (
	"crypto/subtle"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
)

func TestActions_RegisterRunner(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	labels := []string{}
	name := "runner"
	version := "v1.2.3"
	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, labels, name, version)
	assert.NoError(t, err)
	assert.EqualValues(t, name, runner.Name)

	assert.EqualValues(t, 1, subtle.ConstantTimeCompare([]byte(runner.TokenHash), []byte(auth_model.HashToken(token, runner.TokenSalt))), "the token cannot be verified with the same method as routers/api/actions/runner/interceptor.go as of 8228751c55d6a4263f0fec2932ca16181c09c97d")
}
