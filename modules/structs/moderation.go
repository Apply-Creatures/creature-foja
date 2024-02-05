// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import "time"

// BlockedUser represents a blocked user.
type BlockedUser struct {
	BlockID int64 `json:"block_id"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
}
