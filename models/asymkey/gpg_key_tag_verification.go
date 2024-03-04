// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"context"

	"code.gitea.io/gitea/modules/git"
)

func ParseTagWithSignature(ctx context.Context, gitRepo *git.Repository, t *git.Tag) *ObjectVerification {
	o := tagToGitObject(t, gitRepo)
	return ParseObjectWithSignature(ctx, &o)
}
