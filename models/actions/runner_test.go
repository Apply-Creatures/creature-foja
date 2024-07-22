// SPDX-License-Identifier: MIT

package actions

import (
	"encoding/binary"
	"fmt"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateSecret checks that ActionRunner.UpdateSecret() sets the Token,
// TokenSalt and TokenHash fields based on the specified token.
func TestUpdateSecret(t *testing.T) {
	runner := ActionRunner{}
	token := "0123456789012345678901234567890123456789"

	err := runner.UpdateSecret(token)

	require.NoError(t, err)
	assert.Equal(t, token, runner.Token)
	assert.Regexp(t, "^[0-9a-f]{32}$", runner.TokenSalt)
	assert.Equal(t, runner.TokenHash, auth_model.HashToken(token, runner.TokenSalt))
}

func TestDeleteRunner(t *testing.T) {
	const recordID = 12345678
	assert.NoError(t, unittest.PrepareTestDatabase())
	before := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	err := DeleteRunner(db.DefaultContext, recordID)
	assert.NoError(t, err)

	var after ActionRunner
	found, err := db.GetEngine(db.DefaultContext).ID(recordID).Unscoped().Get(&after)
	assert.NoError(t, err)
	assert.True(t, found)

	// Most fields (namely Name, Version, OwnerID, RepoID, Description, Base, RepoRange,
	// TokenHash, TokenSalt, LastOnline, LastActive, AgentLabels and Created) are unaffected
	assert.Equal(t, before.Name, after.Name)
	assert.Equal(t, before.Version, after.Version)
	assert.Equal(t, before.OwnerID, after.OwnerID)
	assert.Equal(t, before.RepoID, after.RepoID)
	assert.Equal(t, before.Description, after.Description)
	assert.Equal(t, before.Base, after.Base)
	assert.Equal(t, before.RepoRange, after.RepoRange)
	assert.Equal(t, before.TokenHash, after.TokenHash)
	assert.Equal(t, before.TokenSalt, after.TokenSalt)
	assert.Equal(t, before.LastOnline, after.LastOnline)
	assert.Equal(t, before.LastActive, after.LastActive)
	assert.Equal(t, before.AgentLabels, after.AgentLabels)
	assert.Equal(t, before.Created, after.Created)

	// Deleted contains a value
	assert.NotNil(t, after.Deleted)

	// UUID was modified
	assert.NotEqual(t, before.UUID, after.UUID)
	// UUID starts with ffffffff-ffff-ffff-
	assert.Equal(t, "ffffffff-ffff-ffff-", after.UUID[:19])
	// UUID ends with LE binary representation of record ID
	idAsBinary := make([]byte, 8)
	binary.LittleEndian.PutUint64(idAsBinary, uint64(recordID))
	idAsHexadecimal := fmt.Sprintf("%.2x%.2x-%.2x%.2x%.2x%.2x%.2x%.2x", idAsBinary[0],
		idAsBinary[1], idAsBinary[2], idAsBinary[3], idAsBinary[4], idAsBinary[5],
		idAsBinary[6], idAsBinary[7])
	assert.Equal(t, idAsHexadecimal, after.UUID[19:])
}
