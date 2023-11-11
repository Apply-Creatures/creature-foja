// Copyright The Forgejo Authors.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	v1 "code.gitea.io/gitea/routers/api/forgejo/v1"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIForgejoVersion(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/forgejo/v1/version")
	resp := MakeRequest(t, req, http.StatusOK)

	var version v1.Version
	DecodeJSON(t, resp, &version)
	assert.Equal(t, "1.0.0", *version.Version)
}
