// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingShowUserEmailExplore(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	showUserEmail := setting.UI.ShowUserEmail
	setting.UI.ShowUserEmail = true

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/explore/users?sort=alphabetically")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Contains(t,
		htmlDoc.doc.Find(".explore.users").Text(),
		"user34@example.com",
	)

	setting.UI.ShowUserEmail = false

	req = NewRequest(t, "GET", "/explore/users?sort=alphabetically")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.NotContains(t,
		htmlDoc.doc.Find(".explore.users").Text(),
		"user34@example.com",
	)

	setting.UI.ShowUserEmail = showUserEmail
}

func TestSettingShowUserEmailProfile(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	showUserEmail := setting.UI.ShowUserEmail

	// user1: keep_email_private = false, user2: keep_email_private = true

	setting.UI.ShowUserEmail = true

	// user1 can see own visible email
	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", "/user1")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Contains(t, htmlDoc.doc.Find(".user.profile").Text(), "user1@example.com")

	// user1 can not see user2's hidden email
	req = NewRequest(t, "GET", "/user2")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	// Should only contain if the user visits their own profile page
	assert.NotContains(t, htmlDoc.doc.Find(".user.profile").Text(), "user2@example.com")

	// user2 can see user1's visible email
	session = loginUser(t, "user2")
	req = NewRequest(t, "GET", "/user1")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.Contains(t, htmlDoc.doc.Find(".user.profile").Text(), "user1@example.com")

	// user2 cannot see own hidden email
	session = loginUser(t, "user2")
	req = NewRequest(t, "GET", "/user2")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.NotContains(t, htmlDoc.doc.Find(".user.profile").Text(), "user2@example.com")

	setting.UI.ShowUserEmail = false

	// user1 cannot see own (now hidden) email
	session = loginUser(t, "user1")
	req = NewRequest(t, "GET", "/user1")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.NotContains(t, htmlDoc.doc.Find(".user.profile").Text(), "user1@example.com")

	setting.UI.ShowUserEmail = showUserEmail
}

func TestSettingLandingPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	landingPage := setting.LandingPageURL

	setting.LandingPageURL = setting.LandingPageHome
	req := NewRequest(t, "GET", "/")
	MakeRequest(t, req, http.StatusOK)

	setting.LandingPageURL = setting.LandingPageExplore
	req = NewRequest(t, "GET", "/")
	resp := MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/explore", resp.Header().Get("Location"))

	setting.LandingPageURL = setting.LandingPageOrganizations
	req = NewRequest(t, "GET", "/")
	resp = MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/explore/organizations", resp.Header().Get("Location"))

	setting.LandingPageURL = setting.LandingPageLogin
	req = NewRequest(t, "GET", "/")
	resp = MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/user/login", resp.Header().Get("Location"))

	setting.LandingPageURL = landingPage
}

func TestSettingSecurityAuthSource(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	active := addAuthSource(t, authSourcePayloadGitLabCustom("gitlab-active"))
	activeExternalLoginUser := &user_model.ExternalLoginUser{
		ExternalID:    "12345",
		UserID:        user.ID,
		LoginSourceID: active.ID,
	}
	err := user_model.LinkExternalToUser(db.DefaultContext, user, activeExternalLoginUser)
	require.NoError(t, err)

	inactive := addAuthSource(t, authSourcePayloadGitLabCustom("gitlab-inactive"))
	inactiveExternalLoginUser := &user_model.ExternalLoginUser{
		ExternalID:    "5678",
		UserID:        user.ID,
		LoginSourceID: inactive.ID,
	}
	err = user_model.LinkExternalToUser(db.DefaultContext, user, inactiveExternalLoginUser)
	require.NoError(t, err)

	// mark the authSource as inactive
	inactive.IsActive = false
	err = auth_model.UpdateSource(db.DefaultContext, inactive)
	require.NoError(t, err)

	session := loginUser(t, "user1")
	req := NewRequest(t, "GET", "user/settings/security")
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.Contains(t, resp.Body.String(), `gitlab-active`)
	assert.Contains(t, resp.Body.String(), `gitlab-inactive`)
}
