// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package optional_test

import (
	"testing"

	"code.gitea.io/gitea/modules/optional"

	"github.com/stretchr/testify/assert"
)

func TestOption(t *testing.T) {
	var uninitialized optional.Option[int]
	assert.False(t, uninitialized.Has())
	assert.Equal(t, int(0), uninitialized.Value())
	assert.Equal(t, int(1), uninitialized.ValueOrDefault(1))

	none := optional.None[int]()
	assert.False(t, none.Has())
	assert.Equal(t, int(0), none.Value())
	assert.Equal(t, int(1), none.ValueOrDefault(1))

	some := optional.Some[int](1)
	assert.True(t, some.Has())
	assert.Equal(t, int(1), some.Value())
	assert.Equal(t, int(1), some.ValueOrDefault(2))

	var ptr *int
	assert.False(t, optional.FromPtr(ptr).Has())

	int1 := 1
	opt1 := optional.FromPtr(&int1)
	assert.True(t, opt1.Has())
	assert.Equal(t, int(1), opt1.Value())

	assert.False(t, optional.FromNonDefault("").Has())

	opt2 := optional.FromNonDefault("test")
	assert.True(t, opt2.Has())
	assert.Equal(t, "test", opt2.Value())

	assert.False(t, optional.FromNonDefault(0).Has())

	opt3 := optional.FromNonDefault(1)
	assert.True(t, opt3.Has())
	assert.Equal(t, int(1), opt3.Value())
}
