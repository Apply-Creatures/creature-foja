// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIIssueTemplateList(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

		t.Run("no templates", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/issue_templates", repo.FullName()))
			resp := MakeRequest(t, req, http.StatusOK)
			var issueTemplates []*api.IssueTemplate
			DecodeJSON(t, resp, &issueTemplates)
			assert.Empty(t, issueTemplates)
		})

		t.Run("existing template", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer func() {
				deleteFileInBranch(user, repo, "ISSUE_TEMPLATE/test.md", repo.DefaultBranch)
			}()

			err := createOrReplaceFileInBranch(user, repo, "ISSUE_TEMPLATE/test.md", repo.DefaultBranch,
				`---
name: 'Template Name'
about: 'This template is for testing!'
title: '[TEST] '
ref: 'main'
---

This is the template!`)
			assert.NoError(t, err)

			req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/issue_templates", repo.FullName()))
			resp := MakeRequest(t, req, http.StatusOK)
			var issueTemplates []*api.IssueTemplate
			DecodeJSON(t, resp, &issueTemplates)
			assert.Len(t, issueTemplates, 1)
			assert.Equal(t, "Template Name", issueTemplates[0].Name)
			assert.Equal(t, "This template is for testing!", issueTemplates[0].About)
			assert.Equal(t, "refs/heads/main", issueTemplates[0].Ref)
			assert.Equal(t, "ISSUE_TEMPLATE/test.md", issueTemplates[0].FileName)
		})
	})
}
