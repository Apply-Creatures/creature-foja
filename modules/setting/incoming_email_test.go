// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loadIncomingEmailFrom(t *testing.T) {
	cfg, _ := NewConfigProviderFromData("")
	sec := cfg.Section("email.incoming")
	sec.NewKey("ENABLED", "true")
	sec.NewKey("USER", "jane.doe@example.com")
	sec.NewKey("PASSWD", "y0u'll n3v3r gUess th1S!!1")
	sec.NewKey("REPLY_TO_ADDRESS", "forge+%{token}@example.com")

	loadIncomingEmailFrom(cfg)

	assert.EqualValues(t, "jane.doe@example.com", IncomingEmail.Username)
	assert.EqualValues(t, "y0u'll n3v3r gUess th1S!!1", IncomingEmail.Password)
}
