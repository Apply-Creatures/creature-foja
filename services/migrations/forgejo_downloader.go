// Copyright 2023 The Forgejo Authors
// SPDX-License-Identifier: MIT

package migrations

import (
	"code.gitea.io/gitea/modules/structs"
)

func init() {
	RegisterDownloaderFactory(&ForgejoDownloaderFactory{})
}

type ForgejoDownloaderFactory struct {
	GiteaDownloaderFactory
}

func (f *ForgejoDownloaderFactory) GitServiceType() structs.GitServiceType {
	return structs.ForgejoService
}
