// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package misc

import (
	"net/http"

	"code.gitea.io/gitea/modules/base"
	"code.gitea.io/gitea/services/context"
)

// tplSwagger swagger page template
const tplForgejoSwagger base.TplName = "swagger/forgejo-ui"

func SwaggerForgejo(ctx *context.Context) {
	ctx.Data["APIVersion"] = "v1"
	ctx.HTML(http.StatusOK, tplForgejoSwagger)
}
