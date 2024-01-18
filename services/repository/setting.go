// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"context"

	"code.gitea.io/gitea/models/db"
	repo_model "code.gitea.io/gitea/models/repo"
	"code.gitea.io/gitea/models/unit"
)

// UpdateRepositoryUnits updates a repository's units
func UpdateRepositoryUnits(ctx context.Context, repo *repo_model.Repository, units []repo_model.RepoUnit, deleteUnitTypes []unit.Type) (err error) {
	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	// Delete existing settings of units before adding again
	for _, u := range units {
		deleteUnitTypes = append(deleteUnitTypes, u.Type)
	}

	if _, err = db.GetEngine(ctx).Where("repo_id = ?", repo.ID).In("type", deleteUnitTypes).Delete(new(repo_model.RepoUnit)); err != nil {
		return err
	}

	if len(units) > 0 {
		if err = db.Insert(ctx, units); err != nil {
			return err
		}
	}

	return committer.Commit()
}
