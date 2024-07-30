// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"code.gitea.io/gitea/modules/container"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_loadAdminFrom(t *testing.T) {
	iniStr := `
	[admin]
	DISABLE_REGULAR_ORG_CREATION = true
  DEFAULT_EMAIL_NOTIFICATIONS = z
  SEND_NOTIFICATION_EMAIL_ON_NEW_USER = true
  USER_DISABLED_FEATURES = a,b
  EXTERNAL_USER_DISABLE_FEATURES = x,y
	`
	cfg, err := NewConfigProviderFromData(iniStr)
	require.NoError(t, err)
	loadAdminFrom(cfg)

	assert.True(t, Admin.DisableRegularOrgCreation)
	assert.EqualValues(t, "z", Admin.DefaultEmailNotification)
	assert.True(t, Admin.SendNotificationEmailOnNewUser)
	assert.EqualValues(t, container.SetOf("a", "b"), Admin.UserDisabledFeatures)
	assert.EqualValues(t, container.SetOf("x", "y"), Admin.ExternalUserDisableFeatures)
}
