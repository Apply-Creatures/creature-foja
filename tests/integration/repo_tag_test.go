// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/models/db"
	git_model "code.gitea.io/gitea/models/git"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/services/release"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTagViewWithoutRelease(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	defer func() {
		releases, err := db.Find[repo_model.Release](db.DefaultContext, repo_model.FindReleasesOptions{
			IncludeTags: true,
			TagNames:    []string{"no-release"},
			RepoID:      repo.ID,
		})
		require.NoError(t, err)

		for _, release := range releases {
			_, err = db.DeleteByID[repo_model.Release](db.DefaultContext, release.ID)
			require.NoError(t, err)
		}
	}()

	err := release.CreateNewTag(git.DefaultContext, owner, repo, "master", "no-release", "release-less tag")
	require.NoError(t, err)

	// Test that the page loads
	req := NewRequestf(t, "GET", "/%s/releases/tag/no-release", repo.FullName())
	resp := MakeRequest(t, req, http.StatusOK)

	// Test that the tags sub-menu is active and has a counter
	htmlDoc := NewHTMLParser(t, resp.Body)
	tagsTab := htmlDoc.Find(".small-menu-items .active.item[href$='/tags']")
	assert.Contains(t, tagsTab.Text(), "4 tags")

	// Test that the release sub-menu isn't active
	releaseLink := htmlDoc.Find(".small-menu-items .item[href$='/releases']")
	assert.False(t, releaseLink.HasClass("active"))

	// Test that the title is displayed
	releaseTitle := strings.TrimSpace(htmlDoc.Find("h4.release-list-title > a").Text())
	assert.Equal(t, "no-release", releaseTitle)

	// Test that there is no "Stable" link
	htmlDoc.AssertElement(t, "h4.release-list-title > span.ui.green.label", false)
}

func TestCreateNewTagProtected(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	t.Run("Code", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		err := release.CreateNewTag(git.DefaultContext, owner, repo, "master", "t-first", "first tag")
		require.NoError(t, err)

		err = release.CreateNewTag(git.DefaultContext, owner, repo, "master", "v-2", "second tag")
		require.Error(t, err)
		assert.True(t, models.IsErrProtectedTagName(err))

		err = release.CreateNewTag(git.DefaultContext, owner, repo, "master", "v-1.1", "third tag")
		require.NoError(t, err)
	})

	t.Run("Git", func(t *testing.T) {
		onGiteaRun(t, func(t *testing.T, u *url.URL) {
			httpContext := NewAPITestContext(t, owner.Name, repo.Name)

			dstPath := t.TempDir()

			u.Path = httpContext.GitPath()
			u.User = url.UserPassword(owner.Name, userPassword)

			doGitClone(dstPath, u)(t)

			_, _, err := git.NewCommand(git.DefaultContext, "tag", "v-2").RunStdString(&git.RunOpts{Dir: dstPath})
			require.NoError(t, err)

			_, _, err = git.NewCommand(git.DefaultContext, "push", "--tags").RunStdString(&git.RunOpts{Dir: dstPath})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Tag v-2 is protected")
		})
	})

	// Cleanup
	releases, err := db.Find[repo_model.Release](db.DefaultContext, repo_model.FindReleasesOptions{
		IncludeTags: true,
		TagNames:    []string{"v-1", "v-1.1"},
		RepoID:      repo.ID,
	})
	require.NoError(t, err)

	for _, release := range releases {
		_, err = db.DeleteByID[repo_model.Release](db.DefaultContext, release.ID)
		require.NoError(t, err)
	}

	protectedTags, err := git_model.GetProtectedTags(db.DefaultContext, repo.ID)
	require.NoError(t, err)

	for _, protectedTag := range protectedTags {
		err = git_model.DeleteProtectedTag(db.DefaultContext, protectedTag)
		require.NoError(t, err)
	}
}
