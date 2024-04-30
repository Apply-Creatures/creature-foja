// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"testing"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/json"
	webhook_module "code.gitea.io/gitea/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGiteaPayload(t *testing.T) {
	dh := defaultHandler{
		forgejo: false,
	}
	hook := &webhook_model.Webhook{
		RepoID:      3,
		IsActive:    true,
		Type:        webhook_module.GITEA,
		URL:         "https://gitea.example.com/",
		Meta:        ``,
		HTTPMethod:  "POST",
		ContentType: webhook_model.ContentTypeJSON,
	}

	// Woodpecker expects the ref to be short on tag creation only
	// https://github.com/woodpecker-ci/woodpecker/blob/00ccec078cdced80cf309cd4da460a5041d7991a/server/forge/gitea/helper.go#L134
	// see https://codeberg.org/codeberg/community/issues/1556
	t.Run("Create", func(t *testing.T) {
		p := createTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventCreate,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(context.Background(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitea.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "test", body.Ref) // short ref
	})

	t.Run("Push", func(t *testing.T) {
		p := pushTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventPush,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(context.Background(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitea.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})

	t.Run("Delete", func(t *testing.T) {
		p := deleteTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventDelete,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(context.Background(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://gitea.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "test", body.Ref) // short ref
	})
}

func TestForgejoPayload(t *testing.T) {
	dh := defaultHandler{
		forgejo: true,
	}
	hook := &webhook_model.Webhook{
		RepoID:      3,
		IsActive:    true,
		Type:        webhook_module.FORGEJO,
		URL:         "https://forgejo.example.com/",
		Meta:        ``,
		HTTPMethod:  "POST",
		ContentType: webhook_model.ContentTypeJSON,
	}

	// always return the full ref for consistency
	t.Run("Create", func(t *testing.T) {
		p := createTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventCreate,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(context.Background(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://forgejo.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})

	t.Run("Push", func(t *testing.T) {
		p := pushTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventPush,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(context.Background(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://forgejo.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})

	t.Run("Delete", func(t *testing.T) {
		p := deleteTestPayload()
		data, err := p.JSONPayload()
		require.NoError(t, err)

		task := &webhook_model.HookTask{
			HookID:         hook.ID,
			EventType:      webhook_module.HookEventDelete,
			PayloadContent: string(data),
			PayloadVersion: 2,
		}

		req, reqBody, err := dh.NewRequest(context.Background(), hook, task)
		require.NoError(t, err)
		require.NotNil(t, req)
		require.NotNil(t, reqBody)

		assert.Equal(t, "POST", req.Method)
		assert.Equal(t, "https://forgejo.example.com/", req.URL.String())
		assert.Equal(t, "sha256=", req.Header.Get("X-Hub-Signature-256"))
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
		var body struct {
			Ref string `json:"ref"`
		}
		err = json.NewDecoder(req.Body).Decode(&body)
		assert.NoError(t, err)
		assert.Equal(t, "refs/heads/test", body.Ref) // full ref
	})
}
