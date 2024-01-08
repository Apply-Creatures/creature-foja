// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Copied and modified from https://github.com/ethantkoenig/rupture (MIT License)

package bleve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadata(t *testing.T) {
	dir := t.TempDir()

	meta, err := readIndexMetadata(dir)
	assert.NoError(t, err)
	assert.Equal(t, &IndexMetadata{}, meta)

	meta.Version = 24
	assert.NoError(t, writeIndexMetadata(dir, meta))

	meta, err = readIndexMetadata(dir)
	assert.NoError(t, err)
	assert.EqualValues(t, 24, meta.Version)
}
