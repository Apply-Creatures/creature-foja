// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

type ObjectID interface {
	String() string
	IsZero() bool
	RawValue() []byte
	Type() ObjectFormat
}

/* SHA1 */
type Sha1Hash [20]byte

func (h *Sha1Hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h *Sha1Hash) IsZero() bool {
	empty := Sha1Hash{}
	return bytes.Equal(empty[:], h[:])
}
func (h *Sha1Hash) RawValue() []byte { return h[:] }
func (*Sha1Hash) Type() ObjectFormat { return Sha1ObjectFormat }

var _ ObjectID = &Sha1Hash{}

func MustIDFromString(hexHash string) ObjectID {
	id, err := NewIDFromString(hexHash)
	if err != nil {
		panic(err)
	}
	return id
}

/* SHA256 */
type Sha256Hash [32]byte

func (h *Sha256Hash) String() string {
	return hex.EncodeToString(h[:])
}

func (h *Sha256Hash) IsZero() bool {
	empty := Sha256Hash{}
	return bytes.Equal(empty[:], h[:])
}
func (h *Sha256Hash) RawValue() []byte { return h[:] }
func (*Sha256Hash) Type() ObjectFormat { return Sha256ObjectFormat }

/* utility */
func NewIDFromString(hexHash string) (ObjectID, error) {
	var theObjectFormat ObjectFormat
	for _, objectFormat := range SupportedObjectFormats {
		if len(hexHash) == objectFormat.FullLength() {
			theObjectFormat = objectFormat
			break
		}
	}

	if theObjectFormat == nil {
		return nil, fmt.Errorf("length %d has no matched object format: %s", len(hexHash), hexHash)
	}

	b, err := hex.DecodeString(hexHash)
	if err != nil {
		return nil, err
	}

	if len(b) != theObjectFormat.FullLength()/2 {
		return theObjectFormat.EmptyObjectID(), fmt.Errorf("length must be %d: %v", theObjectFormat.FullLength(), b)
	}
	return theObjectFormat.MustID(b), nil
}

// IsEmptyCommitID checks if an hexadecimal string represents an empty commit according to git (only '0').
// If objectFormat is not nil, the length will be checked as well (otherwise the lenght must match the sha1 or sha256 length).
func IsEmptyCommitID(commitID string, objectFormat ObjectFormat) bool {
	if commitID == "" {
		return true
	}
	if objectFormat == nil {
		if Sha1ObjectFormat.FullLength() != len(commitID) && Sha256ObjectFormat.FullLength() != len(commitID) {
			return false
		}
	} else if objectFormat.FullLength() != len(commitID) {
		return false
	}
	for _, c := range commitID {
		if c != '0' {
			return false
		}
	}
	return true
}

// ComputeBlobHash compute the hash for a given blob content
func ComputeBlobHash(hashType ObjectFormat, content []byte) ObjectID {
	return hashType.ComputeHash(ObjectBlob, content)
}

type ErrInvalidSHA struct {
	SHA string
}

func (err ErrInvalidSHA) Error() string {
	return fmt.Sprintf("invalid sha: %s", err.SHA)
}
