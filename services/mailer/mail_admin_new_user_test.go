// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mailer

import (
	"context"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/translation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getTestUsers(t *testing.T) []*user_model.User {
	t.Helper()
	admin := new(user_model.User)
	admin.Name = "testadmin"
	admin.IsAdmin = true
	admin.Language = "en_US"
	admin.Email = "admin@example.com"
	require.NoError(t, user_model.CreateUser(db.DefaultContext, admin))

	newUser := new(user_model.User)
	newUser.Name = "new_user"
	newUser.Language = "en_US"
	newUser.IsAdmin = false
	newUser.Email = "new_user@example.com"
	newUser.LastLoginUnix = 1693648327
	newUser.CreatedUnix = 1693648027
	require.NoError(t, user_model.CreateUser(db.DefaultContext, newUser))

	return []*user_model.User{admin, newUser}
}

func cleanUpUsers(ctx context.Context, users []*user_model.User) {
	for _, u := range users {
		db.DeleteByID[user_model.User](ctx, u.ID)
	}
}

func TestAdminNotificationMail_test(t *testing.T) {
	translation.InitLocales(context.Background())
	locale := translation.NewLocale("")
	key := "mail.admin.new_user.user_info"
	translatedKey := locale.Tr(key)
	require.NotEqualValues(t, key, translatedKey)

	mailService := setting.Mailer{
		From:     "test@example.com",
		Protocol: "dummy",
	}

	setting.MailService = &mailService

	// test with SEND_NOTIFICATION_EMAIL_ON_NEW_USER enabled
	setting.Admin.SendNotificationEmailOnNewUser = true

	ctx := context.Background()
	NewContext(ctx)

	users := getTestUsers(t)
	oldSendAsync := sa
	defer func() {
		sa = oldSendAsync
		cleanUpUsers(ctx, users)
	}()

	called := false
	sa = func(msgs ...*Message) {
		assert.Equal(t, len(msgs), 1, "Test provides only one admin user, so only one email must be sent")
		assert.Equal(t, msgs[0].To, users[0].Email, "checks if the recipient is the admin of the instance")
		manageUserURL := setting.AppURL + "admin/users/" + strconv.FormatInt(users[1].ID, 10)
		assert.Contains(t, msgs[0].Body, manageUserURL)
		assert.Contains(t, msgs[0].Body, users[1].HTMLURL())
		assert.Contains(t, msgs[0].Body, translatedKey, "the .Locale translates to nothing")
		assert.Contains(t, msgs[0].Body, users[1].Name, "user name of the newly created user")
		for _, untranslated := range []string{"mail.admin", "admin.users"} {
			assert.NotContains(t, msgs[0].Body, untranslated, "this is an untranslated placeholder prefix")
		}
		called = true
	}
	MailNewUser(ctx, users[1])
	assert.True(t, called)

	// test with SEND_NOTIFICATION_EMAIL_ON_NEW_USER disabled; emails shouldn't be sent
	setting.Admin.SendNotificationEmailOnNewUser = false
	sa = func(msgs ...*Message) {
		assert.Equal(t, 1, 0, "this shouldn't execute. MailNewUser must exit early since SEND_NOTIFICATION_EMAIL_ON_NEW_USER is disabled")
	}

	MailNewUser(ctx, users[1])
}
