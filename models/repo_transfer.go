// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package models

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/timeutil"
)

// RepoTransfer is used to manage repository transfers
type RepoTransfer struct {
	ID          int64 `xorm:"pk autoincr"`
	DoerID      int64
	Doer        *user_model.User `xorm:"-"`
	RecipientID int64
	Recipient   *user_model.User `xorm:"-"`
	RepoID      int64
	TeamIDs     []int64
	Teams       []*organization.Team `xorm:"-"`

	CreatedUnix timeutil.TimeStamp `xorm:"INDEX NOT NULL created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX NOT NULL updated"`
}

func init() {
	db.RegisterModel(new(RepoTransfer))
}

// LoadAttributes fetches the transfer recipient from the database
func (r *RepoTransfer) LoadAttributes(ctx context.Context) error {
	if r.Recipient == nil {
		u, err := user_model.GetUserByID(ctx, r.RecipientID)
		if err != nil {
			return err
		}

		r.Recipient = u
	}

	if r.Recipient.IsOrganization() && len(r.TeamIDs) != len(r.Teams) {
		for _, v := range r.TeamIDs {
			team, err := organization.GetTeamByID(ctx, v)
			if err != nil {
				return err
			}

			if team.OrgID != r.Recipient.ID {
				return fmt.Errorf("team %d belongs not to org %d", v, r.Recipient.ID)
			}

			r.Teams = append(r.Teams, team)
		}
	}

	if r.Doer == nil {
		u, err := user_model.GetUserByID(ctx, r.DoerID)
		if err != nil {
			return err
		}

		r.Doer = u
	}

	return nil
}

// CanUserAcceptTransfer checks if the user has the rights to accept/decline a repo transfer.
// For user, it checks if it's himself
// For organizations, it checks if the user is able to create repos
func (r *RepoTransfer) CanUserAcceptTransfer(ctx context.Context, u *user_model.User) bool {
	if err := r.LoadAttributes(ctx); err != nil {
		log.Error("LoadAttributes: %v", err)
		return false
	}

	if !r.Recipient.IsOrganization() {
		return r.RecipientID == u.ID
	}

	allowed, err := organization.CanCreateOrgRepo(ctx, r.RecipientID, u.ID)
	if err != nil {
		log.Error("CanCreateOrgRepo: %v", err)
		return false
	}

	return allowed
}

// GetPendingRepositoryTransfer fetches the most recent and ongoing transfer
// process for the repository
func GetPendingRepositoryTransfer(ctx context.Context, repo *repo_model.Repository) (*RepoTransfer, error) {
	transfer := new(RepoTransfer)

	has, err := db.GetEngine(ctx).Where("repo_id = ? ", repo.ID).Get(transfer)
	if err != nil {
		return nil, err
	}

	if !has {
		return nil, ErrNoPendingRepoTransfer{RepoID: repo.ID}
	}

	return transfer, nil
}

func DeleteRepositoryTransfer(ctx context.Context, repoID int64) error {
	_, err := db.GetEngine(ctx).Where("repo_id = ?", repoID).Delete(&RepoTransfer{})
	return err
}

// TestRepositoryReadyForTransfer make sure repo is ready to transfer
func TestRepositoryReadyForTransfer(status repo_model.RepositoryStatus) error {
	switch status {
	case repo_model.RepositoryBeingMigrated:
		return fmt.Errorf("repo is not ready, currently migrating")
	case repo_model.RepositoryPendingTransfer:
		return ErrRepoTransferInProgress{}
	}
	return nil
}

// CreatePendingRepositoryTransfer transfer a repo from one owner to a new one.
// it marks the repository transfer as "pending"
func CreatePendingRepositoryTransfer(ctx context.Context, doer, newOwner *user_model.User, repoID int64, teams []*organization.Team) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		repo, err := repo_model.GetRepositoryByID(ctx, repoID)
		if err != nil {
			return err
		}

		// Make sure repo is ready to transfer
		if err := TestRepositoryReadyForTransfer(repo.Status); err != nil {
			return err
		}

		repo.Status = repo_model.RepositoryPendingTransfer
		if err := repo_model.UpdateRepositoryCols(ctx, repo, "status"); err != nil {
			return err
		}

		// Check if new owner has repository with same name.
		if has, err := repo_model.IsRepositoryModelExist(ctx, newOwner, repo.Name); err != nil {
			return fmt.Errorf("IsRepositoryExist: %w", err)
		} else if has {
			return repo_model.ErrRepoAlreadyExist{
				Uname: newOwner.LowerName,
				Name:  repo.Name,
			}
		}

		transfer := &RepoTransfer{
			RepoID:      repo.ID,
			RecipientID: newOwner.ID,
			CreatedUnix: timeutil.TimeStampNow(),
			UpdatedUnix: timeutil.TimeStampNow(),
			DoerID:      doer.ID,
			TeamIDs:     make([]int64, 0, len(teams)),
		}

		for k := range teams {
			transfer.TeamIDs = append(transfer.TeamIDs, teams[k].ID)
		}

		return db.Insert(ctx, transfer)
	})
}

// GetPendingTransfers returns the pending transfers of recipient which were sent by by doer.
func GetPendingTransferIDs(ctx context.Context, reciepientID, doerID int64) ([]int64, error) {
	pendingTransferIDs := make([]int64, 0, 8)
	return pendingTransferIDs, db.GetEngine(ctx).Table("repo_transfer").
		Where("doer_id = ?", doerID).
		And("recipient_id = ?", reciepientID).
		Cols("id").
		Find(&pendingTransferIDs)
}
