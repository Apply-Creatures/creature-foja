// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"fmt"
	"net/http"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/json"
	"code.gitea.io/gitea/modules/log"
	webhook_module "code.gitea.io/gitea/modules/webhook"
)

type packagistHandler struct{}

func (packagistHandler) Type() webhook_module.HookType { return webhook_module.PACKAGIST }

type (
	// PackagistPayload represents a packagist payload
	// as expected by https://packagist.org/about
	PackagistPayload struct {
		PackagistRepository struct {
			URL string `json:"url"`
		} `json:"repository"`
	}

	// PackagistMeta contains the metadata for the webhook
	PackagistMeta struct {
		Username   string `json:"username"`
		APIToken   string `json:"api_token"`
		PackageURL string `json:"package_url"`
	}
)

// Metadata returns packagist metadata
func (packagistHandler) Metadata(w *webhook_model.Webhook) any {
	s := &PackagistMeta{}
	if err := json.Unmarshal([]byte(w.Meta), s); err != nil {
		log.Error("packagistHandler.Metadata(%d): %v", w.ID, err)
	}
	return s
}

// newPackagistRequest creates a request with the [PackagistPayload] for packagist (same payload for all events).
func (packagistHandler) NewRequest(ctx context.Context, w *webhook_model.Webhook, t *webhook_model.HookTask) (*http.Request, []byte, error) {
	meta := &PackagistMeta{}
	if err := json.Unmarshal([]byte(w.Meta), meta); err != nil {
		return nil, nil, fmt.Errorf("packagistHandler.NewRequest meta json: %w", err)
	}

	payload := PackagistPayload{
		PackagistRepository: struct {
			URL string `json:"url"`
		}{
			URL: meta.PackageURL,
		},
	}
	return newJSONRequestWithPayload(payload, w, t, false)
}
