// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/translation"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestDangerZoneConfirmation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	mustInvalidRepoName := func(resp *httptest.ResponseRecorder) {
		t.Helper()

		htmlDoc := NewHTMLParser(t, resp.Body)
		assert.Contains(t,
			htmlDoc.doc.Find(".ui.negative.message").Text(),
			translation.NewLocale("en-US").Tr("form.enterred_invalid_repo_name"),
		)
	}

	t.Run("Transfer ownership", func(t *testing.T) {
		session := loginUser(t, "user2")

		t.Run("Fail", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user2/repo1/settings", map[string]string{
				"_csrf":          GetCSRF(t, session, "/user2/repo1/settings"),
				"action":         "transfer",
				"repo_name":      "repo1",
				"new_owner_name": "user1",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)
			mustInvalidRepoName(resp)
		})
		t.Run("Pass", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user2/repo1/settings", map[string]string{
				"_csrf":          GetCSRF(t, session, "/user2/repo1/settings"),
				"action":         "transfer",
				"repo_name":      "user2/repo1",
				"new_owner_name": "user1",
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.EqualValues(t, flashCookie.Value, "success%3DThis%2Brepository%2Bhas%2Bbeen%2Bmarked%2Bfor%2Btransfer%2Band%2Bawaits%2Bconfirmation%2Bfrom%2B%2522User%2BOne%2522")
		})
	})

	t.Run("Convert fork", func(t *testing.T) {
		session := loginUser(t, "user20")

		t.Run("Fail", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user20/big_test_public_fork_7/settings", map[string]string{
				"_csrf":     GetCSRF(t, session, "/user20/big_test_public_fork_7/settings"),
				"action":    "convert_fork",
				"repo_name": "big_test_public_fork_7",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)
			mustInvalidRepoName(resp)
		})
		t.Run("Pass", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user20/big_test_public_fork_7/settings", map[string]string{
				"_csrf":     GetCSRF(t, session, "/user20/big_test_public_fork_7/settings"),
				"action":    "convert_fork",
				"repo_name": "user20/big_test_public_fork_7",
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.EqualValues(t, flashCookie.Value, "success%3DThe%2Bfork%2Bhas%2Bbeen%2Bconverted%2Binto%2Ba%2Bregular%2Brepository.")
		})
	})

	t.Run("Delete wiki", func(t *testing.T) {
		session := loginUser(t, "user2")

		t.Run("Fail", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user2/repo1/settings", map[string]string{
				"_csrf":     GetCSRF(t, session, "/user2/repo1/settings"),
				"action":    "delete-wiki",
				"repo_name": "repo1",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)
			mustInvalidRepoName(resp)
		})
		t.Run("Pass", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user2/repo1/settings", map[string]string{
				"_csrf":     GetCSRF(t, session, "/user2/repo1/settings"),
				"action":    "delete-wiki",
				"repo_name": "user2/repo1",
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.EqualValues(t, flashCookie.Value, "success%3DThe%2Brepository%2Bwiki%2Bdata%2Bhas%2Bbeen%2Bdeleted.")
		})
	})

	t.Run("Delete", func(t *testing.T) {
		session := loginUser(t, "user2")

		t.Run("Fail", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user2/repo1/settings", map[string]string{
				"_csrf":     GetCSRF(t, session, "/user2/repo1/settings"),
				"action":    "delete",
				"repo_name": "repo1",
			})
			resp := session.MakeRequest(t, req, http.StatusOK)
			mustInvalidRepoName(resp)
		})
		t.Run("Pass", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithValues(t, "POST", "/user2/repo1/settings", map[string]string{
				"_csrf":     GetCSRF(t, session, "/user2/repo1/settings"),
				"action":    "delete",
				"repo_name": "user2/repo1",
			})
			session.MakeRequest(t, req, http.StatusSeeOther)

			flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
			assert.NotNil(t, flashCookie)
			assert.EqualValues(t, flashCookie.Value, "success%3DThe%2Brepository%2Bhas%2Bbeen%2Bdeleted.")
		})
	})
}
