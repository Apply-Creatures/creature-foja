// Copyright 2017 The Gogs Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/routers"
	"code.gitea.io/gitea/tests"
)

func TestCreateForkNoLogin(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/forks", &api.CreateForkOption{})
	MakeRequest(t, req, http.StatusUnauthorized)
}

func TestAPIDisabledForkRepo(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		defer test.MockVariableValue(&setting.Repository.DisableForks, true)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		t.Run("fork listing", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/repos/user2/repo1/forks")
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("forking", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, "user5")
			token := getTokenForLoggedInUser(t, session)

			req := NewRequestWithJSON(t, "POST", "/api/v1/repos/user2/repo1/forks", &api.CreateForkOption{}).AddTokenAuth(token)
			session.MakeRequest(t, req, http.StatusNotFound)
		})
	})
}
