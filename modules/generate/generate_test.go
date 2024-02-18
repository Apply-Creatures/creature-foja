// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package generate

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeJwtSecret(t *testing.T) {
	_, err := DecodeJwtSecret("abcd")
	assert.ErrorContains(t, err, "invalid base64 decoded length")
	_, err = DecodeJwtSecret(strings.Repeat("a", 64))
	assert.ErrorContains(t, err, "invalid base64 decoded length")

	str32 := strings.Repeat("x", 32)
	encoded32 := base64.RawURLEncoding.EncodeToString([]byte(str32))
	decoded32, err := DecodeJwtSecret(encoded32)
	assert.NoError(t, err)
	assert.Equal(t, str32, string(decoded32))
}

func TestNewJwtSecret(t *testing.T) {
	secret, encoded, err := NewJwtSecret()
	assert.NoError(t, err)
	assert.Len(t, secret, 32)
	decoded, err := DecodeJwtSecret(encoded)
	assert.NoError(t, err)
	assert.Equal(t, secret, decoded)
}
