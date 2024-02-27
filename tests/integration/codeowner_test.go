// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	issues_model "code.gitea.io/gitea/models/issues"
	unit_model "code.gitea.io/gitea/models/unit"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"
	"github.com/stretchr/testify/assert"
)

func TestCodeOwner(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		// Create the repo.
		repo, _, f := CreateDeclarativeRepo(t, user2, "",
			[]unit_model.Type{unit_model.TypePullRequests}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation:     "create",
					TreePath:      "CODEOWNERS",
					ContentReader: strings.NewReader("README.md @user5\ntest-file @user4"),
				},
			},
		)
		defer f()

		dstPath := t.TempDir()
		r := fmt.Sprintf("%suser2/%s.git", u.String(), repo.Name)
		u, _ = url.Parse(r)
		u.User = url.UserPassword("user2", userPassword)
		assert.NoError(t, git.CloneWithArgs(context.Background(), nil, u.String(), dstPath, git.CloneRepoOptions{}))

		t.Run("Normal", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			err := os.WriteFile(path.Join(dstPath, "README.md"), []byte("## test content"), 0o666)
			assert.NoError(t, err)

			err = git.AddChanges(dstPath, true)
			assert.NoError(t, err)

			err = git.CommitChanges(dstPath, git.CommitChangesOptions{
				Committer: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Author: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Message: "Add README.",
			})
			assert.NoError(t, err)

			err = git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/main", "-o", "topic=codeowner-normal").Run(&git.RunOpts{Dir: dstPath})
			assert.NoError(t, err)

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "user2/codeowner-normal"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
		})

		t.Run("Out of date", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			// Push the changes made from the previous subtest.
			assert.NoError(t, git.NewCommand(git.DefaultContext, "push", "origin").Run(&git.RunOpts{Dir: dstPath}))

			// Reset the tree to the previous commit.
			assert.NoError(t, git.NewCommand(git.DefaultContext, "reset", "--hard", "HEAD~1").Run(&git.RunOpts{Dir: dstPath}))

			err := os.WriteFile(path.Join(dstPath, "test-file"), []byte("## test content"), 0o666)
			assert.NoError(t, err)

			err = git.AddChanges(dstPath, true)
			assert.NoError(t, err)

			err = git.CommitChanges(dstPath, git.CommitChangesOptions{
				Committer: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Author: &git.Signature{
					Email: "user2@example.com",
					Name:  "user2",
					When:  time.Now(),
				},
				Message: "Add test-file.",
			})
			assert.NoError(t, err)

			err = git.NewCommand(git.DefaultContext, "push", "origin", "HEAD:refs/for/main", "-o", "topic=codeowner-out-of-date").Run(&git.RunOpts{Dir: dstPath})
			assert.NoError(t, err)

			pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{BaseRepoID: repo.ID, HeadBranch: "user2/codeowner-out-of-date"})
			unittest.AssertExistsIf(t, true, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 4})
			unittest.AssertExistsIf(t, false, &issues_model.Review{IssueID: pr.IssueID, Type: issues_model.ReviewTypeRequest, ReviewerID: 5})
		})
	})
}
