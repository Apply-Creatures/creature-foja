// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/json"
	webhook_module "code.gitea.io/gitea/modules/webhook"
	"code.gitea.io/gitea/services/release"
	"code.gitea.io/gitea/tests"

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

func TestWebhookReleaseEvents(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	w := unittest.AssertExistsAndLoadBean(t, &webhook_model.Webhook{
		ID:     1,
		RepoID: repo.ID,
	})
	w.HookEvent = &webhook_module.HookEvent{
		SendEverything: true,
	}
	assert.NoError(t, w.UpdateEvent())
	assert.NoError(t, webhook_model.UpdateWebhook(db.DefaultContext, w))

	hookTasks := retrieveHookTasks(t, w.ID, true)

	gitRepo, err := gitrepo.OpenRepository(git.DefaultContext, repo)
	assert.NoError(t, err)
	defer gitRepo.Close()

	t.Run("CreateRelease", func(t *testing.T) {
		assert.NoError(t, release.CreateRelease(gitRepo, &repo_model.Release{
			RepoID:       repo.ID,
			Repo:         repo,
			PublisherID:  user.ID,
			Publisher:    user,
			TagName:      "v1.1.1",
			Target:       "master",
			Title:        "v1.1.1 is released",
			Note:         "v1.1.1 is released",
			IsDraft:      false,
			IsPrerelease: false,
			IsTag:        false,
		}, nil, ""))

		// check the newly created hooktasks
		hookTasksLenBefore := len(hookTasks)
		hookTasks = retrieveHookTasks(t, w.ID, false)

		checkHookTasks(t, map[webhook_module.HookEventType]string{
			webhook_module.HookEventRelease: "published",
			webhook_module.HookEventCreate:  "", // a tag was created as well
			webhook_module.HookEventPush:    "", // the tag creation also means a push event
		}, hookTasks[:len(hookTasks)-hookTasksLenBefore])

		t.Run("UpdateRelease", func(t *testing.T) {
			rel := unittest.AssertExistsAndLoadBean(t, &repo_model.Release{RepoID: repo.ID, TagName: "v1.1.1"})
			assert.NoError(t, release.UpdateRelease(db.DefaultContext, user, gitRepo, rel, nil, nil, nil, false))

			// check the newly created hooktasks
			hookTasksLenBefore := len(hookTasks)
			hookTasks = retrieveHookTasks(t, w.ID, false)

			checkHookTasks(t, map[webhook_module.HookEventType]string{
				webhook_module.HookEventRelease: "updated",
			}, hookTasks[:len(hookTasks)-hookTasksLenBefore])
		})
	})

	t.Run("CreateNewTag", func(t *testing.T) {
		assert.NoError(t, release.CreateNewTag(db.DefaultContext,
			user,
			repo,
			"master",
			"v1.1.2",
			"v1.1.2 is tagged",
		))

		// check the newly created hooktasks
		hookTasksLenBefore := len(hookTasks)
		hookTasks = retrieveHookTasks(t, w.ID, false)

		checkHookTasks(t, map[webhook_module.HookEventType]string{
			webhook_module.HookEventCreate: "", // tag was created as well
			webhook_module.HookEventPush:   "", // the tag creation also means a push event
		}, hookTasks[:len(hookTasks)-hookTasksLenBefore])

		t.Run("UpdateRelease", func(t *testing.T) {
			rel := unittest.AssertExistsAndLoadBean(t, &repo_model.Release{RepoID: repo.ID, TagName: "v1.1.2"})
			assert.NoError(t, release.UpdateRelease(db.DefaultContext, user, gitRepo, rel, nil, nil, nil, true))

			// check the newly created hooktasks
			hookTasksLenBefore := len(hookTasks)
			hookTasks = retrieveHookTasks(t, w.ID, false)

			checkHookTasks(t, map[webhook_module.HookEventType]string{
				webhook_module.HookEventRelease: "published",
			}, hookTasks[:len(hookTasks)-hookTasksLenBefore])
		})
	})
}

func checkHookTasks(t *testing.T, expectedActions map[webhook_module.HookEventType]string, hookTasks []*webhook_model.HookTask) {
	t.Helper()
	for _, hookTask := range hookTasks {
		expectedAction, ok := expectedActions[hookTask.EventType]
		if !ok {
			t.Errorf("unexpected (or duplicated) event %q", hookTask.EventType)
		}
		var payload struct {
			Action string `json:"action"`
		}
		assert.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payload))
		assert.Equal(t, expectedAction, payload.Action, "unexpected action for %q event", hookTask.EventType)
		delete(expectedActions, hookTask.EventType)
	}
	assert.Empty(t, expectedActions)
}
