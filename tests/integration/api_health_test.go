package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/routers/web/healthcheck"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestApiHeatlhCheck(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("Test health-check pass", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/healthz")
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Contains(t, resp.Header().Values("Cache-Control"), "no-store")

		var status healthcheck.Response
		DecodeJSON(t, resp, &status)
		assert.Equal(t, healthcheck.Pass, status.Status)
		assert.Equal(t, setting.AppName, status.Description)
	})
}
