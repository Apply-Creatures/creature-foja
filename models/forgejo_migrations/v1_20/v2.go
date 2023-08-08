// SPDX-License-Identifier: MIT

package forgejo_v1_20 //nolint:revive

import (
	"xorm.io/xorm"
)

func CreateSemVerTable(x *xorm.Engine) error {
	type ForgejoSemVer struct {
		Version string
	}

	return x.Sync(new(ForgejoSemVer))
}
