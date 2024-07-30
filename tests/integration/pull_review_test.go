// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/test"
	issue_service "code.gitea.io/gitea/services/issue"
	repo_service "code.gitea.io/gitea/services/repository"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullView_ReviewerMissed(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	req := NewRequest(t, "GET", "/pulls")
	resp := session.MakeRequest(t, req, http.StatusOK)
	assert.True(t, test.IsNormalPageCompleted(resp.Body.String()))

	req = NewRequest(t, "GET", "/user2/repo1/pulls/3")
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.True(t, test.IsNormalPageCompleted(resp.Body.String()))

	// if some reviews are missing, the page shouldn't fail
	reviews, err := issues_model.FindReviews(db.DefaultContext, issues_model.FindReviewOptions{
		IssueID: 2,
	})
	require.NoError(t, err)
	for _, r := range reviews {
		require.NoError(t, issues_model.DeleteReview(db.DefaultContext, r))
	}
	req = NewRequest(t, "GET", "/user2/repo1/pulls/2")
	resp = session.MakeRequest(t, req, http.StatusOK)
	assert.True(t, test.IsNormalPageCompleted(resp.Body.String()))
}

func loadComment(t *testing.T, commentID string) *issues_model.Comment {
	t.Helper()
	id, err := strconv.ParseInt(commentID, 10, 64)
	require.NoError(t, err)
	return unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: id})
}

func TestPullView_ResolveInvalidatedReviewComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	req := NewRequest(t, "GET", "/user2/repo1/pulls/3/files")
	session.MakeRequest(t, req, http.StatusOK)

	t.Run("single outdated review (line 1)", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequest(t, "GET", "/user2/repo1/pulls/3/files/reviews/new_comment")
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)
		req = NewRequestWithValues(t, "POST", "/user2/repo1/pulls/3/files/reviews/comments", map[string]string{
			"_csrf":            doc.GetInputValueByName("_csrf"),
			"origin":           doc.GetInputValueByName("origin"),
			"latest_commit_id": doc.GetInputValueByName("latest_commit_id"),
			"side":             "proposed",
			"line":             "1",
			"path":             "iso-8859-1.txt",
			"diff_start_cid":   doc.GetInputValueByName("diff_start_cid"),
			"diff_end_cid":     doc.GetInputValueByName("diff_end_cid"),
			"diff_base_cid":    doc.GetInputValueByName("diff_base_cid"),
			"content":          "nitpicking comment",
			"pending_review":   "",
		})
		session.MakeRequest(t, req, http.StatusOK)

		req = NewRequestWithValues(t, "POST", "/user2/repo1/pulls/3/files/reviews/submit", map[string]string{
			"_csrf":     doc.GetInputValueByName("_csrf"),
			"commit_id": doc.GetInputValueByName("latest_commit_id"),
			"content":   "looks good",
			"type":      "comment",
		})
		session.MakeRequest(t, req, http.StatusOK)

		// retrieve comment_id by reloading the comment page
		req = NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp = session.MakeRequest(t, req, http.StatusOK)
		doc = NewHTMLParser(t, resp.Body)
		commentID, ok := doc.Find(`[data-action="Resolve"]`).Attr("data-comment-id")
		assert.True(t, ok)

		// adjust the database to mark the comment as invalidated
		// (to invalidate it properly, one should push a commit which should trigger this logic,
		// in the meantime, use this quick-and-dirty trick)
		comment := loadComment(t, commentID)
		require.NoError(t, issues_model.UpdateCommentInvalidate(context.Background(), &issues_model.Comment{
			ID:          comment.ID,
			Invalidated: true,
		}))

		req = NewRequestWithValues(t, "POST", "/user2/repo1/issues/resolve_conversation", map[string]string{
			"_csrf":      doc.GetInputValueByName("_csrf"),
			"origin":     "timeline",
			"action":     "Resolve",
			"comment_id": commentID,
		})
		resp = session.MakeRequest(t, req, http.StatusOK)

		// even on template error, the page returns HTTP 200
		// count the comments to ensure success.
		doc = NewHTMLParser(t, resp.Body)
		assert.Len(t, doc.Find(`.comments > .comment`).Nodes, 1)
	})

	t.Run("outdated and newer review (line 2)", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequest(t, "GET", "/user2/repo1/pulls/3/files/reviews/new_comment")
		resp := session.MakeRequest(t, req, http.StatusOK)
		doc := NewHTMLParser(t, resp.Body)

		var firstReviewID int64
		{
			// first (outdated) review
			req = NewRequestWithValues(t, "POST", "/user2/repo1/pulls/3/files/reviews/comments", map[string]string{
				"_csrf":            doc.GetInputValueByName("_csrf"),
				"origin":           doc.GetInputValueByName("origin"),
				"latest_commit_id": doc.GetInputValueByName("latest_commit_id"),
				"side":             "proposed",
				"line":             "2",
				"path":             "iso-8859-1.txt",
				"diff_start_cid":   doc.GetInputValueByName("diff_start_cid"),
				"diff_end_cid":     doc.GetInputValueByName("diff_end_cid"),
				"diff_base_cid":    doc.GetInputValueByName("diff_base_cid"),
				"content":          "nitpicking comment",
				"pending_review":   "",
			})
			session.MakeRequest(t, req, http.StatusOK)

			req = NewRequestWithValues(t, "POST", "/user2/repo1/pulls/3/files/reviews/submit", map[string]string{
				"_csrf":     doc.GetInputValueByName("_csrf"),
				"commit_id": doc.GetInputValueByName("latest_commit_id"),
				"content":   "looks good",
				"type":      "comment",
			})
			session.MakeRequest(t, req, http.StatusOK)

			// retrieve comment_id by reloading the comment page
			req = NewRequest(t, "GET", "/user2/repo1/pulls/3")
			resp = session.MakeRequest(t, req, http.StatusOK)
			doc = NewHTMLParser(t, resp.Body)
			commentID, ok := doc.Find(`[data-action="Resolve"]`).Attr("data-comment-id")
			assert.True(t, ok)

			// adjust the database to mark the comment as invalidated
			// (to invalidate it properly, one should push a commit which should trigger this logic,
			// in the meantime, use this quick-and-dirty trick)
			comment := loadComment(t, commentID)
			require.NoError(t, issues_model.UpdateCommentInvalidate(context.Background(), &issues_model.Comment{
				ID:          comment.ID,
				Invalidated: true,
			}))
			firstReviewID = comment.ReviewID
			assert.NotZero(t, firstReviewID)
		}

		// ID of the first comment for the second (up-to-date) review
		var commentID string

		{
			// second (up-to-date) review on the same line
			// make a second review
			req = NewRequestWithValues(t, "POST", "/user2/repo1/pulls/3/files/reviews/comments", map[string]string{
				"_csrf":            doc.GetInputValueByName("_csrf"),
				"origin":           doc.GetInputValueByName("origin"),
				"latest_commit_id": doc.GetInputValueByName("latest_commit_id"),
				"side":             "proposed",
				"line":             "2",
				"path":             "iso-8859-1.txt",
				"diff_start_cid":   doc.GetInputValueByName("diff_start_cid"),
				"diff_end_cid":     doc.GetInputValueByName("diff_end_cid"),
				"diff_base_cid":    doc.GetInputValueByName("diff_base_cid"),
				"content":          "nitpicking comment",
				"pending_review":   "",
			})
			session.MakeRequest(t, req, http.StatusOK)

			req = NewRequestWithValues(t, "POST", "/user2/repo1/pulls/3/files/reviews/submit", map[string]string{
				"_csrf":     doc.GetInputValueByName("_csrf"),
				"commit_id": doc.GetInputValueByName("latest_commit_id"),
				"content":   "looks better",
				"type":      "comment",
			})
			session.MakeRequest(t, req, http.StatusOK)

			// retrieve comment_id by reloading the comment page
			req = NewRequest(t, "GET", "/user2/repo1/pulls/3")
			resp = session.MakeRequest(t, req, http.StatusOK)
			doc = NewHTMLParser(t, resp.Body)

			commentIDs := doc.Find(`[data-action="Resolve"]`).Map(func(i int, elt *goquery.Selection) string {
				v, _ := elt.Attr("data-comment-id")
				return v
			})
			assert.Len(t, commentIDs, 2) // 1 for the outdated review, 1 for the current review

			// check that the first comment is for the previous review
			comment := loadComment(t, commentIDs[0])
			assert.Equal(t, comment.ReviewID, firstReviewID)

			// check that the second comment is for a different review
			comment = loadComment(t, commentIDs[1])
			assert.NotZero(t, comment.ReviewID)
			assert.NotEqual(t, comment.ReviewID, firstReviewID)

			commentID = commentIDs[1] // save commentID for later
		}

		req = NewRequestWithValues(t, "POST", "/user2/repo1/issues/resolve_conversation", map[string]string{
			"_csrf":      doc.GetInputValueByName("_csrf"),
			"origin":     "timeline",
			"action":     "Resolve",
			"comment_id": commentID,
		})
		resp = session.MakeRequest(t, req, http.StatusOK)

		// even on template error, the page returns HTTP 200
		// count the comments to ensure success.
		doc = NewHTMLParser(t, resp.Body)
		comments := doc.Find(`.comments > .comment`)
		assert.Len(t, comments.Nodes, 1) // the outdated comment belongs to another review and should not be shown
	})

	t.Run("Files Changed tab", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		for _, c := range []struct {
			style, outdated string
			expectedCount   int
		}{
			{"unified", "true", 3},  // 1 comment on line 1 + 2 comments on line 3
			{"unified", "false", 1}, // 1 comment on line 3 is not outdated
			{"split", "true", 3},    // 1 comment on line 1 + 2 comments on line 3
			{"split", "false", 1},   // 1 comment on line 3 is not outdated
		} {
			t.Run(c.style+"+"+c.outdated, func(t *testing.T) {
				req := NewRequest(t, "GET", "/user2/repo1/pulls/3/files?style="+c.style+"&show-outdated="+c.outdated)
				resp := session.MakeRequest(t, req, http.StatusOK)

				doc := NewHTMLParser(t, resp.Body)
				comments := doc.Find(`.comments > .comment`)
				assert.Len(t, comments.Nodes, c.expectedCount)
			})
		}
	})

	t.Run("Conversation tab", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		req := NewRequest(t, "GET", "/user2/repo1/pulls/3")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		comments := doc.Find(`.comments > .comment`)
		assert.Len(t, comments.Nodes, 3) // 1 comment on line 1 + 2 comments on line 3
	})
}

func TestPullView_CodeOwner(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// Create the repo.
		repo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, user2, user2, repo_service.CreateRepoOptions{
			Name:             "test_codeowner",
			Readme:           "Default",
			AutoInit:         true,
			ObjectFormatName: git.Sha1ObjectFormat.Name(),
			DefaultBranch:    "master",
		})
		require.NoError(t, err)

		// add CODEOWNERS to default branch
		_, err = files_service.ChangeRepoFiles(db.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
			OldBranch: repo.DefaultBranch,
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "CODEOWNERS",
					ContentReader: strings.NewReader("README.md @user5\n"),
				},
			},
		})
		require.NoError(t, err)

		t.Run("First Pull Request", func(t *testing.T) {
			// create a new branch to prepare for pull request
			_, err = files_service.ChangeRepoFiles(db.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
				NewBranch: "codeowner-basebranch",
				Files: []*files_service.ChangeRepoFile{
					{
						Operation:     "update",
						TreePath:      "README.md",
						ContentReader: strings.NewReader("# This is a new project\n"),
					},
				},
			})
			require.NoError(t, err)

			// Create a pull request.
			session := loginUser(t, "user2")
			testPullCreate(t, session, "user2", "test_codeowner", false, repo.DefaultBranch, "codeowner-basebranch", "Test Pull Request")

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadRepoID: repo.ID, HeadBranch: "codeowner-basebranch"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
			require.NoError(t, pr.LoadIssue(db.DefaultContext))

			err := issue_service.ChangeTitle(db.DefaultContext, pr.Issue, user2, "[WIP] Test Pull Request")
			require.NoError(t, err)
			prUpdated1 := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
			require.NoError(t, prUpdated1.LoadIssue(db.DefaultContext))
			assert.EqualValues(t, "[WIP] Test Pull Request", prUpdated1.Issue.Title)

			err = issue_service.ChangeTitle(db.DefaultContext, prUpdated1.Issue, user2, "Test Pull Request2")
			require.NoError(t, err)
			prUpdated2 := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID})
			require.NoError(t, prUpdated2.LoadIssue(db.DefaultContext))
			assert.EqualValues(t, "Test Pull Request2", prUpdated2.Issue.Title)
		})

		// change the default branch CODEOWNERS file to change README.md's codeowner
		_, err = files_service.ChangeRepoFiles(db.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation:     "update",
					TreePath:      "CODEOWNERS",
					ContentReader: strings.NewReader("README.md @user8\n"),
				},
			},
		})
		require.NoError(t, err)

		t.Run("Second Pull Request", func(t *testing.T) {
			// create a new branch to prepare for pull request
			_, err = files_service.ChangeRepoFiles(db.DefaultContext, repo, user2, &files_service.ChangeRepoFilesOptions{
				NewBranch: "codeowner-basebranch2",
				Files: []*files_service.ChangeRepoFile{
					{
						Operation:     "update",
						TreePath:      "README.md",
						ContentReader: strings.NewReader("# This is a new project2\n"),
					},
				},
			})
			require.NoError(t, err)

			// Create a pull request.
			session := loginUser(t, "user2")
			testPullCreate(t, session, "user2", "test_codeowner", false, repo.DefaultBranch, "codeowner-basebranch2", "Test Pull Request2")

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "codeowner-basebranch2"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 8})
		})

		t.Run("Forked Repo Pull Request", func(t *testing.T) {
			user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
			forkedRepo, err := repo_service.ForkRepository(db.DefaultContext, user2, user5, repo_service.ForkRepoOptions{
				BaseRepo: repo,
				Name:     "test_codeowner_fork",
			})
			require.NoError(t, err)

			// create a new branch to prepare for pull request
			_, err = files_service.ChangeRepoFiles(db.DefaultContext, forkedRepo, user5, &files_service.ChangeRepoFilesOptions{
				NewBranch: "codeowner-basebranch-forked",
				Files: []*files_service.ChangeRepoFile{
					{
						Operation:     "update",
						TreePath:      "README.md",
						ContentReader: strings.NewReader("# This is a new forked project\n"),
					},
				},
			})
			require.NoError(t, err)

			session := loginUser(t, "user5")
			testPullCreate(t, session, "user5", "test_codeowner_fork", false, forkedRepo.DefaultBranch, "codeowner-basebranch-forked", "Test Pull Request2")

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "codeowner-basebranch-forked"})
			unittest.AssertExistsIf(t, false, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 8})
		})
	})
}

func TestPullView_GivenApproveOrRejectReviewOnClosedPR(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		user1Session := loginUser(t, "user1")
		user2Session := loginUser(t, "user2")

		// Have user1 create a fork of repo1.
		testRepoFork(t, user1Session, "user2", "repo1", "user1", "repo1")

		t.Run("Submit approve/reject review on merged PR", func(t *testing.T) {
			// Create a merged PR (made by user1) in the upstream repo1.
			testEditFile(t, user1Session, "user1", "repo1", "master", "README.md", "Hello, World (Edited)\n")
			resp := testPullCreate(t, user1Session, "user1", "repo1", false, "master", "master", "This is a pull title")
			elem := strings.Split(test.RedirectURL(resp), "/")
			assert.EqualValues(t, "pulls", elem[3])
			testPullMerge(t, user1Session, elem[1], elem[2], elem[4], repo_model.MergeStyleMerge, false)

			// Grab the CSRF token.
			req := NewRequest(t, "GET", path.Join(elem[1], elem[2], "pulls", elem[4]))
			resp = user2Session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			// Submit an approve review on the PR.
			testSubmitReview(t, user2Session, htmlDoc.GetCSRF(), "user2", "repo1", elem[4], "", "approve", http.StatusUnprocessableEntity)

			// Submit a reject review on the PR.
			testSubmitReview(t, user2Session, htmlDoc.GetCSRF(), "user2", "repo1", elem[4], "", "reject", http.StatusUnprocessableEntity)
		})

		t.Run("Submit approve/reject review on closed PR", func(t *testing.T) {
			// Created a closed PR (made by user1) in the upstream repo1.
			testEditFileToNewBranch(t, user1Session, "user1", "repo1", "master", "a-test-branch", "README.md", "Hello, World (Edited...again)\n")
			resp := testPullCreate(t, user1Session, "user1", "repo1", false, "master", "a-test-branch", "This is a pull title")
			elem := strings.Split(test.RedirectURL(resp), "/")
			assert.EqualValues(t, "pulls", elem[3])
			testIssueClose(t, user1Session, elem[1], elem[2], elem[4])

			// Grab the CSRF token.
			req := NewRequest(t, "GET", path.Join(elem[1], elem[2], "pulls", elem[4]))
			resp = user2Session.MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)

			// Submit an approve review on the PR.
			testSubmitReview(t, user2Session, htmlDoc.GetCSRF(), "user2", "repo1", elem[4], "", "approve", http.StatusUnprocessableEntity)

			// Submit a reject review on the PR.
			testSubmitReview(t, user2Session, htmlDoc.GetCSRF(), "user2", "repo1", elem[4], "", "reject", http.StatusUnprocessableEntity)
		})
	})
}

func testSubmitReview(t *testing.T, session *TestSession, csrf, owner, repo, pullNumber, commitID, reviewType string, expectedSubmitStatus int) *httptest.ResponseRecorder {
	options := map[string]string{
		"_csrf":     csrf,
		"commit_id": commitID,
		"content":   "test",
		"type":      reviewType,
	}

	submitURL := path.Join(owner, repo, "pulls", pullNumber, "files", "reviews", "submit")
	req := NewRequestWithValues(t, "POST", submitURL, options)
	return session.MakeRequest(t, req, expectedSubmitStatus)
}

func testIssueClose(t *testing.T, session *TestSession, owner, repo, issueNumber string) *httptest.ResponseRecorder {
	req := NewRequest(t, "GET", path.Join(owner, repo, "pulls", issueNumber))
	resp := session.MakeRequest(t, req, http.StatusOK)

	htmlDoc := NewHTMLParser(t, resp.Body)
	closeURL := path.Join(owner, repo, "issues", issueNumber, "comments")

	options := map[string]string{
		"_csrf":  htmlDoc.GetCSRF(),
		"status": "close",
	}

	req = NewRequestWithValues(t, "POST", closeURL, options)
	return session.MakeRequest(t, req, http.StatusOK)
}
