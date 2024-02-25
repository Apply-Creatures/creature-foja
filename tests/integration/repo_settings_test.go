// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	gitea_context "code.gitea.io/gitea/modules/context"
	"code.gitea.io/gitea/modules/setting"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestRepoSettingsUnits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID, Name: "repo1"})
	session := loginUser(t, user.Name)

	req := NewRequest(t, "GET", fmt.Sprintf("%s/settings/units", repo.Link()))
	session.MakeRequest(t, req, http.StatusOK)
}

func TestRepoAddMoreUnits(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	session := loginUser(t, user.Name)

	// Make sure there are no disabled repos in the settings!
	setting.Repository.DisabledRepoUnits = []string{}
	unit_model.LoadUnitConfig()

	// Create a known-good repo, with all units enabled.
	repo, _, f := CreateDeclarativeRepo(t, user, "", []unit_model.Type{
		unit_model.TypeCode,
		unit_model.TypePullRequests,
		unit_model.TypeProjects,
		unit_model.TypePackages,
		unit_model.TypeActions,
		unit_model.TypeIssues,
		unit_model.TypeWiki,
	}, nil, nil)
	defer f()

	assertAddMore := func(t *testing.T, present bool) {
		t.Helper()

		req := NewRequest(t, "GET", repo.Link())
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		htmlDoc.AssertElement(t, fmt.Sprintf("a[href='%s/settings/units']", repo.Link()), present)
	}

	t.Run("no add more with all units enabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		assertAddMore(t, false)
	})

	t.Run("add more if units can be enabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer func() {
			repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, []repo_model.RepoUnit{{
				RepoID: repo.ID,
				Type:   unit_model.TypePackages,
			}}, nil)
		}()

		// Disable the Packages unit
		err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, nil, []unit_model.Type{unit_model.TypePackages})
		assert.NoError(t, err)

		assertAddMore(t, true)
	})

	t.Run("no add more if unit is globally disabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer func() {
			repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, []repo_model.RepoUnit{{
				RepoID: repo.ID,
				Type:   unit_model.TypePackages,
			}}, nil)
			setting.Repository.DisabledRepoUnits = []string{}
			unit_model.LoadUnitConfig()
		}()

		// Disable the Packages unit globally
		setting.Repository.DisabledRepoUnits = []string{"repo.packages"}
		unit_model.LoadUnitConfig()

		// Disable the Packages unit
		err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, nil, []unit_model.Type{unit_model.TypePackages})
		assert.NoError(t, err)

		// The "Add more" link appears no more
		assertAddMore(t, false)
	})

	t.Run("issues & ext tracker globally disabled", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		defer func() {
			repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, []repo_model.RepoUnit{{
				RepoID: repo.ID,
				Type:   unit_model.TypeIssues,
			}}, nil)
			setting.Repository.DisabledRepoUnits = []string{}
			unit_model.LoadUnitConfig()
		}()

		// Disable both Issues and ExternalTracker units globally
		setting.Repository.DisabledRepoUnits = []string{"repo.issues", "repo.ext_issues"}
		unit_model.LoadUnitConfig()

		// Disable the Issues unit
		err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, nil, []unit_model.Type{unit_model.TypeIssues})
		assert.NoError(t, err)

		// The "Add more" link appears no more
		assertAddMore(t, false)
	})
}

func TestProtectedBranch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: user.ID})
	session := loginUser(t, user.Name)

	t.Run("Add", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		link := fmt.Sprintf("/%s/settings/branches/edit", repo.FullName())

		req := NewRequestWithValues(t, "POST", link, map[string]string{
			"_csrf":       GetCSRF(t, session, link),
			"rule_name":   "master",
			"enable_push": "true",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)

		// Verify it was added.
		unittest.AssertExistsIf(t, true, &git_model.ProtectedBranch{RuleName: "master", RepoID: repo.ID})
	})

	t.Run("Add duplicate", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		link := fmt.Sprintf("/%s/settings/branches/edit", repo.FullName())

		req := NewRequestWithValues(t, "POST", link, map[string]string{
			"_csrf":           GetCSRF(t, session, link),
			"rule_name":       "master",
			"require_signed_": "true",
		})
		session.MakeRequest(t, req, http.StatusSeeOther)
		flashCookie := session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.EqualValues(t, "error%3DThere%2Bis%2Balready%2Ba%2Brule%2Bfor%2Bthis%2Bset%2Bof%2Bbranches", flashCookie.Value)

		// Verify it wasn't added.
		unittest.AssertCount(t, &git_model.ProtectedBranch{RuleName: "master", RepoID: repo.ID}, 1)
	})
}
