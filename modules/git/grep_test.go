// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGrepSearch(t *testing.T) {
	repo, err := openRepositoryWithDefaultContext(filepath.Join(testReposDir, "language_stats_repo"))
	assert.NoError(t, err)
	defer repo.Close()

	res, err := GrepSearch(context.Background(), repo, "void", GrepOptions{})
	assert.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{3},
			LineCodes:   []string{" public static void main(String[] args)"},
		},
		{
			Filename:    "main.vendor.java",
			LineNumbers: []int{3},
			LineCodes:   []string{" public static void main(String[] args)"},
		},
	}, res)

	res, err = GrepSearch(context.Background(), repo, "void", GrepOptions{MaxResultLimit: 1})
	assert.NoError(t, err)
	assert.Equal(t, []*GrepResult{
		{
			Filename:    "java-hello/main.java",
			LineNumbers: []int{3},
			LineCodes:   []string{" public static void main(String[] args)"},
		},
	}, res)

	res, err = GrepSearch(context.Background(), repo, "no-such-content", GrepOptions{})
	assert.NoError(t, err)
	assert.Len(t, res, 0)

	res, err = GrepSearch(context.Background(), &Repository{Path: "no-such-git-repo"}, "no-such-content", GrepOptions{})
	assert.Error(t, err)
	assert.Len(t, res, 0)
}

func TestGrepLongFiles(t *testing.T) {
	tmpDir := t.TempDir()

	err := InitRepository(DefaultContext, tmpDir, false, Sha1ObjectFormat.Name())
	assert.NoError(t, err)

	gitRepo, err := openRepositoryWithDefaultContext(tmpDir)
	assert.NoError(t, err)
	defer gitRepo.Close()

	assert.NoError(t, os.WriteFile(path.Join(tmpDir, "README.md"), bytes.Repeat([]byte{'a'}, 65*1024), 0o666))

	err = AddChanges(tmpDir, true)
	assert.NoError(t, err)

	err = CommitChanges(tmpDir, CommitChangesOptions{Message: "Long file"})
	assert.NoError(t, err)

	res, err := GrepSearch(context.Background(), gitRepo, "a", GrepOptions{})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Len(t, res[0].LineCodes[0], 65*1024)
}
