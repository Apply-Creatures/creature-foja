// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func testRepoFork(t *testing.T, session *TestSession, ownerName, repoName, forkOwnerName, forkRepoName string) *httptest.ResponseRecorder {
	t.Helper()

	forkOwner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: forkOwnerName})

	// Step0: check the existence of the to-fork repo
	req := NewRequestf(t, "GET", "/%s/%s", forkOwnerName, forkRepoName)
	session.MakeRequest(t, req, http.StatusNotFound)

	// Step1: visit the /fork page
	forkURL := fmt.Sprintf("/%s/%s/fork", ownerName, repoName)
	req = NewRequest(t, "GET", forkURL)
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Step2: fill the form of the forking
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find(fmt.Sprintf("form.ui.form[action=\"%s\"]", forkURL)).Attr("action")
	assert.True(t, exists, "The template has changed")
	_, exists = htmlDoc.doc.Find(fmt.Sprintf(".owner.dropdown .item[data-value=\"%d\"]", forkOwner.ID)).Attr("data-value")
	assert.True(t, exists, fmt.Sprintf("Fork owner '%s' is not present in select box", forkOwnerName))
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf":     htmlDoc.GetCSRF(),
		"uid":       fmt.Sprintf("%d", forkOwner.ID),
		"repo_name": forkRepoName,
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	// Step3: check the existence of the forked repo
	req = NewRequestf(t, "GET", "/%s/%s", forkOwnerName, forkRepoName)
	resp = session.MakeRequest(t, req, http.StatusOK)

	return resp
}

func testRepoForkLegacyRedirect(t *testing.T, session *TestSession, ownerName, repoName string) {
	t.Helper()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: ownerName})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: owner.ID, Name: repoName})

	// Visit the /repo/fork/:id url
	req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID)
	resp := session.MakeRequest(t, req, http.StatusMovedPermanently)

	assert.Equal(t, repo.Link()+"/fork", resp.Header().Get("Location"))
}

func TestRepoFork(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user5"})
		session := loginUser(t, user5.Name)

		t.Run("by name", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user5.ID, Name: "repo1"})
				repo_service.DeleteRepository(db.DefaultContext, user5, repo, false)
			}()
			testRepoFork(t, session, "user2", "repo1", "user5", "repo1")
		})

		t.Run("legacy redirect", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			testRepoForkLegacyRedirect(t, session, "user2", "repo1")

			t.Run("private 404", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Make sure the repo we try to fork is private
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 31, IsPrivate: true})

				// user5 does not have access to user2/repo20
				req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID) // user2/repo20
				session.MakeRequest(t, req, http.StatusNotFound)
			})
			t.Run("authenticated private redirect", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Make sure the repo we try to fork is private
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 31, IsPrivate: true})

				// user1 has access to user2/repo20
				session := loginUser(t, "user1")
				req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID) // user2/repo20
				session.MakeRequest(t, req, http.StatusMovedPermanently)
			})
			t.Run("no code unit", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// Make sure the repo we try to fork is private.
				// We're also choosing user15/big_test_private_2, becase it has the Code unit disabled.
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 20, IsPrivate: true})

				// user1, even though an admin, can't fork a repo without a code unit.
				session := loginUser(t, "user1")
				req := NewRequestf(t, "GET", "/repo/fork/%d", repo.ID) // user15/big_test_private_2
				session.MakeRequest(t, req, http.StatusNotFound)
			})
		})
	})
}

func TestRepoForkToOrg(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")
		org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "org3"})

		t.Run("by name", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: org3.ID, Name: "repo1"})
				repo_service.DeleteRepository(db.DefaultContext, org3, repo, false)
			}()

			testRepoFork(t, session, "user2", "repo1", "org3", "repo1")

			// Check that no more forking is allowed as user2 owns repository
			//  and org3 organization that owner user2 is also now has forked this repository
			req := NewRequest(t, "GET", "/user2/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			_, exists := htmlDoc.doc.Find("a.ui.button[href^=\"/fork\"]").Attr("href")
			assert.False(t, exists, "Forking should not be allowed anymore")
		})

		t.Run("legacy redirect", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testRepoForkLegacyRedirect(t, session, "user2", "repo1")
		})
	})
}
