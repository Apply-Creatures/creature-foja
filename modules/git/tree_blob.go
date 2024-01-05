// Copyright 2015 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import "strings"

// GetBlobByPath get the blob object according the path
func (t *Tree) GetBlobByPath(relpath string) (*Blob, error) {
	entry, err := t.GetTreeEntryByPath(relpath)
	if err != nil {
		return nil, err
	}

	if !entry.IsDir() && !entry.IsSubModule() {
		return entry.Blob(), nil
	}

	return nil, ErrNotExist{"", relpath}
}

// GetBlobByFoldedPath returns the blob object at relpath, regardless of the
// case of relpath. If there are multiple files with the same case-insensitive
// name, the first one found will be returned.
func (t *Tree) GetBlobByFoldedPath(relpath string) (*Blob, error) {
	entries, err := t.ListEntries()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), relpath) {
			return t.GetBlobByPath(entry.Name())
		}
	}

	return nil, ErrNotExist{"", relpath}
}
