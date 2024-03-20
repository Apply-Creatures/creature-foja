// Copyright 2024  The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	webhook_module "code.gitea.io/gitea/modules/webhook"
)

type gogsHandler struct{ defaultHandler }

func (gogsHandler) Type() webhook_module.HookType { return webhook_module.GOGS }
