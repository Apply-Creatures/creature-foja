// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import "xorm.io/xorm"

func AddHideArchiveLinksToRelease(x *xorm.Engine) error {
	type Release struct {
		ID               int64 `xorm:"pk autoincr"`
		HideArchiveLinks bool  `xorm:"NOT NULL DEFAULT false"`
	}

	return x.Sync(&Release{})
}
