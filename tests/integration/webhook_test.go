// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/json"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
)

func TestWebhookPayloadRef(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, giteaURL *url.URL) {
		w := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{ID: 1})
		w.HookEvent = &webhook_module.HookEvent{
			SendEverything: true,
		}
		assert.NoError(t, w.UpdateEvent())
		assert.NoError(t, webhook_model.UpdateWebhook(db.DefaultContext, w))

		hookTasks := retrieveHookTasks(t, w.ID, true)
		hookTasksLenBefore := len(hookTasks)

		session := loginUser(t, "user2")
		// create new branch
		csrf := GetCSRF(t, session, "user2/repo1")
		req := NewRequestWithValues(t, "POST", "user2/repo1/branches/_new/branch/master",
			map[string]string{
				"_csrf":           csrf,
				"new_branch_name": "arbre",
				"create_tag":      "false",
			},
		)
		session.MakeRequest(t, req, http.StatusSeeOther)
		// delete the created branch
		req = NewRequestWithValues(t, "POST", "user2/repo1/branches/delete?name=arbre",
			map[string]string{
				"_csrf": csrf,
			},
		)
		session.MakeRequest(t, req, http.StatusOK)

		// check the newly created hooktasks
		hookTasks = retrieveHookTasks(t, w.ID, false)
		expected := map[webhook_module.HookEventType]bool{
			webhook_module.HookEventCreate: true,
			webhook_module.HookEventPush:   true, // the branch creation also creates a push event
			webhook_module.HookEventDelete: true,
		}
		for _, hookTask := range hookTasks[:len(hookTasks)-hookTasksLenBefore] {
			if !expected[hookTask.EventType] {
				t.Errorf("unexpected (or duplicated) event %q", hookTask.EventType)
			}

			var payload struct {
				Ref string `json:"ref"`
			}
			assert.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payload))
			assert.Equal(t, "refs/heads/arbre", payload.Ref, "unexpected ref for %q event", hookTask.EventType)
			delete(expected, hookTask.EventType)
		}
		assert.Empty(t, expected)
	})
}
