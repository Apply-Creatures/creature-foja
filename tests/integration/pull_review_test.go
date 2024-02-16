// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"net/http"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
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

func loadComment(t *testing.T, commentID string) *issues.Comment {
	t.Helper()
	id, err := strconv.ParseInt(commentID, 10, 64)
	assert.NoError(t, err)
	return unittest.AssertExistsAndLoadBean(t, &issues.Comment{ID: id})
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
		assert.NoError(t, issues.UpdateCommentInvalidate(context.Background(), &issues.Comment{
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
			assert.NoError(t, issues.UpdateCommentInvalidate(context.Background(), &issues.Comment{
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
