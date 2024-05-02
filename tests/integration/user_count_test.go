// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	project_model "code.gitea.io/gitea/models/project"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/optional"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userCountTest struct {
	doer         *user_model.User
	user         *user_model.User
	session      *TestSession
	repoCount    int64
	projectCount int64
	memberCount  int64
	teamCount    int64
}

func (countTest *userCountTest) Init(t *testing.T, doerID, userID int64) {
	countTest.doer = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: doerID})
	countTest.user = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})
	countTest.session = loginUser(t, countTest.doer.Name)

	var err error

	countTest.repoCount, err = repo_model.CountRepository(db.DefaultContext, &repo_model.SearchRepoOptions{
		Actor:       countTest.doer,
		OwnerID:     countTest.user.ID,
		Private:     true,
		Collaborate: optional.Some(false),
	})
	require.NoError(t, err)

	var projectType project_model.Type
	if countTest.user.IsOrganization() {
		projectType = project_model.TypeOrganization
	} else {
		projectType = project_model.TypeIndividual
	}
	countTest.projectCount, err = db.Count[project_model.Project](db.DefaultContext, project_model.SearchOptions{
		OwnerID:  countTest.user.ID,
		IsClosed: optional.Some(false),
		Type:     projectType,
	})
	require.NoError(t, err)

	if !countTest.user.IsOrganization() {
		return
	}

	org := (*organization.Organization)(countTest.user)

	isMember, err := org.IsOrgMember(db.DefaultContext, countTest.doer.ID)
	require.NoError(t, err)

	countTest.memberCount, err = organization.CountOrgMembers(db.DefaultContext, &organization.FindOrgMembersOpts{
		OrgID:      org.ID,
		PublicOnly: !isMember,
	})
	require.NoError(t, err)

	teams, err := org.LoadTeams(db.DefaultContext)
	require.NoError(t, err)

	countTest.teamCount = int64(len(teams))
}

func (countTest *userCountTest) getCount(doc *goquery.Document, name string) (int64, error) {
	selection := doc.Find(fmt.Sprintf("[test-name=\"%s\"]", name))

	if selection.Length() != 1 {
		return 0, fmt.Errorf("%s was not found", name)
	}

	return strconv.ParseInt(selection.Text(), 10, 64)
}

func (countTest *userCountTest) TestPage(t *testing.T, page string, orgLink bool) {
	t.Run(page, func(t *testing.T) {
		var userLink string

		if orgLink {
			userLink = countTest.user.OrganisationLink()
		} else {
			userLink = countTest.user.HomeLink()
		}

		req := NewRequestf(t, "GET", "%s/%s", userLink, page)
		resp := countTest.session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		repoCount, err := countTest.getCount(htmlDoc.doc, "repository-count")
		require.NoError(t, err)
		assert.Equal(t, countTest.repoCount, repoCount)

		projectCount, err := countTest.getCount(htmlDoc.doc, "project-count")
		require.NoError(t, err)
		assert.Equal(t, countTest.projectCount, projectCount)

		if !countTest.user.IsOrganization() {
			return
		}

		memberCount, err := countTest.getCount(htmlDoc.doc, "member-count")
		require.NoError(t, err)
		assert.Equal(t, countTest.memberCount, memberCount)

		teamCount, err := countTest.getCount(htmlDoc.doc, "team-count")
		require.NoError(t, err)
		assert.Equal(t, countTest.teamCount, teamCount)
	})
}

func TestFrontendHeaderCountUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	countTest := new(userCountTest)
	countTest.Init(t, 2, 2)

	countTest.TestPage(t, "", false)
	countTest.TestPage(t, "?tab=repositories", false)
	countTest.TestPage(t, "-/projects", false)
	countTest.TestPage(t, "-/packages", false)
	countTest.TestPage(t, "?tab=activity", false)
	countTest.TestPage(t, "?tab=stars", false)
}

func TestFrontendHeaderCountOrg(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	countTest := new(userCountTest)
	countTest.Init(t, 15, 17)

	countTest.TestPage(t, "", false)
	countTest.TestPage(t, "-/projects", false)
	countTest.TestPage(t, "-/packages", false)
	countTest.TestPage(t, "members", true)
	countTest.TestPage(t, "teams", true)

	countTest.TestPage(t, "settings", true)
	countTest.TestPage(t, "settings/hooks", true)
	countTest.TestPage(t, "settings/labels", true)
	countTest.TestPage(t, "settings/applications", true)
	countTest.TestPage(t, "settings/packages", true)
	countTest.TestPage(t, "settings/actions/runners", true)
	countTest.TestPage(t, "settings/actions/secrets", true)
	countTest.TestPage(t, "settings/actions/variables", true)
	countTest.TestPage(t, "settings/blocked_users", true)
	countTest.TestPage(t, "settings/delete", true)
}
