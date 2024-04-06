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

func resultFilenames(t testing.TB, doc *HTMLDoc) []string {
	filenameSelections := doc.doc.Find(".repository.search").Find(".repo-search-result").Find(".header").Find("span.file")
	result := make([]string, filenameSelections.Length())
	filenameSelections.Each(func(i int, selection *goquery.Selection) {
		result[i] = selection.Text()
	})
	return result
}

func TestSearchRepoIndexer(t *testing.T) {
	testSearchRepo(t, true)
}

func TestSearchRepoNoIndexer(t *testing.T) {
	testSearchRepo(t, false)
}

func testSearchRepo(t *testing.T, indexer bool) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Indexer.RepoIndexerEnabled, indexer)()

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "repo1")
	assert.NoError(t, err)

	if indexer {
		code_indexer.UpdateRepoIndexer(repo)
	}

	testSearch(t, "/user2/repo1/search?q=Description&page=1", []string{"README.md"})

	defer test.MockVariableValue(&setting.Indexer.IncludePatterns, setting.IndexerGlobFromString("**.txt"))()
	defer test.MockVariableValue(&setting.Indexer.ExcludePatterns, setting.IndexerGlobFromString("**/y/**"))()

	repo, err = repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "glob")
	assert.NoError(t, err)

	if indexer {
		code_indexer.UpdateRepoIndexer(repo)
	}

	testSearch(t, "/user2/glob/search?q=loren&page=1", []string{"a.txt"})
	testSearch(t, "/user2/glob/search?q=loren&page=1&fuzzy=false", []string{"a.txt"})

	if indexer {
		// fuzzy search: matches both file3 (x/b.txt) and file1 (a.txt)
		// when indexer is enabled
		testSearch(t, "/user2/glob/search?q=file3&page=1", []string{"x/b.txt", "a.txt"})
		testSearch(t, "/user2/glob/search?q=file4&page=1", []string{"x/b.txt", "a.txt"})
		testSearch(t, "/user2/glob/search?q=file5&page=1", []string{"x/b.txt", "a.txt"})
	} else {
		// fuzzy search: OR of all the keywords
		// when indexer is disabled
		testSearch(t, "/user2/glob/search?q=file3+file1&page=1", []string{"a.txt", "x/b.txt"})
		testSearch(t, "/user2/glob/search?q=file4&page=1", []string{})
		testSearch(t, "/user2/glob/search?q=file5&page=1", []string{})
	}

	testSearch(t, "/user2/glob/search?q=file3&page=1&fuzzy=false", []string{"x/b.txt"})
	testSearch(t, "/user2/glob/search?q=file4&page=1&fuzzy=false", []string{})
	testSearch(t, "/user2/glob/search?q=file5&page=1&fuzzy=false", []string{})
}

func testSearch(t *testing.T, url string, expected []string) {
	req := NewRequest(t, "GET", url)
	resp := MakeRequest(t, req, http.StatusOK)

	filenames := resultFilenames(t, NewHTMLParser(t, resp.Body))
	assert.EqualValues(t, expected, filenames)
}
