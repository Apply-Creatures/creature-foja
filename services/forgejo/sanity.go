// SPDX-License-Identifier: MIT

package forgejo

import (
	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

var (
	ForgejoV5DatabaseVersion = int64(260)
	ForgejoV4DatabaseVersion = int64(244)
)

var logFatal = log.Fatal

func fatal(err error) error {
	logFatal("%v", err)
	return err
}

func PreMigrationSanityChecks(e db.Engine, dbVersion int64, cfg setting.ConfigProvider) error {
	return nil
}
