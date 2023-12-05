// Copyright 2023 The Forgejo Authors
// SPDX-License-Identifier: MIT

package migrations

import (
	"testing"

	"code.gitea.io/gitea/modules/structs"

	"github.com/stretchr/testify/require"
)

func TestForgejoDownload(t *testing.T) {
	require.NotNil(t, getFactoryFromServiceType(structs.ForgejoService))
}
