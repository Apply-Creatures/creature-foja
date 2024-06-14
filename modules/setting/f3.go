// SPDX-License-Identifier: MIT

package setting

import (
	"code.gitea.io/gitea/modules/log"
)

// Friendly Forge Format (F3) settings
var (
	F3 = struct {
		Enabled bool
	}{
		Enabled: false,
	}
)

func loadF3From(rootCfg ConfigProvider) {
	if err := rootCfg.Section("F3").MapTo(&F3); err != nil {
		log.Fatal("Failed to map F3 settings: %v", err)
	}
}
