// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	code_indexer "code.gitea.io/gitea/modules/indexer/code"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func resultFilenames(t testing.TB, doc *goquery.Selection) []string {
	filenameSelections := doc.Find(".header").Find("span.file")
	result := make([]string, filenameSelections.Length())
	filenameSelections.Each(func(i int, selection *goquery.Selection) {
		result[i] = selection.Text()
	})
	return result
}

func checkResultLinks(t *testing.T, substr string, doc *goquery.Selection) {
	t.Helper()
	linkSelections := doc.Find("a[href]")
	linkSelections.Each(func(i int, selection *goquery.Selection) {
		assert.Contains(t, selection.AttrOr("href", ""), substr)
	})
}

func testSearchRepo(t *testing.T, useExternalIndexer bool) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Indexer.RepoIndexerEnabled, useExternalIndexer)()

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "repo1")
	assert.NoError(t, err)

	gitReference := "/branch/" + repo.DefaultBranch

	if useExternalIndexer {
		gitReference = "/commit/"
		executeIndexer(t, repo, code_indexer.UpdateRepoIndexer)
	}

	testSearch(t, "/user2/repo1/search?q=Description&page=1", gitReference, []string{"README.md"})

	if useExternalIndexer {
		setting.Indexer.IncludePatterns = setting.IndexerGlobFromString("**.txt")
		setting.Indexer.ExcludePatterns = setting.IndexerGlobFromString("**/y/**")

		repo, err = repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "glob")
		assert.NoError(t, err)

		executeIndexer(t, repo, code_indexer.UpdateRepoIndexer)

		testSearch(t, "/user2/glob/search?q=loren&page=1", gitReference, []string{"a.txt"})
		testSearch(t, "/user2/glob/search?q=file3&page=1", gitReference, []string{"x/b.txt"})
		testSearch(t, "/user2/glob/search?q=file4&page=1", gitReference, []string{})
		testSearch(t, "/user2/glob/search?q=file5&page=1", gitReference, []string{})
	}
}

func TestIndexerSearchRepo(t *testing.T) {
	testSearchRepo(t, true)
}

func TestNoIndexerSearchRepo(t *testing.T) {
	testSearchRepo(t, false)
}

func testSearch(t *testing.T, url, gitRef string, expected []string) {
	req := NewRequest(t, "GET", url)
	resp := MakeRequest(t, req, http.StatusOK)

	doc := NewHTMLParser(t, resp.Body).doc.
		Find(".repository.search").
		Find(".repo-search-result")

	filenames := resultFilenames(t, doc)
	assert.EqualValues(t, expected, filenames)

	checkResultLinks(t, gitRef, doc)
}

func executeIndexer(t *testing.T, repo *repo_model.Repository, op func(*repo_model.Repository)) {
	op(repo)
}
