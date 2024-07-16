// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: AGPL-3.0-only

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	issue_service "code.gitea.io/gitea/services/issue"
	pull_service "code.gitea.io/gitea/services/pull"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestPullRequestIcons(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		repo, _, f := CreateDeclarativeRepo(t, user, "pr-icons", []unit_model.Type{unit_model.TypeCode, unit_model.TypePullRequests}, nil, nil)
		defer f()

		session := loginUser(t, user.LoginName)

		// Individual PRs
		t.Run("Open", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createOpenPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "green", "octicon-git-pull-request")
		})

		t.Run("WIP (Open)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createOpenWipPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "grey", "octicon-git-pull-request-draft")
		})

		t.Run("Closed", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createClosedPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "red", "octicon-git-pull-request-closed")
		})

		t.Run("WIP (Closed)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createClosedWipPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "red", "octicon-git-pull-request-closed")
		})

		t.Run("Merged", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			pull := createMergedPullRequest(db.DefaultContext, t, user, repo)
			testPullRequestIcon(t, session, pull, "purple", "octicon-git-merge")
		})

		// List
		req := NewRequest(t, "GET", repo.HTMLURL()+"/pulls?state=all")
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		t.Run("List Open", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "open", "green", "octicon-git-pull-request")
		})

		t.Run("List WIP (Open)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "open-wip", "grey", "octicon-git-pull-request-draft")
		})

		t.Run("List Closed", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "closed", "red", "octicon-git-pull-request-closed")
		})

		t.Run("List Closed (WIP)", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "closed-wip", "red", "octicon-git-pull-request-closed")
		})

		t.Run("List Merged", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			testPullRequestListIcon(t, doc, "merged", "purple", "octicon-git-merge")
		})
	})
}

func testPullRequestIcon(t *testing.T, session *TestSession, pr *issues_model.PullRequest, expectedColor, expectedIcon string) {
	req := NewRequest(t, "GET", pr.Issue.HTMLURL())
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, fmt.Sprintf("div.issue-state-label.%s > svg.%s", expectedColor, expectedIcon), true)

	req = NewRequest(t, "GET", pr.BaseRepo.HTMLURL()+"/branches")
	resp = session.MakeRequest(t, req, http.StatusOK)
	doc = NewHTMLParser(t, resp.Body)
	doc.AssertElement(t, fmt.Sprintf(`a[href="/%s/pulls/%d"].%s > svg.%s`, pr.BaseRepo.FullName(), pr.Issue.Index, expectedColor, expectedIcon), true)
}

func testPullRequestListIcon(t *testing.T, doc *HTMLDoc, name, expectedColor, expectedIcon string) {
	sel := doc.doc.Find("div#issue-list > div.flex-item").
		FilterFunction(func(_ int, selection *goquery.Selection) bool {
			return selection.Find(fmt.Sprintf(`div.flex-item-icon > svg.%s.%s`, expectedColor, expectedIcon)).Length() == 1 &&
				strings.HasSuffix(selection.Find("a.issue-title").Text(), name)
		})

	assert.Equal(t, 1, sel.Length())
}

func createOpenPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "open")

	assert.False(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.False(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createOpenWipPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "open-wip")

	err := issue_service.ChangeTitle(ctx, pull.Issue, user, "WIP: "+pull.Issue.Title)
	assert.NoError(t, err)

	assert.False(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.True(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createClosedPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "closed")

	err := issue_service.ChangeStatus(ctx, pull.Issue, user, "", true)
	assert.NoError(t, err)

	assert.True(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.False(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createClosedWipPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "closed-wip")

	err := issue_service.ChangeTitle(ctx, pull.Issue, user, "WIP: "+pull.Issue.Title)
	assert.NoError(t, err)

	err = issue_service.ChangeStatus(ctx, pull.Issue, user, "", true)
	assert.NoError(t, err)

	assert.True(t, pull.Issue.IsClosed)
	assert.False(t, pull.HasMerged)
	assert.True(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createMergedPullRequest(ctx context.Context, t *testing.T, user *user_model.User, repo *repo_model.Repository) *issues_model.PullRequest {
	pull := createPullRequest(t, user, repo, "merged")

	gitRepo, err := git.OpenRepository(ctx, repo.RepoPath())
	defer gitRepo.Close()

	assert.NoError(t, err)

	err = pull_service.Merge(ctx, pull, user, gitRepo, repo_model.MergeStyleMerge, pull.HeadCommitID, "merge", false)
	assert.NoError(t, err)

	assert.False(t, pull.Issue.IsClosed)
	assert.True(t, pull.CanAutoMerge())
	assert.False(t, pull.IsWorkInProgress(ctx))

	return pull
}

func createPullRequest(t *testing.T, user *user_model.User, repo *repo_model.Repository, name string) *issues_model.PullRequest {
	branch := "branch-" + name
	title := "Testing " + name

	_, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user, &files_service.ChangeRepoFilesOptions{
		Files: []*files_service.ChangeRepoFile{
			{
				Operation:     "update",
				TreePath:      "README.md",
				ContentReader: strings.NewReader("Update README"),
			},
		},
		Message:   "Update README",
		OldBranch: "main",
		NewBranch: branch,
		Author: &files_service.IdentityOptions{
			Name:  user.Name,
			Email: user.Email,
		},
		Committer: &files_service.IdentityOptions{
			Name:  user.Name,
			Email: user.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    time.Now(),
			Committer: time.Now(),
		},
	})

	assert.NoError(t, err)

	pullIssue := &issues_model.Issue{
		RepoID:   repo.ID,
		Title:    title,
		PosterID: user.ID,
		Poster:   user,
		IsPull:   true,
	}

	pullRequest := &issues_model.PullRequest{
		HeadRepoID: repo.ID,
		BaseRepoID: repo.ID,
		HeadBranch: branch,
		BaseBranch: "main",
		HeadRepo:   repo,
		BaseRepo:   repo,
		Type:       issues_model.PullRequestGitea,
	}
	err = pull_service.NewPullRequest(git.DefaultContext, repo, pullIssue, nil, nil, pullRequest, nil)
	assert.NoError(t, err)

	return pullRequest
}
