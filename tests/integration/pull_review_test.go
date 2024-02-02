// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestPullView_ReviewerMissed(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	req := NewRequest(t, "GET", "/pulls")
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", "/user2/repo1/pulls/3")
	session.MakeRequest(t, req, http.StatusOK)
}

func TestPullView_ResolveInvalidatedReviewComment(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	session := loginUser(t, "user1")

	req := NewRequest(t, "GET", "/user2/repo1/pulls/3/files")
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", "/user2/repo1/pulls/3/files/reviews/new_comment")
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
	id, err := strconv.ParseInt(commentID, 10, 64)
	assert.NoError(t, err)
	assert.NoError(t, issues.UpdateCommentInvalidate(context.Background(), &issues.Comment{
		ID:          id,
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
	// search the button to mark the comment as unresolved to ensure success.
	doc = NewHTMLParser(t, resp.Body)
	assert.Len(t, doc.Find(`[data-action="UnResolve"][data-comment-id="`+commentID+`"]`).Nodes, 1)
}
