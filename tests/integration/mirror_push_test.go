// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/gitrepo"
	"code.gitea.io/gitea/modules/setting"
	gitea_context "code.gitea.io/gitea/services/context"
	doctor "code.gitea.io/gitea/services/doctor"
	"code.gitea.io/gitea/services/migrations"
	mirror_service "code.gitea.io/gitea/services/mirror"
	repo_service "code.gitea.io/gitea/services/repository"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMirrorPush(t *testing.T) {
	onGiteaRun(t, testMirrorPush)
}

func testMirrorPush(t *testing.T, u *url.URL) {
	defer tests.PrepareTestEnv(t)()

	setting.Migrations.AllowLocalNetworks = true
	require.NoError(t, migrations.Init())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	srcRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	mirrorRepo, err := repo_service.CreateRepositoryDirectly(db.DefaultContext, user, user, repo_service.CreateRepoOptions{
		Name: "test-push-mirror",
	})
	require.NoError(t, err)

	ctx := NewAPITestContext(t, user.LowerName, srcRepo.Name)

	doCreatePushMirror(ctx, fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(ctx.Username), url.PathEscape(mirrorRepo.Name)), user.LowerName, userPassword)(t)
	doCreatePushMirror(ctx, fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(ctx.Username), url.PathEscape("does-not-matter")), user.LowerName, userPassword)(t)

	mirrors, _, err := repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, mirrors, 2)

	ok := mirror_service.SyncPushMirror(context.Background(), mirrors[0].ID)
	assert.True(t, ok)

	srcGitRepo, err := gitrepo.OpenRepository(git.DefaultContext, srcRepo)
	require.NoError(t, err)
	defer srcGitRepo.Close()

	srcCommit, err := srcGitRepo.GetBranchCommit("master")
	require.NoError(t, err)

	mirrorGitRepo, err := gitrepo.OpenRepository(git.DefaultContext, mirrorRepo)
	require.NoError(t, err)
	defer mirrorGitRepo.Close()

	mirrorCommit, err := mirrorGitRepo.GetBranchCommit("master")
	require.NoError(t, err)

	assert.Equal(t, srcCommit.ID, mirrorCommit.ID)

	// Test that we can "repair" push mirrors where the remote doesn't exist in git's state.
	// To do that, we artificially remove the remote...
	cmd := git.NewCommand(db.DefaultContext, "remote", "rm").AddDynamicArguments(mirrors[0].RemoteName)
	_, _, err = cmd.RunStdString(&git.RunOpts{Dir: srcRepo.RepoPath()})
	require.NoError(t, err)

	// ...then ensure that trying to get its remote address fails
	_, err = repo_model.GetPushMirrorRemoteAddress(srcRepo.OwnerName, srcRepo.Name, mirrors[0].RemoteName)
	require.Error(t, err)

	// ...and that we can fix it.
	err = doctor.FixPushMirrorsWithoutGitRemote(db.DefaultContext, nil, true)
	require.NoError(t, err)

	// ...and after fixing, we only have one remote
	mirrors, _, err = repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, mirrors, 1)

	// ...one we can get the address of, and it's not the one we removed
	remoteAddress, err := repo_model.GetPushMirrorRemoteAddress(srcRepo.OwnerName, srcRepo.Name, mirrors[0].RemoteName)
	require.NoError(t, err)
	assert.Contains(t, remoteAddress, "does-not-matter")

	// Cleanup
	doRemovePushMirror(ctx, fmt.Sprintf("%s%s/%s", u.String(), url.PathEscape(ctx.Username), url.PathEscape(mirrorRepo.Name)), user.LowerName, userPassword, int(mirrors[0].ID))(t)
	mirrors, _, err = repo_model.GetPushMirrorsByRepoID(db.DefaultContext, srcRepo.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Empty(t, mirrors)
}

func doCreatePushMirror(ctx APITestContext, address, username, password string) func(t *testing.T) {
	return func(t *testing.T) {
		csrf := GetCSRF(t, ctx.Session, fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)))

		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)), map[string]string{
			"_csrf":                csrf,
			"action":               "push-mirror-add",
			"push_mirror_address":  address,
			"push_mirror_username": username,
			"push_mirror_password": password,
			"push_mirror_interval": "0",
		})
		ctx.Session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := ctx.Session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "success")
	}
}

func doRemovePushMirror(ctx APITestContext, address, username, password string, pushMirrorID int) func(t *testing.T) {
	return func(t *testing.T) {
		csrf := GetCSRF(t, ctx.Session, fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)))

		req := NewRequestWithValues(t, "POST", fmt.Sprintf("/%s/%s/settings", url.PathEscape(ctx.Username), url.PathEscape(ctx.Reponame)), map[string]string{
			"_csrf":                csrf,
			"action":               "push-mirror-remove",
			"push_mirror_id":       strconv.Itoa(pushMirrorID),
			"push_mirror_address":  address,
			"push_mirror_username": username,
			"push_mirror_password": password,
			"push_mirror_interval": "0",
		})
		ctx.Session.MakeRequest(t, req, http.StatusSeeOther)

		flashCookie := ctx.Session.GetCookie(gitea_context.CookieNameFlash)
		assert.NotNil(t, flashCookie)
		assert.Contains(t, flashCookie.Value, "success")
	}
}
