// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"os"
	"testing"

	"github.com/mholt/archiver/v3"
	"github.com/stretchr/testify/assert"
)

type mockArchiver struct {
	addedFiles []string
}

func (mockArchiver) Create(out io.Writer) error {
	return nil
}

func (m *mockArchiver) Write(f archiver.File) error {
	m.addedFiles = append(m.addedFiles, f.Name())
	return nil
}

func (mockArchiver) Close() error {
	return nil
}

func TestAddRecursiveExclude(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		dir := t.TempDir()
		archiver := &mockArchiver{}

		err := addRecursiveExclude(archiver, "", dir, []string{}, false)
		assert.NoError(t, err)
		assert.Empty(t, archiver.addedFiles)
	})

	t.Run("Single file", func(t *testing.T) {
		dir := t.TempDir()
		err := os.WriteFile(dir+"/example", nil, 0o666)
		assert.NoError(t, err)

		t.Run("No exclude", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, nil, false)
			assert.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 1)
			assert.EqualValues(t, "example", archiver.addedFiles[0])
		})

		t.Run("With exclude", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/example"}, false)
			assert.NoError(t, err)
			assert.Empty(t, archiver.addedFiles)
		})
	})

	t.Run("File inside directory", func(t *testing.T) {
		dir := t.TempDir()
		err := os.MkdirAll(dir+"/deep/nested/folder", 0o750)
		assert.NoError(t, err)
		err = os.WriteFile(dir+"/deep/nested/folder/example", nil, 0o666)
		assert.NoError(t, err)
		err = os.WriteFile(dir+"/deep/nested/folder/another-file", nil, 0o666)
		assert.NoError(t, err)

		t.Run("No exclude", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, nil, false)
			assert.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 5)
			assert.EqualValues(t, "deep", archiver.addedFiles[0])
			assert.EqualValues(t, "deep/nested", archiver.addedFiles[1])
			assert.EqualValues(t, "deep/nested/folder", archiver.addedFiles[2])
			assert.EqualValues(t, "deep/nested/folder/example", archiver.addedFiles[3])
			assert.EqualValues(t, "deep/nested/folder/another-file", archiver.addedFiles[4])
		})

		t.Run("Exclude first directory", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/deep"}, false)
			assert.NoError(t, err)
			assert.Empty(t, archiver.addedFiles)
		})

		t.Run("Exclude nested directory", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/deep/nested/folder"}, false)
			assert.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 2)
			assert.EqualValues(t, "deep", archiver.addedFiles[0])
			assert.EqualValues(t, "deep/nested", archiver.addedFiles[1])
		})

		t.Run("Exclude file", func(t *testing.T) {
			archiver := &mockArchiver{}

			err = addRecursiveExclude(archiver, "", dir, []string{dir + "/deep/nested/folder/example"}, false)
			assert.NoError(t, err)
			assert.Len(t, archiver.addedFiles, 4)
			assert.EqualValues(t, "deep", archiver.addedFiles[0])
			assert.EqualValues(t, "deep/nested", archiver.addedFiles[1])
			assert.EqualValues(t, "deep/nested/folder", archiver.addedFiles[2])
			assert.EqualValues(t, "deep/nested/folder/another-file", archiver.addedFiles[3])
		})
	})
}
