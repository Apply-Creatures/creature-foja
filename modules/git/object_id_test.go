// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidSHAPattern(t *testing.T) {
	h := Sha1ObjectFormat
	assert.True(t, h.IsValid("fee1"))
	assert.True(t, h.IsValid("abc000"))
	assert.True(t, h.IsValid("9023902390239023902390239023902390239023"))
	assert.False(t, h.IsValid("90239023902390239023902390239023902390239023"))
	assert.False(t, h.IsValid("abc"))
	assert.False(t, h.IsValid("123g"))
	assert.False(t, h.IsValid("some random text"))

	assert.Equal(t, "79ee38a6416c1ede423ec7ee0a8639ceea4aad22", ComputeBlobHash(Sha1ObjectFormat, []byte("some random blob")).String())
	assert.Equal(t, "d5c6407415d85df49592672aa421aed39b9db5e3", ComputeBlobHash(Sha1ObjectFormat, []byte("same length blob")).String())
	assert.Equal(t, "df0b5174ed06ae65aea40d43316bcbc21d82c9e3158ce2661df2ad28d7931dd6", ComputeBlobHash(Sha256ObjectFormat, []byte("some random blob")).String())
}
