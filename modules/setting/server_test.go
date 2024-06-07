// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestDisplayNameDefault(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "Beyond coding. We Forge.")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME}: {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo: Beyond coding. We Forge.", displayName)
}

func TestDisplayNameEmptySlogan(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME}: {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo", displayName)
}

func TestDisplayNameCustomFormat(t *testing.T) {
	defer test.MockVariableValue(&AppName, "Forgejo")()
	defer test.MockVariableValue(&AppSlogan, "Beyond coding. We Forge.")()
	defer test.MockVariableValue(&AppDisplayNameFormat, "{APP_NAME} - {APP_SLOGAN}")()
	displayName := generateDisplayName()
	assert.Equal(t, "Forgejo - Beyond coding. We Forge.", displayName)
}
