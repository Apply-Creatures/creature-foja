// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPullSummaryCommits(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		testUser := "user2"
		testRepo := "repo1"
		branchOld := "master"
		branchNew := "new-branch"
		session := loginUser(t, testUser)

		// Create a branch with commit, open a PR and see if the summary is displayed correctly for 1 commit
		testEditFileToNewBranch(t, session, testUser, testRepo, branchOld, branchNew, "README.md", "test of pull summary")
		url := path.Join(testUser, testRepo, "compare", branchOld+"..."+branchNew)
		req := NewRequestWithValues(t, "POST", url,
			map[string]string{
				"_csrf": GetCSRF(t, session, url),
				"title": "1st pull request to test summary",
			},
		)
		session.MakeRequest(t, req, http.StatusOK)
		testPullSummaryCommits(t, session, testUser, testRepo, "6", "wants to merge 1 commit")

		// Merge the PR and see if the summary is displayed correctly for 1 commit
		testPullMerge(t, session, testUser, testRepo, "6", "merge", true)
		testPullSummaryCommits(t, session, testUser, testRepo, "6", "merged 1 commit")

		// Create a branch with 2 commits, open a PR and see if the summary is displayed correctly for 2 commits
		testEditFileToNewBranch(t, session, testUser, testRepo, branchOld, branchNew, "README.md", "test of pull summary (the 2nd)")
		testEditFile(t, session, testUser, testRepo, branchNew, "README.md", "test of pull summary (the 3rd)")
		req = NewRequestWithValues(t, "POST", url,
			map[string]string{
				"_csrf": GetCSRF(t, session, url),
				"title": "2nd pull request to test summary",
			},
		)
		session.MakeRequest(t, req, http.StatusOK)
		testPullSummaryCommits(t, session, testUser, testRepo, "7", "wants to merge 2 commits")

		// Merge the PR and see if the summary is displayed correctly for 2 commits
		testPullMerge(t, session, testUser, testRepo, "7", "merge", true)
		testPullSummaryCommits(t, session, testUser, testRepo, "7", "merged 2 commits")
	})
}

func testPullSummaryCommits(t *testing.T, session *TestSession, user, repo, pullNum, expectedSummary string) {
	t.Helper()
	req := NewRequest(t, "GET", path.Join(user, repo, "pulls", pullNum))
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	text := strings.TrimSpace(doc.doc.Find(".pull-desc").Text())
	assert.Contains(t, text, expectedSummary)
}
