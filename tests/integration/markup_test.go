// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	auth_model "code.gitea.io/gitea/models/auth"
	api "code.gitea.io/gitea/modules/structs"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func TestRenderAlertBlocks(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user1")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteMisc)

	assertAlertBlock := func(t *testing.T, input, alertType, alertIcon string) {
		t.Helper()

		blockquoteAttr := fmt.Sprintf(`<blockquote class="gt-py-3 attention attention-%s"`, strings.ToLower(alertType))
		classAttr := fmt.Sprintf(`class="attention-%s"`, strings.ToLower(alertType))
		iconAttr := fmt.Sprintf(`class="svg octicon-%s"`, alertIcon)

		req := NewRequestWithJSON(t, "POST", "/api/v1/markdown", &api.MarkdownOption{
			Text: input,
			Mode: "markdown",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		body := resp.Body.String()
		assert.Contains(t, body, blockquoteAttr)
		assert.Contains(t, body, classAttr)
		assert.Contains(t, body, iconAttr)
	}

	t.Run("legacy style", func(t *testing.T) {
		for alertType, alertIcon := range map[string]string{"Note": "info", "Warning": "alert"} {
			t.Run(alertType, func(t *testing.T) {
				input := fmt.Sprintf(`> **%s**
>
> This is a %s.`, alertType, alertType)

				assertAlertBlock(t, input, alertType, alertIcon)
			})
		}
	})

	t.Run("modern style", func(t *testing.T) {
		for alertType, alertIcon := range map[string]string{
			"NOTE":      "info",
			"TIP":       "light-bulb",
			"IMPORTANT": "report",
			"WARNING":   "alert",
			"CAUTION":   "stop",
		} {
			t.Run(alertType, func(t *testing.T) {
				input := fmt.Sprintf(`> [!%s]
>
> This is a %s.`, alertType, alertType)

				assertAlertBlock(t, input, alertType, alertIcon)
			})
		}
	})
}
