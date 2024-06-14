// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package tests

import (
	"testing"

	forgejo_log "code.gitea.io/gitea/modules/log"
	driver_options "code.gitea.io/gitea/services/f3/driver/options"
	"code.gitea.io/gitea/services/f3/util"

	"code.forgejo.org/f3/gof3/v3/options"
)

func newTestOptions(t *testing.T) options.Interface {
	o := options.GetFactory(driver_options.Name)().(*driver_options.Options)
	o.SetLogger(util.NewF3Logger(nil, forgejo_log.GetLogger(forgejo_log.DEFAULT)))
	return o
}
