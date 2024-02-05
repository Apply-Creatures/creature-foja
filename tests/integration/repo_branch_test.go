// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/modules/translation"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func testCreateBranch(t testing.TB, session *TestSession, user, repo, oldRefSubURL, newBranchName string, expectedStatus int) string {
	var csrf string
	if expectedStatus == http.StatusNotFound {
		csrf = GetCSRF(t, session, path.Join(user, repo, "src/branch/master"))
	} else {
		csrf = GetCSRF(t, session, path.Join(user, repo, "src", oldRefSubURL))
	}
	req := NewRequestWithValues(t, "POST", path.Join(user, repo, "branches/_new", oldRefSubURL), map[string]string{
		"_csrf":           csrf,
		"new_branch_name": newBranchName,
	})
	resp := session.MakeRequest(t, req, expectedStatus)
	if expectedStatus != http.StatusSeeOther {
		return ""
	}
	return test.RedirectURL(resp)
}

func TestCreateBranch(t *testing.T) {
	onGiteaRun(t, testCreateBranches)
}

func testCreateBranches(t *testing.T, giteaURL *url.URL) {
	tests := []struct {
		OldRefSubURL   string
		NewBranch      string
		CreateRelease  string
		FlashMessage   string
		ExpectedStatus int
		CheckBranch    bool
	}{
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      "feature/test1",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.create_success", "feature/test1"),
			CheckBranch:    true,
		},
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      "",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("form.NewBranchName") + translation.NewLocale("en-US").Tr("form.require_error"),
		},
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      "feature=test1",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.create_success", "feature=test1"),
			CheckBranch:    true,
		},
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      strings.Repeat("b", 101),
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("form.NewBranchName") + translation.NewLocale("en-US").Tr("form.max_size_error", "100"),
		},
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      "master",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.branch_already_exists", "master"),
		},
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      "master/test",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.branch_name_conflict", "master/test", "master"),
		},
		{
			OldRefSubURL:   "commit/acd1d892867872cb47f3993468605b8aa59aa2e0",
			NewBranch:      "feature/test2",
			ExpectedStatus: http.StatusNotFound,
		},
		{
			OldRefSubURL:   "commit/65f1bf27bc3bf70f64657658635e66094edbcb4d",
			NewBranch:      "feature/test3",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.create_success", "feature/test3"),
			CheckBranch:    true,
		},
		{
			OldRefSubURL:   "branch/master",
			NewBranch:      "v1.0.0",
			CreateRelease:  "v1.0.0",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.tag_collision", "v1.0.0"),
		},
		{
			OldRefSubURL:   "tag/v1.0.0",
			NewBranch:      "feature/test4",
			CreateRelease:  "v1.0.1",
			ExpectedStatus: http.StatusSeeOther,
			FlashMessage:   translation.NewLocale("en-US").Tr("repo.branch.create_success", "feature/test4"),
			CheckBranch:    true,
		},
	}

	session := loginUser(t, "user2")
	for _, test := range tests {
		if test.CheckBranch {
			unittest.AssertNotExistsBean(t, &git_model.Branch{RepoID: 1, Name: test.NewBranch})
		}
		if test.CreateRelease != "" {
			createNewRelease(t, session, "/user2/repo1", test.CreateRelease, test.CreateRelease, false, false)
		}
		redirectURL := testCreateBranch(t, session, "user2", "repo1", test.OldRefSubURL, test.NewBranch, test.ExpectedStatus)
		if test.ExpectedStatus == http.StatusSeeOther {
			req := NewRequest(t, "GET", redirectURL)
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			assert.Contains(t,
				strings.TrimSpace(htmlDoc.doc.Find(".ui.message").Text()),
				test.FlashMessage,
			)
		}
		if test.CheckBranch {
			unittest.AssertExistsAndLoadBean(t, &git_model.Branch{RepoID: 1, Name: test.NewBranch})
		}
	}
}

func TestCreateBranchInvalidCSRF(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user2")
	req := NewRequestWithValues(t, "POST", "user2/repo1/branches/_new/branch/master", map[string]string{
		"_csrf":           "fake_csrf",
		"new_branch_name": "test",
	})
	resp := session.MakeRequest(t, req, http.StatusSeeOther)
	loc := resp.Header().Get("Location")
	assert.Equal(t, setting.AppSubURL+"/", loc)
	resp = session.MakeRequest(t, NewRequest(t, "GET", loc), http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Equal(t,
		"Bad Request: invalid CSRF token",
		strings.TrimSpace(htmlDoc.doc.Find(".ui.message").Text()),
	)
}

func TestDatabaseMissingABranch(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, URL *url.URL) {
		adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
		session := loginUser(t, "user2")

		// Create two branches
		testCreateBranch(t, session, "user2", "repo1", "branch/master", "will-be-present", http.StatusSeeOther)
		testCreateBranch(t, session, "user2", "repo1", "branch/master", "will-be-missing", http.StatusSeeOther)

		// Run the repo branch sync, to ensure the db and git agree.
		err2 := repo_service.AddAllRepoBranchesToSyncQueue(graceful.GetManager().ShutdownContext(), adminUser.ID)
		assert.NoError(t, err2)

		// Delete one branch from git only, leaving it in the database
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		cmd := git.NewCommand(db.DefaultContext, "branch", "-D").AddDynamicArguments("will-be-missing")
		_, _, err := cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
		assert.NoError(t, err)

		// Verify that loading the repo's branches page works still, and that it
		// reports at least three branches (master, will-be-present, and
		// will-be-missing).
		req := NewRequest(t, "GET", "/user2/repo1/branches")
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		firstBranchCount, _ := strconv.Atoi(doc.Find(".repository-menu a[href*='/branches'] b").Text())
		assert.GreaterOrEqual(t, firstBranchCount, 3)

		// Run the repo branch sync again
		err2 = repo_service.AddAllRepoBranchesToSyncQueue(graceful.GetManager().ShutdownContext(), adminUser.ID)
		assert.NoError(t, err2)

		// Verify that loading the repo's branches page works still, and that it
		// reports one branch less than the first time.
		//
		// NOTE: This assumes that the branch counter on the web UI is out of
		// date before the sync. If that problem gets resolved, we'll have to
		// find another way to test that the syncing works.
		req = NewRequest(t, "GET", "/user2/repo1/branches")
		resp = session.MakeRequest(t, req, http.StatusOK)
		doc = NewHTMLParser(t, resp.Body)
		secondBranchCount, _ := strconv.Atoi(doc.Find(".repository-menu a[href*='/branches'] b").Text())
		assert.Equal(t, firstBranchCount-1, secondBranchCount)
	})
}
