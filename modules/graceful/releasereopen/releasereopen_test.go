// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package releasereopen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testReleaseReopener struct {
	count int
}

func (t *testReleaseReopener) ReleaseReopen() error {
	t.count++
	return nil
}

func TestManager(t *testing.T) {
	m := NewManager()

	t1 := &testReleaseReopener{}
	t2 := &testReleaseReopener{}
	t3 := &testReleaseReopener{}

	_ = m.Register(t1)
	c2 := m.Register(t2)
	_ = m.Register(t3)

	require.NoError(t, m.ReleaseReopen())
	assert.EqualValues(t, 1, t1.count)
	assert.EqualValues(t, 1, t2.count)
	assert.EqualValues(t, 1, t3.count)

	c2()

	require.NoError(t, m.ReleaseReopen())
	assert.EqualValues(t, 2, t1.count)
	assert.EqualValues(t, 1, t2.count)
	assert.EqualValues(t, 2, t3.count)
}
