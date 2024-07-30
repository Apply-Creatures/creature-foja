// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetBranches(t *testing.T) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	require.NoError(t, err)
	defer bareRepo1.Close()

	branches, countAll, err := bareRepo1.GetBranchNames(0, 2)

	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.EqualValues(t, 3, countAll)
	assert.ElementsMatch(t, []string{"master", "branch2"}, branches)

	branches, countAll, err = bareRepo1.GetBranchNames(0, 0)

	require.NoError(t, err)
	assert.Len(t, branches, 3)
	assert.EqualValues(t, 3, countAll)
	assert.ElementsMatch(t, []string{"master", "branch2", "branch1"}, branches)

	branches, countAll, err = bareRepo1.GetBranchNames(5, 1)

	require.NoError(t, err)
	assert.Empty(t, branches)
	assert.EqualValues(t, 3, countAll)
	assert.ElementsMatch(t, []string{}, branches)
}

func BenchmarkRepository_GetBranches(b *testing.B) {
	bareRepo1Path := filepath.Join(testReposDir, "repo1_bare")
	bareRepo1, err := openRepositoryWithDefaultContext(bareRepo1Path)
	if err != nil {
		b.Fatal(err)
	}
	defer bareRepo1.Close()

	for i := 0; i < b.N; i++ {
		_, _, err := bareRepo1.GetBranchNames(0, 0)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestGetRefsBySha(t *testing.T) {
	bareRepo5Path := filepath.Join(testReposDir, "repo5_pulls")
	bareRepo5, err := OpenRepository(DefaultContext, bareRepo5Path)
	if err != nil {
		t.Fatal(err)
	}
	defer bareRepo5.Close()

	// do not exist
	branches, err := bareRepo5.GetRefsBySha("8006ff9adbf0cb94da7dad9e537e53817f9fa5c0", "")
	require.NoError(t, err)
	assert.Empty(t, branches)

	// refs/pull/1/head
	branches, err = bareRepo5.GetRefsBySha("c83380d7056593c51a699d12b9c00627bd5743e9", PullPrefix)
	require.NoError(t, err)
	assert.EqualValues(t, []string{"refs/pull/1/head"}, branches)

	branches, err = bareRepo5.GetRefsBySha("d8e0bbb45f200e67d9a784ce55bd90821af45ebd", BranchPrefix)
	require.NoError(t, err)
	assert.EqualValues(t, []string{"refs/heads/master", "refs/heads/master-clone"}, branches)

	branches, err = bareRepo5.GetRefsBySha("58a4bcc53ac13e7ff76127e0fb518b5262bf09af", BranchPrefix)
	require.NoError(t, err)
	assert.EqualValues(t, []string{"refs/heads/test-patch-1"}, branches)
}

func BenchmarkGetRefsBySha(b *testing.B) {
	bareRepo5Path := filepath.Join(testReposDir, "repo5_pulls")
	bareRepo5, err := OpenRepository(DefaultContext, bareRepo5Path)
	if err != nil {
		b.Fatal(err)
	}
	defer bareRepo5.Close()

	_, _ = bareRepo5.GetRefsBySha("8006ff9adbf0cb94da7dad9e537e53817f9fa5c0", "")
	_, _ = bareRepo5.GetRefsBySha("d8e0bbb45f200e67d9a784ce55bd90821af45ebd", "")
	_, _ = bareRepo5.GetRefsBySha("c83380d7056593c51a699d12b9c00627bd5743e9", "")
	_, _ = bareRepo5.GetRefsBySha("58a4bcc53ac13e7ff76127e0fb518b5262bf09af", "")
}
