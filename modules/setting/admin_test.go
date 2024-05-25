// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"code.gitea.io/gitea/modules/container"

	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
	loadAdminFrom(cfg)

	assert.EqualValues(t, true, Admin.DisableRegularOrgCreation)
	assert.EqualValues(t, "z", Admin.DefaultEmailNotification)
	assert.EqualValues(t, true, Admin.SendNotificationEmailOnNewUser)
	assert.EqualValues(t, container.SetOf("a", "b"), Admin.UserDisabledFeatures)
	assert.EqualValues(t, container.SetOf("x", "y"), Admin.ExternalUserDisableFeatures)
}
