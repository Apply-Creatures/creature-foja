// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/forgefed"
	"code.gitea.io/gitea/models/unittest"
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

	federatedRoutes := http.NewServeMux()
	federatedRoutes.HandleFunc("/.well-known/nodeinfo",
		func(res http.ResponseWriter, req *http.Request) {
			// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/.well-known/nodeinfo
			responseBody := fmt.Sprintf(`{"links":[{"href":"http://%s/api/v1/nodeinfo","rel":"http://nodeinfo.diaspora.software/ns/schema/2.1"}]}`, req.Host)
			t.Logf("response: %s", responseBody)
			// TODO: as soon as content-type will become important:  content-type: application/json;charset=utf-8
			fmt.Fprint(res, responseBody)
		})
	federatedRoutes.HandleFunc("/api/v1/nodeinfo",
		func(res http.ResponseWriter, req *http.Request) {
			// curl -H "Accept: application/json" https://federated-repo.prod.meissa.de/api/v1/nodeinfo
			responseBody := fmt.Sprintf(`{"version":"2.1","software":{"name":"forgejo","version":"1.20.0+dev-3183-g976d79044",` +
				`"repository":"https://codeberg.org/forgejo/forgejo.git","homepage":"https://forgejo.org/"},` +
				`"protocols":["activitypub"],"services":{"inbound":[],"outbound":["rss2.0"]},` +
				`"openRegistrations":true,"usage":{"users":{"total":14,"activeHalfyear":2}},"metadata":{}}`)
			fmt.Fprint(res, responseBody)
		})
	federatedRoutes.HandleFunc("/",
		func(res http.ResponseWriter, req *http.Request) {
			t.Errorf("Unhandled request: %q", req.URL.EscapedPath())
		})
	federatedSrv := httptest.NewServer(federatedRoutes)
	defer federatedSrv.Close()

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
		repoInboxURL := fmt.Sprintf(
			"%s/api/v1/activitypub/repository-id/%v/inbox",
			srv.URL, repositoryID)

		activity := []byte(fmt.Sprintf(
			`{"type":"Like",`+
				`"startTime":"%s",`+
				`"actor":"%s/api/v1/activitypub/user-id/2",`+
				`"object":"%s/api/v1/activitypub/repository-id/%v"}`,
			time.Now().UTC().Format(time.RFC3339),
			federatedSrv.URL, srv.URL, repositoryID))
		t.Logf("activity: %s", activity)
		resp, err := c.Post(activity, repoInboxURL)

		assert.NoError(t, err)
		assert.Equal(t, http.StatusNoContent, resp.StatusCode)

		federationHost := unittest.AssertExistsAndLoadBean(t, &forgefed.FederationHost{ID: 1})
		assert.Equal(t, "127.0.0.1", federationHost.HostFqdn)

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
