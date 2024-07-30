// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func testRepoStarringOrWatching(t *testing.T, action, listURI string) {
	t.Helper()

	defer tests.PrepareTestEnv(t)()

	oppositeAction := "un" + action
	session := loginUser(t, "user5")

	// Star/Watch the repo as user5
	req := NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/action/%s", action), map[string]string{
		"_csrf": GetCSRF(t, session, "/user2/repo1"),
	})
	session.MakeRequest(t, req, http.StatusOK)

	// Load the repo home as user5
	req = NewRequest(t, "GET", "/user2/repo1")
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Verify that the star/watch button is now the opposite
	htmlDoc := NewHTMLParser(t, resp.Body)
	actionButton := htmlDoc.Find(fmt.Sprintf("form[action='/user2/repo1/action/%s']", oppositeAction))
	assert.Equal(t, 1, actionButton.Length())
	text := strings.ToLower(actionButton.Find("button span.text").Text())
	assert.Equal(t, oppositeAction, text)

	// Load stargazers/watchers as user5
	req = NewRequestf(t, "GET", "/user2/repo1/%s", listURI)
	resp = session.MakeRequest(t, req, http.StatusOK)

	// Verify that "user5" is among the stargazers/watchers
	htmlDoc = NewHTMLParser(t, resp.Body)
	htmlDoc.AssertElement(t, ".user-cards .list .item.ui.segment > a[href='/user5']", true)

	// Unstar/unwatch the repo as user5
	req = NewRequestWithValues(t, "POST", fmt.Sprintf("/user2/repo1/action/%s", oppositeAction), map[string]string{
		"_csrf": GetCSRF(t, session, "/user2/repo1"),
	})
	session.MakeRequest(t, req, http.StatusOK)

	// Load the repo home as user5
	req = NewRequest(t, "GET", "/user2/repo1")
	resp = session.MakeRequest(t, req, http.StatusOK)

	// Verify that the star/watch button is now back to its default
	htmlDoc = NewHTMLParser(t, resp.Body)
	actionButton = htmlDoc.Find(fmt.Sprintf("form[action='/user2/repo1/action/%s']", action))
	assert.Equal(t, 1, actionButton.Length())
	text = strings.ToLower(actionButton.Find("button span.text").Text())
	assert.Equal(t, action, text)

	// Load stargazers/watchers as user5
	req = NewRequestf(t, "GET", "/user2/repo1/%s", listURI)
	resp = session.MakeRequest(t, req, http.StatusOK)

	// Verify that "user5" is not among the stargazers/watchers
	htmlDoc = NewHTMLParser(t, resp.Body)
	htmlDoc.AssertElement(t, ".user-cards .list .item.ui.segment > a[href='/user5']", false)
}

func TestRepoStarUnstarUI(t *testing.T) {
	testRepoStarringOrWatching(t, "star", "stars")
}

func TestRepoWatchUnwatchUI(t *testing.T) {
	testRepoStarringOrWatching(t, "watch", "watchers")
}

func TestDisabledStars(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Repository.DisableStars, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	t.Run("repo star, unstar", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "POST", "/user2/repo1/action/star")
		MakeRequest(t, req, http.StatusNotFound)

		req = NewRequest(t, "POST", "/user2/repo1/action/unstar")
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("repo stargazers", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/user2/repo1/stars")
		MakeRequest(t, req, http.StatusNotFound)
	})
}
