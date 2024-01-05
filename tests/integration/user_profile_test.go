// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/git"
	repo_service "code.gitea.io/gitea/services/repository"
	files_service "code.gitea.io/gitea/services/repository/files"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
)

func createProfileRepo(t *testing.T, readmeName string) func() {
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	// Create a new repository
	repo, err := repo_service.CreateRepository(db.DefaultContext, user2, user2, repo_service.CreateRepoOptions{
		Name:          ".profile",
		DefaultBranch: "main",
		IsPrivate:     false,
		AutoInit:      true,
		Readme:        "Default",
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, repo)

	deleteInitialReadmeResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user2,
		&files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{
				{
					Operation: "delete",
					TreePath:  "README.md",
				},
			},
			Message: "Delete the initial readme",
			Author: &files_service.IdentityOptions{
				Name:  user2.Name,
				Email: user2.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  user2.Name,
				Email: user2.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
	assert.NoError(t, err)
	assert.NotEmpty(t, deleteInitialReadmeResp)

	if readmeName != "" {
		addReadmeResp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user2,
			&files_service.ChangeRepoFilesOptions{
				Files: []*files_service.ChangeRepoFile{
					{
						Operation:     "create",
						TreePath:      readmeName,
						ContentReader: strings.NewReader("# Hi!\n"),
					},
				},
				Message: "Add a readme",
				Author: &files_service.IdentityOptions{
					Name:  user2.Name,
					Email: user2.Email,
				},
				Committer: &files_service.IdentityOptions{
					Name:  user2.Name,
					Email: user2.Email,
				},
				Dates: &files_service.CommitDateOptions{
					Author:    time.Now(),
					Committer: time.Now(),
				},
			})

		assert.NoError(t, err)
		assert.NotEmpty(t, addReadmeResp)
	}

	return func() {
		repo_service.DeleteRepository(db.DefaultContext, user2, repo, false)
	}
}

func TestUserProfile(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		checkReadme := func(t *testing.T, title, readmeFilename string, expectedCount int) {
			t.Run(title, func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()
				defer createProfileRepo(t, readmeFilename)()

				req := NewRequest(t, "GET", "/user2")
				resp := MakeRequest(t, req, http.StatusOK)

				doc := NewHTMLParser(t, resp.Body)
				readmeCount := doc.Find("#readme_profile").Length()

				assert.Equal(t, expectedCount, readmeCount)
			})
		}

		checkReadme(t, "No readme", "", 0)
		checkReadme(t, "README.md", "README.md", 1)
		checkReadme(t, "readme.md", "readme.md", 1)
		checkReadme(t, "ReadMe.mD", "ReadMe.mD", 1)
		checkReadme(t, "readme.org does not render", "README.org", 0)
	})
}
