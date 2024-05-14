// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
)

func (id ActorID) AsWellKnownNodeInfoURI() string {
	wellKnownPath := ".well-known/nodeinfo"
	var result string
	if id.Port == "" {
		result = fmt.Sprintf("%s://%s/%s", id.Schema, id.Host, wellKnownPath)
	} else {
		result = fmt.Sprintf("%s://%s:%s/%s", id.Schema, id.Host, id.Port, wellKnownPath)
	}
	return result
}
