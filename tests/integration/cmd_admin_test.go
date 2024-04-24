// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"

	"github.com/stretchr/testify/assert"
)

func Test_Cmd_AdminUser(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		for _, testCase := range []struct {
			name               string
			options            []string
			mustChangePassword bool
		}{
			{
				name:               "default",
				options:            []string{},
				mustChangePassword: true,
			},
			{
				name:               "--must-change-password=false",
				options:            []string{"--must-change-password=false"},
				mustChangePassword: false,
			},
			{
				name:               "--must-change-password=true",
				options:            []string{"--must-change-password=true"},
				mustChangePassword: true,
			},
			{
				name:               "--must-change-password",
				options:            []string{"--must-change-password"},
				mustChangePassword: true,
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				name := "testuser"

				options := []string{"user", "create", "--username", name, "--password", "password", "--email", name + "@example.com"}
				options = append(options, testCase.options...)
				output, err := runMainApp("admin", options...)
				assert.NoError(t, err)
				assert.Contains(t, output, "has been successfully created")
				user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: name})
				assert.Equal(t, testCase.mustChangePassword, user.MustChangePassword)

				_, err = runMainApp("admin", "user", "delete", "--username", name)
				assert.NoError(t, err)
				unittest.AssertNotExistsBean(t, &user_model.User{Name: name})
			})
		}
	})
}

func Test_Cmd_AdminFirstUser(t *testing.T) {
	onGiteaRun(t, func(*testing.T, *url.URL) {
		for _, testCase := range []struct {
			name               string
			options            []string
			mustChangePassword bool
			isAdmin            bool
		}{
			{
				name:               "default",
				options:            []string{},
				mustChangePassword: false,
				isAdmin:            false,
			},
			{
				name:               "--must-change-password=false",
				options:            []string{"--must-change-password=false"},
				mustChangePassword: false,
				isAdmin:            false,
			},
			{
				name:               "--must-change-password=true",
				options:            []string{"--must-change-password=true"},
				mustChangePassword: true,
				isAdmin:            false,
			},
			{
				name:               "--must-change-password",
				options:            []string{"--must-change-password"},
				mustChangePassword: true,
				isAdmin:            false,
			},
			{
				name:               "--admin default",
				options:            []string{"--admin"},
				mustChangePassword: false,
				isAdmin:            true,
			},
			{
				name:               "--admin --must-change-password=false",
				options:            []string{"--admin", "--must-change-password=false"},
				mustChangePassword: false,
				isAdmin:            true,
			},
			{
				name:               "--admin --must-change-password=true",
				options:            []string{"--admin", "--must-change-password=true"},
				mustChangePassword: true,
				isAdmin:            true,
			},
			{
				name:               "--admin --must-change-password",
				options:            []string{"--admin", "--must-change-password"},
				mustChangePassword: true,
				isAdmin:            true,
			},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				db.GetEngine(db.DefaultContext).Exec("DELETE FROM `user`")
				db.GetEngine(db.DefaultContext).Exec("DELETE FROM `email_address`")
				assert.Equal(t, int64(0), user_model.CountUsers(db.DefaultContext, nil))
				name := "testuser"

				options := []string{"user", "create", "--username", name, "--password", "password", "--email", name + "@example.com"}
				options = append(options, testCase.options...)
				output, err := runMainApp("admin", options...)
				assert.NoError(t, err)
				assert.Contains(t, output, "has been successfully created")
				user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: name})
				assert.Equal(t, testCase.mustChangePassword, user.MustChangePassword)
				assert.Equal(t, testCase.isAdmin, user.IsAdmin)
			})
		}
	})
}
