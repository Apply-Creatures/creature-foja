// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/unittest"
	"code.gitea.io/gitea/modules/templates"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/contexttest"
	"code.gitea.io/gitea/services/pull"

	"github.com/stretchr/testify/assert"
)

func TestRenderConversation(t *testing.T) {
	unittest.PrepareTestEnv(t)

	pr, _ := issues_model.GetPullRequestByID(db.DefaultContext, 2)
	_ = pr.LoadIssue(db.DefaultContext)
	_ = pr.Issue.LoadPoster(db.DefaultContext)
	_ = pr.Issue.LoadRepo(db.DefaultContext)

	run := func(name string, cb func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder)) {
		t.Run(name, func(t *testing.T) {
			ctx, resp := contexttest.MockContext(t, "/")
			ctx.Render = templates.HTMLRenderer()
			contexttest.LoadUser(t, ctx, pr.Issue.PosterID)
			contexttest.LoadRepo(t, ctx, pr.BaseRepoID)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()
			cb(t, ctx, resp)
		})
	}

	var preparedComment *issues_model.Comment
	run("prepare", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		comment, err := pull.CreateCodeComment(ctx, pr.Issue.Poster, ctx.Repo.GitRepo, pr.Issue, 1, "content", "", false, 0, pr.HeadCommitID, nil)
		if !assert.NoError(t, err) {
			return
		}
		comment.Invalidated = true
		err = issues_model.UpdateCommentInvalidate(ctx, comment)
		if !assert.NoError(t, err) {
			return
		}
		preparedComment = comment
	})
	if !assert.NotNil(t, preparedComment) {
		return
	}
	run("diff with outdated", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "diff")
		assert.Contains(t, resp.Body.String(), `<div class="content comment-container"`)
	})
	run("diff without outdated", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = false
		renderConversation(ctx, preparedComment, "diff")
		// unlike gitea, Forgejo renders the conversation (with the "outdated" label)
		assert.Contains(t, resp.Body.String(), `repo.issues.review.outdated_description`)
	})
	run("timeline with outdated", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "timeline")
		assert.Contains(t, resp.Body.String(), `<div id="code-comments-`)
	})
	run("timeline is not affected by ShowOutdatedComments=false", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		ctx.Data["ShowOutdatedComments"] = false
		renderConversation(ctx, preparedComment, "timeline")
		assert.Contains(t, resp.Body.String(), `<div id="code-comments-`)
	})
	run("diff non-existing review", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		reviews, err := issues_model.FindReviews(db.DefaultContext, issues_model.FindReviewOptions{
			IssueID: 2,
		})
		assert.NoError(t, err)
		for _, r := range reviews {
			assert.NoError(t, issues_model.DeleteReview(db.DefaultContext, r))
		}
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "diff")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.NotContains(t, resp.Body.String(), `status-page-500`)
	})
	run("timeline non-existing review", func(t *testing.T, ctx *context.Context, resp *httptest.ResponseRecorder) {
		reviews, err := issues_model.FindReviews(db.DefaultContext, issues_model.FindReviewOptions{
			IssueID: 2,
		})
		assert.NoError(t, err)
		for _, r := range reviews {
			assert.NoError(t, issues_model.DeleteReview(db.DefaultContext, r))
		}
		ctx.Data["ShowOutdatedComments"] = true
		renderConversation(ctx, preparedComment, "timeline")
		assert.Equal(t, http.StatusOK, resp.Code)
		assert.NotContains(t, resp.Body.String(), `status-page-500`)
	})
}
