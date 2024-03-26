// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	gitea_context "code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestRenameBranch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: repo.ID, Name: "master"})

	// get branch setting page
	session := loginUser(t, "user2")
	t.Run("Normal", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "master",
			"to":    "main",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		// check new branch link
		req = NewRequest(t, "GET", "/user2/repo1/src/branch/main/README.md")
		session.MakeRequest(t, req, http.StatusOK)

		// check old branch link
		req = NewRequest(t, "GET", "/user2/repo1/src/branch/master/README.md")
		resp := session.MakeRequest(t, req, http.StatusSeeOther)
		location := resp.Header().Get("Location")
		assert.Equal(t, "/user2/repo1/src/branch/main/README.md", location)

		// check db
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		assert.Equal(t, "main", repo1.DefaultBranch)
	})

	t.Run("Protected branch", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Add protected branch
		req := NewRequestWithValues(t, "POST", "/user2/repo1/settings/branches/edit", map[string]string{
			"_csrf":       GetCSRF(t, session, "/user2/repo1/settings/branches/edit"),
			"rule_name":   "*",
			"enable_push": "true",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		// Verify it was added.
		unittest.AssertExistsIf(t, true, &git_model.ProtectedBranch{RuleName: "*", RepoID: repo.ID})

		req = NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", map[string]string{
			"_csrf": GetCSRF(t, session, "/user2/repo1/settings/branches"),
			"from":  "main",
			"to":    "main2",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.EqualValues(t, "error%3DCannot%2Brename%2Bbranch%2Bmain2%2Bbecause%2Bit%2Bis%2Ba%2Bprotected%2Bbranch.", flashCookie.Value)

		// Verify it didn't change.
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		assert.Equal(t, "main", repo1.DefaultBranch)
	})
}
