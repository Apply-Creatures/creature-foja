// Copyright 2024  The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	webhook_model "code.gitea.io/gitea/models/webhook"
	"code.gitea.io/gitea/modules/log"
	webhook_module "code.gitea.io/gitea/modules/webhook"
)

var _ Handler = defaultHandler{}

type defaultHandler struct {
	forgejo bool
}

func (dh defaultHandler) Type() webhook_module.HookType {
	if dh.forgejo {
		return webhook_module.FORGEJO
	}
	return webhook_module.GITEA
}

func (defaultHandler) Metadata(*webhook_model.Webhook) any { return nil }

func (defaultHandler) FormFields(bind func(any)) FormFields {
	panic("TODO")
}

func (defaultHandler) NewRequest(ctx context.Context, w *webhook_model.Webhook, t *webhook_model.HookTask) (req *http.Request, body []byte, err error) {
	switch w.HTTPMethod {
	case "":
		log.Info("HTTP Method for %s webhook %s [ID: %d] is not set, defaulting to POST", w.Type, w.URL, w.ID)
		fallthrough
	case http.MethodPost:
		switch w.ContentType {
		case webhook_model.ContentTypeJSON:
			req, err = http.NewRequest("POST", w.URL, strings.NewReader(t.PayloadContent))
			if err != nil {
				return nil, nil, err
			}

			req.Header.Set("Content-Type", "application/json")
		case webhook_model.ContentTypeForm:
			forms := url.Values{
				"payload": []string{t.PayloadContent},
			}

			req, err = http.NewRequest("POST", w.URL, strings.NewReader(forms.Encode()))
			if err != nil {
				return nil, nil, err
			}

			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		default:
			return nil, nil, fmt.Errorf("invalid content type: %v", w.ContentType)
		}
	case http.MethodGet:
		u, err := url.Parse(w.URL)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid URL: %w", err)
		}
		vals := u.Query()
		vals["payload"] = []string{t.PayloadContent}
		u.RawQuery = vals.Encode()
		req, err = http.NewRequest("GET", u.String(), nil)
		if err != nil {
			return nil, nil, err
		}
	case http.MethodPut:
		switch w.Type {
		case webhook_module.MATRIX: // used when t.Version == 1
			txnID, err := getMatrixTxnID([]byte(t.PayloadContent))
			if err != nil {
				return nil, nil, err
			}
			url := fmt.Sprintf("%s/%s", w.URL, url.PathEscape(txnID))
			req, err = http.NewRequest("PUT", url, strings.NewReader(t.PayloadContent))
			if err != nil {
				return nil, nil, err
			}
		default:
			return nil, nil, fmt.Errorf("invalid http method: %v", w.HTTPMethod)
		}
	default:
		return nil, nil, fmt.Errorf("invalid http method: %v", w.HTTPMethod)
	}

	body = []byte(t.PayloadContent)
	return req, body, addDefaultHeaders(req, []byte(w.Secret), t, body)
}

func addDefaultHeaders(req *http.Request, secret []byte, t *webhook_model.HookTask, payloadContent []byte) error {
	var signatureSHA1 string
	var signatureSHA256 string
	if len(secret) > 0 {
		sig1 := hmac.New(sha1.New, secret)
		sig256 := hmac.New(sha256.New, secret)
		_, err := io.MultiWriter(sig1, sig256).Write(payloadContent)
		if err != nil {
			// this error should never happen, since the hashes are writing to []byte and always return a nil error.
			return fmt.Errorf("prepareWebhooks.sigWrite: %w", err)
		}
		signatureSHA1 = hex.EncodeToString(sig1.Sum(nil))
		signatureSHA256 = hex.EncodeToString(sig256.Sum(nil))
	}

	event := t.EventType.Event()
	eventType := string(t.EventType)
	req.Header.Add("X-Forgejo-Delivery", t.UUID)
	req.Header.Add("X-Forgejo-Event", event)
	req.Header.Add("X-Forgejo-Event-Type", eventType)
	req.Header.Add("X-Forgejo-Signature", signatureSHA256)
	req.Header.Add("X-Gitea-Delivery", t.UUID)
	req.Header.Add("X-Gitea-Event", event)
	req.Header.Add("X-Gitea-Event-Type", eventType)
	req.Header.Add("X-Gitea-Signature", signatureSHA256)
	req.Header.Add("X-Gogs-Delivery", t.UUID)
	req.Header.Add("X-Gogs-Event", event)
	req.Header.Add("X-Gogs-Event-Type", eventType)
	req.Header.Add("X-Gogs-Signature", signatureSHA256)
	req.Header.Add("X-Hub-Signature", "sha1="+signatureSHA1)
	req.Header.Add("X-Hub-Signature-256", "sha256="+signatureSHA256)
	req.Header["X-GitHub-Delivery"] = []string{t.UUID}
	req.Header["X-GitHub-Event"] = []string{event}
	req.Header["X-GitHub-Event-Type"] = []string{eventType}
	return nil
}
