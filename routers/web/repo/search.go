// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"net/http"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/base"
	code_indexer "code.gitea.io/gitea/modules/indexer/code"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/services/context"
	"code.gitea.io/gitea/services/repository/files"
)

const tplSearch base.TplName = "repo/search"

// Search render repository search page
func Search(ctx *context.Context) {
	language := ctx.FormTrim("l")
	keyword := ctx.FormTrim("q")

	isFuzzy := ctx.FormOptionalBool("fuzzy").ValueOrDefault(true)

	ctx.Data["Keyword"] = keyword
	ctx.Data["Language"] = language
	ctx.Data["IsFuzzy"] = isFuzzy
	ctx.Data["PageIsViewCode"] = true

	if keyword == "" {
		ctx.HTML(http.StatusOK, tplSearch)
		return
	}

	ctx.Data["Repo"] = ctx.Repo.Repository

	page := ctx.FormInt("page")
	if page <= 0 {
		page = 1
	}

	if setting.Indexer.RepoIndexerEnabled {
		ctx.Data["CodeIndexerDisabled"] = false

		total, searchResults, searchResultLanguages, err := code_indexer.PerformSearch(ctx, &code_indexer.SearchOptions{
			RepoIDs:        []int64{ctx.Repo.Repository.ID},
			Keyword:        keyword,
			IsKeywordFuzzy: isFuzzy,
			Language:       language,
			Paginator: &db.ListOptions{
				Page:     page,
				PageSize: setting.UI.RepoSearchPagingNum,
			},
		})
		if err != nil {
			if code_indexer.IsAvailable(ctx) {
				ctx.ServerError("SearchResults", err)
				return
			}
			ctx.Data["CodeIndexerUnavailable"] = true
		} else {
			ctx.Data["CodeIndexerUnavailable"] = !code_indexer.IsAvailable(ctx)
		}

		ctx.Data["SearchResults"] = searchResults
		ctx.Data["SearchResultLanguages"] = searchResultLanguages

		pager := context.NewPagination(total, setting.UI.RepoSearchPagingNum, page, 5)
		pager.SetDefaultParams(ctx)
		pager.AddParam(ctx, "l", "Language")
		ctx.Data["Page"] = pager
	} else {
		data, err := files.NewRepoGrep(ctx, ctx.Repo.Repository, keyword)
		if err != nil {
			ctx.ServerError("NewRepoGrep", err)
			return
		}

		ctx.Data["CodeIndexerDisabled"] = true
		ctx.Data["SearchResults"] = data

		pager := context.NewPagination(len(data), setting.UI.RepoSearchPagingNum, page, 5)
		pager.SetDefaultParams(ctx)
		ctx.Data["Page"] = pager
	}

	ctx.HTML(http.StatusOK, tplSearch)
}
