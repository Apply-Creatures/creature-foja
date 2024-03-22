// Copyright 2024  The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"html/template"
	"net/http"

	webhook_model "code.gitea.io/gitea/models/webhook"
	webhook_module "code.gitea.io/gitea/modules/webhook"
	"code.gitea.io/gitea/services/forms"
)

type gogsHandler struct{ defaultHandler }

func (gogsHandler) Type() webhook_module.HookType { return webhook_module.GOGS }
func (gogsHandler) Icon(size int) template.HTML   { return imgIcon("gogs.ico", size) }

func (gogsHandler) FormFields(bind func(any)) FormFields {
	var form struct {
		forms.WebhookForm
		PayloadURL  string `binding:"Required;ValidUrl"`
		ContentType int    `binding:"Required"`
		Secret      string
	}
	bind(&form)

	contentType := webhook_model.ContentTypeJSON
	if webhook_model.HookContentType(form.ContentType) == webhook_model.ContentTypeForm {
		contentType = webhook_model.ContentTypeForm
	}
	return FormFields{
		WebhookForm: form.WebhookForm,
		URL:         form.PayloadURL,
		ContentType: contentType,
		Secret:      form.Secret,
		HTTPMethod:  http.MethodPost,
		Metadata:    nil,
	}
}
