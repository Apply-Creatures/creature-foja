// Copyright 2024 The Forgejo Authors
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"testing"
	"time"

	issues_model "code.gitea.io/gitea/models/issues"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	repo_module "code.gitea.io/gitea/modules/repository"
	"code.gitea.io/gitea/modules/test"
	pull_service "code.gitea.io/gitea/services/pull"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestSynchronized(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// unmerged pull request of user2/repo1 from branch2 to master
	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
	// tip of tests/gitea-repositories-meta/user2/repo1 branch2
	pull.HeadCommitID = "985f0301dba5e7b34be866819cd15ad3d8f508ee"

	require.Equal(t, pull.HeadRepoID, pull.BaseRepoID)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: pull.HeadRepoID})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	t.Run("AddTestPullRequestTask", func(t *testing.T) {
		logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
		logChecker.Filter("Updating PR").StopMark("TestPullRequest ")
		defer cleanup()

		opt := &repo_module.PushUpdateOptions{
			PusherID:     owner.ID,
			PusherName:   owner.Name,
			RepoUserName: owner.Name,
			RepoName:     repo.Name,
			RefFullName:  git.RefName("refs/heads/branch2"),
			OldCommitID:  pull.HeadCommitID,
			NewCommitID:  pull.HeadCommitID,
		}
		require.NoError(t, repo_service.PushUpdate(opt))
		logFiltered, logStopped := logChecker.Check(5 * time.Second)
		assert.True(t, logStopped)
		assert.True(t, logFiltered[0])
	})

	for _, testCase := range []struct {
		name     string
		maxPR    int64
		expected bool
	}{
		{
			name:     "TestPullRequest process PR",
			maxPR:    pull.Index,
			expected: true,
		},
		{
			name:     "TestPullRequest skip PR",
			maxPR:    pull.Index - 1,
			expected: false,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			logChecker, cleanup := test.NewLogChecker(log.DEFAULT, log.TRACE)
			logChecker.Filter("Updating PR").StopMark("TestPullRequest ")
			defer cleanup()

			pull_service.TestPullRequest(context.Background(), owner, repo.ID, testCase.maxPR, "branch2", true, pull.HeadCommitID, pull.HeadCommitID)
			logFiltered, logStopped := logChecker.Check(5 * time.Second)
			assert.True(t, logStopped)
			assert.Equal(t, testCase.expected, logFiltered[0])
		})
	}
}
