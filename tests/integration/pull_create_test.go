// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/graceful"
	"code.gitea.io/gitea/modules/test"
	repo_service "code.gitea.io/gitea/services/repository"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func testPullCreate(t *testing.T, session *TestSession, user, repo string, toSelf bool, targetBranch, sourceBranch, title string) *httptest.ResponseRecorder {
	req := NewRequest(t, "GET", path.Join(user, repo))
	resp := session.MakeRequest(t, req, http.StatusOK)

	// Click the PR button to create a pull
	htmlDoc := NewHTMLParser(t, resp.Body)
	link, exists := htmlDoc.doc.Find("#new-pull-request").Attr("href")
	assert.True(t, exists, "The template has changed")

	targetUser := strings.Split(link, "/")[1]
	if toSelf && targetUser != user {
		link = strings.Replace(link, targetUser, user, 1)
	}

	if targetBranch != "master" {
		link = strings.Replace(link, "master...", targetBranch+"...", 1)
	}
	if sourceBranch != "master" {
		if targetUser == user {
			link = strings.Replace(link, "...master", "..."+sourceBranch, 1)
		} else {
			link = strings.Replace(link, ":master", ":"+sourceBranch, 1)
		}
	}

	req = NewRequest(t, "GET", link)
	resp = session.MakeRequest(t, req, http.StatusOK)

	// Submit the form for creating the pull
	htmlDoc = NewHTMLParser(t, resp.Body)
	link, exists = htmlDoc.doc.Find("form.ui.form").Attr("action")
	assert.True(t, exists, "The template has changed")
	req = NewRequestWithValues(t, "POST", link, map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
		"title": title,
	})
	resp = session.MakeRequest(t, req, http.StatusOK)
	return resp
}

func TestPullCreate(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "This is a pull title")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/repo1/pulls/[0-9]*$", url)

		// check .diff can be accessed and matches performed change
		req := NewRequest(t, "GET", url+".diff")
		resp = session.MakeRequest(t, req, http.StatusOK)
		assert.Regexp(t, `\+Hello, World \(Edited\)`, resp.Body)
		assert.Regexp(t, "^diff", resp.Body)
		assert.NotRegexp(t, "diff.*diff", resp.Body) // not two diffs, just one

		// check .patch can be accessed and matches performed change
		req = NewRequest(t, "GET", url+".patch")
		resp = session.MakeRequest(t, req, http.StatusOK)
		assert.Regexp(t, `\+Hello, World \(Edited\)`, resp.Body)
		assert.Regexp(t, "diff", resp.Body)
		assert.Regexp(t, `Subject: \[PATCH\] Update README.md`, resp.Body)
		assert.NotRegexp(t, "diff.*diff", resp.Body) // not two diffs, just one
	})
}

func TestPullCreate_TitleEscape(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testEditFile(t, session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master", "<i>XSS PR</i>")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/repo1/pulls/[0-9]*$", url)

		// Edit title
		req := NewRequest(t, "GET", url)
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)
		editTestTitleURL, exists := htmlDoc.doc.Find("#save-edit-title").First().Attr("data-update-url")
		assert.True(t, exists, "The template has changed")

		req = NewRequestWithValues(t, "POST", editTestTitleURL, map[string]string{
			"_csrf": htmlDoc.GetCSRF(),
			"title": "<u>XSS PR</u>",
		})
		session.MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", url)
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc = NewHTMLParser(t, resp.Body)
		titleHTML, err := htmlDoc.doc.Find(".comment-list .timeline-item.event .text b").First().Html()
		assert.NoError(t, err)
		assert.Equal(t, "<strike>&lt;i&gt;XSS PR&lt;/i&gt;</strike>", titleHTML)
		titleHTML, err = htmlDoc.doc.Find(".comment-list .timeline-item.event .text b").Next().Html()
		assert.NoError(t, err)
		assert.Equal(t, "&lt;u&gt;XSS PR&lt;/u&gt;", titleHTML)
	})
}

func testUIDeleteBranch(t *testing.T, session *TestSession, ownerName, repoName, branchName string) {
	relURL := "/" + path.Join(ownerName, repoName, "branches")
	req := NewRequest(t, "GET", relURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", relURL+"/delete", map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
		"name":  branchName,
	})
	session.MakeRequest(t, req, http.StatusOK)
}

func testDeleteRepository(t *testing.T, session *TestSession, ownerName, repoName string) {
	relURL := "/" + path.Join(ownerName, repoName, "settings")
	req := NewRequest(t, "GET", relURL)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	req = NewRequestWithValues(t, "POST", relURL+"?action=delete", map[string]string{
		"_csrf":     htmlDoc.GetCSRF(),
		"repo_name": fmt.Sprintf("%s/%s", ownerName, repoName),
	})
	session.MakeRequest(t, req, http.StatusSeeOther)
}

func TestPullBranchDelete(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer tests.PrepareTestEnv(t)()

		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")
		testCreateBranch(t, session, "user1", "repo1", "branch/master", "master1", http.StatusSeeOther)
		testEditFile(t, session, "user1", "repo1", "master1", "README.md", "Hello, World (Edited)\n")
		resp := testPullCreate(t, session, "user1", "repo1", false, "master", "master1", "This is a pull title")

		// check the redirected URL
		url := test.RedirectURL(resp)
		assert.Regexp(t, "^/user2/repo1/pulls/[0-9]*$", url)
		req := NewRequest(t, "GET", url)
		session.MakeRequest(t, req, http.StatusOK)

		// delete head branch and confirm pull page is ok
		testUIDeleteBranch(t, session, "user1", "repo1", "master1")
		req = NewRequest(t, "GET", url)
		session.MakeRequest(t, req, http.StatusOK)

		// delete head repository and confirm pull page is ok
		testDeleteRepository(t, session, "user1", "repo1")
		req = NewRequest(t, "GET", url)
		session.MakeRequest(t, req, http.StatusOK)
	})
}

func TestRecentlyPushed(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user1")
		testRepoFork(t, session, "user2", "repo1", "user1", "repo1")

		testCreateBranch(t, session, "user1", "repo1", "branch/master", "recent-push", http.StatusSeeOther)
		testEditFile(t, session, "user1", "repo1", "recent-push", "README.md", "Hello recently!\n")

		testCreateBranch(t, session, "user2", "repo1", "branch/master", "recent-push-base", http.StatusSeeOther)
		testEditFile(t, session, "user2", "repo1", "recent-push-base", "README.md", "Hello, recently, from base!\n")

		baseRepo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "repo1")
		assert.NoError(t, err)
		repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user1", "repo1")
		assert.NoError(t, err)

		enablePRs := func(t *testing.T, repo *repo_model.Repository) {
			t.Helper()

			err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo,
				[]repo_model.RepoUnit{{
					RepoID: repo.ID,
					Type:   unit_model.TypePullRequests,
				}},
				nil)
			assert.NoError(t, err)
		}

		disablePRs := func(t *testing.T, repo *repo_model.Repository) {
			t.Helper()

			err := repo_service.UpdateRepositoryUnits(db.DefaultContext, repo, nil,
				[]unit_model.Type{unit_model.TypePullRequests})
			assert.NoError(t, err)
		}

		testBanner := func(t *testing.T) {
			t.Helper()

			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			message := strings.TrimSpace(htmlDoc.Find(".ui.message").Text())
			link, _ := htmlDoc.Find(".ui.message a").Attr("href")
			expectedMessage := "You pushed on branch recent-push"

			assert.Contains(t, message, expectedMessage)
			assert.Equal(t, "/user1/repo1/src/branch/recent-push", link)
		}

		// Test that there's a recently pushed branches banner, and it contains
		// a link to the branch.
		t.Run("recently-pushed-banner", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testBanner(t)
		})

		// Test that it is still there if the fork has PRs disabled, but the
		// base repo still has them enabled.
		t.Run("with-fork-prs-disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				enablePRs(t, repo)
			}()

			disablePRs(t, repo)
			testBanner(t)
		})

		// Test that it is still there if the fork has PRs enabled, but the base
		// repo does not.
		t.Run("with-base-prs-disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				enablePRs(t, baseRepo)
			}()

			disablePRs(t, baseRepo)
			testBanner(t)
		})

		// Test that the banner is not present if both the base and current
		// repo have PRs disabled.
		t.Run("with-prs-disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				enablePRs(t, baseRepo)
				enablePRs(t, repo)
			}()

			disablePRs(t, repo)
			disablePRs(t, baseRepo)

			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, ".ui.message", false)
		})

		// Test that visiting the base repo has the banner too, and includes
		// recent push notifications from both the fork, and the base repo.
		t.Run("on the base repo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Count recently pushed branches on the fork
			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, ".ui.message", true)

			// Count recently pushed branches on the base repo
			req = NewRequest(t, "GET", "/user2/repo1")
			resp = session.MakeRequest(t, req, http.StatusOK)
			htmlDoc = NewHTMLParser(t, resp.Body)
			messageCountOnBase := htmlDoc.Find(".ui.message").Length()

			// We have two messages on the base: one from the fork, one on the
			// base itself.
			assert.Equal(t, 2, messageCountOnBase)
		})

		// Test that the banner's links point to the right repos
		t.Run("link validity", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// We're testing against the origin repo, because that has both
			// local branches, and another from a fork, so we can test both in
			// one test!

			req := NewRequest(t, "GET", "/user2/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			messages := htmlDoc.Find(".ui.message")

			prButtons := messages.Find("a[role='button']")
			branchLinks := messages.Find("a[href*='/src/branch/']")

			// ** base repo tests **
			basePRLink, _ := prButtons.First().Attr("href")
			baseBranchLink, _ := branchLinks.First().Attr("href")
			baseBranchName := branchLinks.First().Text()

			// branch in the same repo does not have a `user/repo:` qualifier.
			assert.Equal(t, "recent-push-base", baseBranchName)
			// branch link points to the same repo
			assert.Equal(t, "/user2/repo1/src/branch/recent-push-base", baseBranchLink)
			// PR link compares against the correct rep, and unqualified branch name
			assert.Equal(t, "/user2/repo1/compare/master...recent-push-base", basePRLink)

			// ** forked repo tests **
			forkPRLink, _ := prButtons.Last().Attr("href")
			forkBranchLink, _ := branchLinks.Last().Attr("href")
			forkBranchName := branchLinks.Last().Text()

			// branch in the forked repo has a `user/repo:` qualifier.
			assert.Equal(t, "user1/repo1:recent-push", forkBranchName)
			// branch link points to the forked repo
			assert.Equal(t, "/user1/repo1/src/branch/recent-push", forkBranchLink)
			// PR link compares against the correct rep, and qualified branch name
			assert.Equal(t, "/user2/repo1/compare/master...user1/repo1:recent-push", forkPRLink)
		})

		t.Run("unrelated branches are not shown", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})

			// Create a new branch with no relation to the default branch.
			// 1. Create a new Tree object
			cmd := git.NewCommand(db.DefaultContext, "write-tree")
			treeID, _, gitErr := cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			assert.NoError(t, gitErr)
			treeID = strings.TrimSpace(treeID)
			// 2. Create a new (empty) commit
			cmd = git.NewCommand(db.DefaultContext, "commit-tree", "-m", "Initial orphan commit").AddDynamicArguments(treeID)
			commitID, _, gitErr := cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			assert.NoError(t, gitErr)
			commitID = strings.TrimSpace(commitID)
			// 3. Create a new ref pointing to the orphaned commit
			cmd = git.NewCommand(db.DefaultContext, "update-ref", "refs/heads/orphan1").AddDynamicArguments(commitID)
			_, _, gitErr = cmd.RunStdString(&git.RunOpts{Dir: repo.RepoPath()})
			assert.NoError(t, gitErr)
			// 4. Sync the git repo to the database
			syncErr := repo_service.AddAllRepoBranchesToSyncQueue(graceful.GetManager().ShutdownContext(), adminUser.ID)
			assert.NoError(t, syncErr)
			// 5. Add a fresh commit, so that FindRecentlyPushedBranches has
			// something to find.
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
			changeResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, owner,
				&files_service.ChangeRepoFilesOptions{
					Files: []*files_service.ChangeRepoFile{
						{
							Operation:     "create",
							TreePath:      "README.md",
							ContentReader: strings.NewReader("a readme file"),
						},
					},
					Message:   "Add README.md",
					OldBranch: "orphan1",
					NewBranch: "orphan1",
				})
			assert.NoError(t, err)
			assert.NotEmpty(t, changeResp)

			// Check that we only have 1 message on the main repo, the orphaned
			// one is not shown.
			req := NewRequest(t, "GET", "/user1/repo1")
			resp := session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			htmlDoc.AssertElement(t, ".ui.message", true)
			link, _ := htmlDoc.Find(".ui.message a[href*='/src/branch/']").Attr("href")
			assert.Equal(t, "/user1/repo1/src/branch/recent-push", link)
		})
	})
}
