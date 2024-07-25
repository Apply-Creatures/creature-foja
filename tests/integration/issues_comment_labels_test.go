// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

// TestIssuesCommentLabels is a test for user (role) labels in comment headers in PRs and issues.
func TestIssuesCommentLabels(t *testing.T) {
	user := "user2"
	repo := "repo1"

	ownerTooltip := "This user is the owner of this repository."
	authorTooltipPR := "This user is the author of this pull request."
	authorTooltipIssue := "This user is the author of this issue."
	contributorTooltip := "This user has previously committed in this repository."
	newContributorTooltip := "This is the first contribution of this user to the repository."

	// Test pulls
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		sessionUser1 := loginUser(t, "user1")
		sessionUser2 := loginUser(t, "user2")
		sessionUser11 := loginUser(t, "user11")

		// Open a new PR as user2
		testEditFileToNewBranch(t, sessionUser2, user, repo, "master", "comment-labels", "README.md", "test of comment labels\naline")
		sessionUser2.MakeRequest(t, NewRequestWithValues(t, "POST", path.Join(user, repo, "compare", "master...comment-labels"),
			map[string]string{
				"_csrf": GetCSRF(t, sessionUser2, path.Join(user, repo, "compare", "master...comment-labels")),
				"title": "Pull used for testing commit labels",
			},
		), http.StatusOK)

		// Pull number, expected to be 6
		testID := "6"

		// Add a few comments
		// (first: Owner)
		testEasyLeavePRReviewComment(t, sessionUser2, user, repo, testID, "README.md", "1", "New review comment from user2 on this line", "")

		// Have to fetch reply ID for reviews
		response := sessionUser2.MakeRequest(t, NewRequest(t, "GET", path.Join(user, repo, "pulls", testID)), http.StatusOK)
		page := NewHTMLParser(t, response.Body)
		replyID, _ := page.Find(".comment-form input[name='reply']").Attr("value")

		testEasyLeavePRReviewComment(t, sessionUser2, user, repo, testID, "README.md", "1", "Another review comment from user2 on this line", replyID)
		testEasyLeavePRComment(t, sessionUser2, user, repo, testID, "New comment from user2 on this PR")   // Author, Owner
		testEasyLeavePRComment(t, sessionUser1, user, repo, testID, "New comment from user1 on this PR")   // Contributor
		testEasyLeavePRComment(t, sessionUser11, user, repo, testID, "New comment from user11 on this PR") // First-time contributor

		// Fetch the PR page
		response = sessionUser2.MakeRequest(t, NewRequest(t, "GET", path.Join(user, repo, "pulls", testID)), http.StatusOK)
		page = NewHTMLParser(t, response.Body)
		commentHeads := page.Find(".timeline .comment .comment-header .comment-header-right")
		assert.EqualValues(t, 6, commentHeads.Length())

		// Test the first comment and it's label "Owner"
		labels := commentHeads.Eq(0).Find(".role-label")
		assert.EqualValues(t, 1, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Owner", ownerTooltip)

		// Test the second (review) comment and it's labels "Author" and "Owner"
		labels = commentHeads.Eq(1).Find(".role-label")
		assert.EqualValues(t, 2, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Author", authorTooltipPR)
		testIssueCommentUserLabel(t, labels.Eq(1), "Owner", ownerTooltip)

		// Test the third (review) comment and it's labels "Author" and "Owner"
		labels = commentHeads.Eq(2).Find(".role-label")
		assert.EqualValues(t, 2, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Author", authorTooltipPR)
		testIssueCommentUserLabel(t, labels.Eq(1), "Owner", ownerTooltip)

		// Test the fourth comment and it's labels "Author" and "Owner"
		labels = commentHeads.Eq(3).Find(".role-label")
		assert.EqualValues(t, 2, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Author", authorTooltipPR)
		testIssueCommentUserLabel(t, labels.Eq(1), "Owner", ownerTooltip)

		// Test the fivth comment and it's label "Contributor"
		labels = commentHeads.Eq(4).Find(".role-label")
		assert.EqualValues(t, 1, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Contributor", contributorTooltip)

		// Test the sixth comment and it's label "First-time contributor"
		labels = commentHeads.Eq(5).Find(".role-label")
		assert.EqualValues(t, 1, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "First-time contributor", newContributorTooltip)
	})

	// Test issues
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		sessionUser1 := loginUser(t, "user1")
		sessionUser2 := loginUser(t, "user2")
		sessionUser5 := loginUser(t, "user5")

		// Open a new issue in the same repo
		sessionUser2.MakeRequest(t, NewRequestWithValues(t, "POST", path.Join(user, repo, "issues/new"),
			map[string]string{
				"_csrf": GetCSRF(t, sessionUser2, path.Join(user, repo)),
				"title": "Issue used for testing commit labels",
			},
		), http.StatusOK)

		// Issue number, expected to be 6
		testID := "6"
		// Add a few comments
		// (first: Owner)
		testEasyLeaveIssueComment(t, sessionUser2, user, repo, testID, "New comment from user2 on this issue") // Author, Owner
		testEasyLeaveIssueComment(t, sessionUser1, user, repo, testID, "New comment from user1 on this issue") // Contributor
		testEasyLeaveIssueComment(t, sessionUser5, user, repo, testID, "New comment from user5 on this issue") // no labels

		// Fetch the issue page
		response := sessionUser2.MakeRequest(t, NewRequest(t, "GET", path.Join(user, repo, "issues", testID)), http.StatusOK)
		page := NewHTMLParser(t, response.Body)
		commentHeads := page.Find(".timeline .comment .comment-header .comment-header-right")
		assert.EqualValues(t, 4, commentHeads.Length())

		// Test the first comment and it's label "Owner"
		labels := commentHeads.Eq(0).Find(".role-label")
		assert.EqualValues(t, 1, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Owner", ownerTooltip)

		// Test the second comment and it's labels "Author" and "Owner"
		labels = commentHeads.Eq(1).Find(".role-label")
		assert.EqualValues(t, 2, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Author", authorTooltipIssue)
		testIssueCommentUserLabel(t, labels.Eq(1), "Owner", ownerTooltip)

		// Test the third comment and it's label "Contributor"
		labels = commentHeads.Eq(2).Find(".role-label")
		assert.EqualValues(t, 1, labels.Length())
		testIssueCommentUserLabel(t, labels.Eq(0), "Contributor", contributorTooltip)

		// Test the fifth comment and it's lack of labels
		labels = commentHeads.Eq(3).Find(".role-label")
		assert.EqualValues(t, 0, labels.Length())
	})
}

// testIssueCommentUserLabel is used to verify properties of a user label from a comment
func testIssueCommentUserLabel(t *testing.T, label *goquery.Selection, expectedTitle, expectedTooltip string) {
	t.Helper()
	title := label.Text()
	tooltip, exists := label.Attr("data-tooltip-content")
	assert.True(t, exists)
	assert.EqualValues(t, expectedTitle, strings.TrimSpace(title))
	assert.EqualValues(t, expectedTooltip, strings.TrimSpace(tooltip))
}

// testEasyLeaveIssueComment is used to create a comment on an issue with minimum code and parameters
func testEasyLeaveIssueComment(t *testing.T, session *TestSession, user, repo, id, message string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", path.Join(user, repo, "issues", id, "comments"), map[string]string{
		"_csrf":   GetCSRF(t, session, path.Join(user, repo, "issues", id)),
		"content": message,
		"status":  "",
	}), 200)
}

// testEasyLeaveIssueComment is used to create a comment on a pull request with minimum code and parameters
// The POST request is supposed to use "issues" in the path. The CSRF is supposed to be generated for the PR page.
func testEasyLeavePRComment(t *testing.T, session *TestSession, user, repo, id, message string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", path.Join(user, repo, "issues", id, "comments"), map[string]string{
		"_csrf":   GetCSRF(t, session, path.Join(user, repo, "pulls", id)),
		"content": message,
		"status":  "",
	}), 200)
}

// testEasyLeavePRReviewComment is used to add review comments to specific lines of changed files in the diff of the PR.
func testEasyLeavePRReviewComment(t *testing.T, session *TestSession, user, repo, id, file, line, message, replyID string) {
	t.Helper()
	values := map[string]string{
		"_csrf":         GetCSRF(t, session, path.Join(user, repo, "pulls", id, "files")),
		"origin":        "diff",
		"side":          "proposed",
		"line":          line,
		"path":          file,
		"content":       message,
		"single_review": "true",
	}
	if len(replyID) > 0 {
		values["reply"] = replyID
	}
	session.MakeRequest(t, NewRequestWithValues(t, "POST", path.Join(user, repo, "pulls", id, "files/reviews/comments"), values), http.StatusOK)
}
