// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/translation"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestSignup(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	setting.Service.EnableCaptcha = false

	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUser",
		"email":     "exampleUser@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	// should be able to view new user's page
	req = NewRequest(t, "GET", "/exampleUser")
	MakeRequest(t, req, http.StatusOK)
}

func TestSignupAsRestricted(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	setting.Service.EnableCaptcha = false
	setting.Service.DefaultUserIsRestricted = true

	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "restrictedUser",
		"email":     "restrictedUser@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	// should be able to view new user's page
	req = NewRequest(t, "GET", "/restrictedUser")
	MakeRequest(t, req, http.StatusOK)

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "restrictedUser"})
	assert.True(t, user2.IsRestricted)
}

func TestSignupEmail(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	setting.Service.EnableCaptcha = false

	tests := []struct {
		email      string
		wantStatus int
		wantMsg    string
	}{
		{"exampleUser@example.com\r\n", http.StatusOK, translation.NewLocale("en-US").TrString("form.email_invalid")},
		{"exampleUser@example.com\r", http.StatusOK, translation.NewLocale("en-US").TrString("form.email_invalid")},
		{"exampleUser@example.com\n", http.StatusOK, translation.NewLocale("en-US").TrString("form.email_invalid")},
		{"exampleUser@example.com", http.StatusSeeOther, ""},
	}

	for i, test := range tests {
		req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
			"user_name": fmt.Sprintf("exampleUser%d", i),
			"email":     test.email,
			"password":  "examplePassword!1",
			"retype":    "examplePassword!1",
		})
		resp := MakeRequest(t, req, test.wantStatus)
		if test.wantMsg != "" {
			htmlDoc := NewHTMLParser(t, resp.Body)
			assert.Equal(t,
				test.wantMsg,
				strings.TrimSpace(htmlDoc.doc.Find(".ui.message").Text()),
			)
		}
	}
}

func TestSignupEmailChangeForInactiveUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Disable the captcha & enable email confirmation for registrations
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()
	defer test.MockVariableValue(&setting.Service.RegisterEmailConfirm, true)()

	// Create user
	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUserX",
		"email":     "wrong-email@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusOK)

	session := loginUserWithPassword(t, "exampleUserX", "examplePassword!1")

	// Verify that the initial e-mail is the wrong one.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserX"})
	assert.Equal(t, "wrong-email@example.com", user.Email)

	// Change the email address
	req = NewRequestWithValues(t, "POST", "/user/activate", map[string]string{
		"email": "fine-email@example.com",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Verify that the email was updated
	user = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserX"})
	assert.Equal(t, "fine-email@example.com", user.Email)

	// Try to change the email again
	req = NewRequestWithValues(t, "POST", "/user/activate", map[string]string{
		"email": "wrong-again@example.com",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)
	// Verify that the email was NOT updated
	user = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserX"})
	assert.Equal(t, "fine-email@example.com", user.Email)
}

func TestSignupEmailChangeForActiveUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Disable the captcha & enable email confirmation for registrations
	defer test.MockVariableValue(&setting.Service.EnableCaptcha, false)()
	defer test.MockVariableValue(&setting.Service.RegisterEmailConfirm, false)()

	// Create user
	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUserY",
		"email":     "wrong-email-2@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	session := loginUserWithPassword(t, "exampleUserY", "examplePassword!1")

	// Verify that the initial e-mail is the wrong one.
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserY"})
	assert.Equal(t, "wrong-email-2@example.com", user.Email)

	// Changing the email for a validated address is not available
	req = NewRequestWithValues(t, "POST", "/user/activate", map[string]string{
		"email": "fine-email-2@example.com",
	})
	session.MakeRequest(t, req, http.StatusNotFound)

	// Verify that the email remained unchanged
	user = unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "exampleUserY"})
	assert.Equal(t, "wrong-email-2@example.com", user.Email)
}
