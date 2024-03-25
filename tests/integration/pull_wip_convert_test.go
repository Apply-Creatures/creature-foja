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

func TestPullWIPConvertSidebar(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		testRepo := "repo1"
		branchOld := "master"
		branchNew := "wip"
		userOwner := "user2"
		userUnrelated := "user4"
		sessionOwner := loginUser(t, userOwner)         // Owner of the repo. Expected to see the offers.
		sessionUnrelated := loginUser(t, userUnrelated) // Unrelated user. Not expected to see the offers.

		// Create a branch with commit, open a PR and check who is seeing the Add WIP offering
		testEditFileToNewBranch(t, sessionOwner, userOwner, testRepo, branchOld, branchNew, "README.md", "test of wip offering")
		url := path.Join(userOwner, testRepo, "compare", branchOld+"..."+branchNew)
		req := NewRequestWithValues(t, "POST", url,
			map[string]string{
				"_csrf": GetCSRF(t, sessionOwner, url),
				"title": "pull used for testing wip offering",
			},
		)
		sessionOwner.MakeRequest(t, req, http.StatusOK)
		testPullWIPConvertSidebar(t, sessionOwner, userOwner, testRepo, "6", "Still in progress? Add WIP: prefix")
		testPullWIPConvertSidebar(t, sessionUnrelated, userOwner, testRepo, "6", "")

		// Add WIP: prefix and check who is seeing the Remove WIP offering
		req = NewRequestWithValues(t, "POST", path.Join(userOwner, testRepo, "pulls/6/title"),
			map[string]string{
				"_csrf": GetCSRF(t, sessionOwner, path.Join(userOwner, testRepo, "pulls/6")),
				"title": "WIP: pull used for testing wip offering",
			},
		)
		sessionOwner.MakeRequest(t, req, http.StatusOK)
		testPullWIPConvertSidebar(t, sessionOwner, userOwner, testRepo, "6", "Ready for review? Remove WIP: prefix")
		testPullWIPConvertSidebar(t, sessionUnrelated, userOwner, testRepo, "6", "")
	})
}

func testPullWIPConvertSidebar(t *testing.T, session *TestSession, user, repo, pullNum, expected string) {
	t.Helper()
	req := NewRequest(t, "GET", path.Join(user, repo, "pulls", pullNum))
	resp := session.MakeRequest(t, req, http.StatusOK)
	doc := NewHTMLParser(t, resp.Body)
	text := strings.TrimSpace(doc.doc.Find(".toggle-wip a").Text())
	assert.Equal(t, expected, text)
}
