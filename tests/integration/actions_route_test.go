// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	actions_model "code.gitea.io/gitea/models/actions"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	files_service "code.gitea.io/gitea/services/repository/files"

	"github.com/stretchr/testify/assert"
)

func TestActionsWebRouteLatestRun(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// create the repo
		repo, _, f := CreateDeclarativeRepo(t, user2, "",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      ".gitea/workflows/pr.yml",
					ContentReader: strings.NewReader("name: test\non:\n  push:\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo helloworld\n"),
				},
			},
		)
		defer f()

		// a run has been created
		assert.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID}))

		// Hit the `/actions/runs/latest` route
		req := NewRequest(t, "GET", fmt.Sprintf("%s/actions/runs/latest", repo.HTMLURL()))
		resp := MakeRequest(t, req, http.StatusTemporaryRedirect)

		// Verify that it redirects to the run we just created
		expectedURI := fmt.Sprintf("%s/actions/runs/1", repo.HTMLURL())
		assert.Equal(t, expectedURI, resp.Header().Get("Location"))
	})
}
