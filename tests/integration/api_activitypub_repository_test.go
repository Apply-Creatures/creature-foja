// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/activitypub"
	forgefed_modules "code.gitea.io/gitea/modules/forgefed"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers"

	"github.com/stretchr/testify/assert"
)

func TestActivityPubRepository(t *testing.T) {
	setting.Federation.Enabled = true
	testWebRoutes = routers.NormalRoutes()
	defer func() {
		setting.Federation.Enabled = false
		testWebRoutes = routers.NormalRoutes()
	}()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		repositoryID := 2
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/activitypub/repository-id/%v", repositoryID))
		resp := MakeRequest(t, req, http.StatusOK)
		body := resp.Body.Bytes()
		assert.Contains(t, string(body), "@context")

		var repository forgefed_modules.Repository
		err := repository.UnmarshalJSON(body)
		assert.NoError(t, err)

		assert.Regexp(t, fmt.Sprintf("activitypub/repository-id/%v$", repositoryID), repository.GetID().String())
	})
}

func TestActivityPubMissingRepository(t *testing.T) {
	setting.Federation.Enabled = true
	testWebRoutes = routers.NormalRoutes()
	defer func() {
		setting.Federation.Enabled = false
		testWebRoutes = routers.NormalRoutes()
	}()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		repositoryID := 9999999
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/activitypub/repository-id/%v", repositoryID))
		resp := MakeRequest(t, req, http.StatusNotFound)
		assert.Contains(t, resp.Body.String(), "repository does not exist")
	})
}

func TestActivityPubRepositoryInboxValid(t *testing.T) {
	setting.Federation.Enabled = true
	testWebRoutes = routers.NormalRoutes()
	defer func() {
		setting.Federation.Enabled = false
		testWebRoutes = routers.NormalRoutes()
	}()

	srv := httptest.NewServer(testWebRoutes)
	defer srv.Close()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		appURL := setting.AppURL
		setting.AppURL = srv.URL + "/"
		defer func() {
			setting.Database.LogSQL = false
			setting.AppURL = appURL
		}()
		actionsUser := user.NewActionsUser()
		repositoryID := 2
		c, err := activitypub.NewClient(db.DefaultContext, actionsUser, "not used")
		assert.NoError(t, err)
		repoInboxURL := fmt.Sprintf("%s/api/v1/activitypub/repository-id/%v/inbox",
			srv.URL, repositoryID)

		activity := []byte(fmt.Sprintf(`{"type":"Like","startTime":"2024-03-27T00:00:00Z","actor":"%s/api/v1/activitypub/user-id/2","object":"%s/api/v1/activitypub/repository-id/%v"}`,
			srv.URL, srv.URL, repositoryID))
		resp, err := c.Post(activity, repoInboxURL)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})
}

func TestActivityPubRepositoryInboxInvalid(t *testing.T) {
	setting.Federation.Enabled = true
	testWebRoutes = routers.NormalRoutes()
	defer func() {
		setting.Federation.Enabled = false
		testWebRoutes = routers.NormalRoutes()
	}()

	srv := httptest.NewServer(testWebRoutes)
	defer srv.Close()

	onGiteaRun(t, func(*testing.T, *url.URL) {
		appURL := setting.AppURL
		setting.AppURL = srv.URL + "/"
		defer func() {
			setting.Database.LogSQL = false
			setting.AppURL = appURL
		}()
		actionsUser := user.NewActionsUser()
		repositoryID := 2
		c, err := activitypub.NewClient(db.DefaultContext, actionsUser, "not used")
		assert.NoError(t, err)
		repoInboxURL := fmt.Sprintf("%s/api/v1/activitypub/repository-id/%v/inbox",
			srv.URL, repositoryID)

		activity := []byte(`{"type":"Wrong"}`)
		resp, err := c.Post(activity, repoInboxURL)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotAcceptable, resp.StatusCode)
	})
}
