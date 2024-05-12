// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"
)

// ProjectIssue saves relation from issue to a project
type ProjectIssue struct { //revive:disable-line:exported
	ID        int64 `xorm:"pk autoincr"`
	IssueID   int64 `xorm:"INDEX"`
	ProjectID int64 `xorm:"INDEX"`

	// ProjectBoardID should not be zero since 1.22. If it's zero, the issue will not be displayed on UI and it might result in errors.
	ProjectBoardID int64 `xorm:"INDEX"`

	// the sorting order on the board
	Sorting int64 `xorm:"NOT NULL DEFAULT 0"`
}

func init() {
	db.RegisterModel(new(ProjectIssue))
}

func deleteProjectIssuesByProjectID(ctx context.Context, projectID int64) error {
	_, err := db.GetEngine(ctx).Where("project_id=?", projectID).Delete(&ProjectIssue{})
	return err
}

// NumIssues return counter of all issues assigned to a project
func (p *Project) NumIssues(ctx context.Context) int {
	c, err := db.GetEngine(ctx).Table("project_issue").
		Where("project_id=?", p.ID).
		GroupBy("issue_id").
		Cols("issue_id").
		Count()
	if err != nil {
		log.Error("NumIssues: %v", err)
		return 0
	}
	return int(c)
}

// NumClosedIssues return counter of closed issues assigned to a project
func (p *Project) NumClosedIssues(ctx context.Context) int {
	c, err := db.GetEngine(ctx).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=?", p.ID, true).
		Cols("issue_id").
		Count()
	if err != nil {
		log.Error("NumClosedIssues: %v", err)
		return 0
	}
	return int(c)
}

// NumOpenIssues return counter of open issues assigned to a project
func (p *Project) NumOpenIssues(ctx context.Context) int {
	c, err := db.GetEngine(ctx).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=?", p.ID, false).
		Cols("issue_id").
		Count()
	if err != nil {
		log.Error("NumOpenIssues: %v", err)
		return 0
	}
	return int(c)
}

// MoveIssuesOnProjectBoard moves or keeps issues in a column and sorts them inside that column
func MoveIssuesOnProjectBoard(ctx context.Context, board *Board, sortedIssueIDs map[int64]int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		sess := db.GetEngine(ctx)
		issueIDs := util.ValuesOfMap(sortedIssueIDs)

		count, err := sess.Table(new(ProjectIssue)).Where("project_id=?", board.ProjectID).In("issue_id", issueIDs).Count()
		if err != nil {
			return err
		}
		if int(count) != len(sortedIssueIDs) {
			return fmt.Errorf("all issues have to be added to a project first")
		}

		for sorting, issueID := range sortedIssueIDs {
			_, err = sess.Exec("UPDATE `project_issue` SET project_board_id=?, sorting=? WHERE issue_id=?", board.ID, sorting, issueID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *Board) moveIssuesToAnotherColumn(ctx context.Context, newColumn *Board) error {
	if b.ProjectID != newColumn.ProjectID {
		return fmt.Errorf("columns have to be in the same project")
	}

	if b.ID == newColumn.ID {
		return nil
	}

	res := struct {
		MaxSorting int64
		IssueCount int64
	}{}
	if _, err := db.GetEngine(ctx).Select("max(sorting) as max_sorting, count(*) as issue_count").
		Table("project_issue").
		Where("project_id=?", newColumn.ProjectID).
		And("project_board_id=?", newColumn.ID).
		Get(&res); err != nil {
		return err
	}

	issues, err := b.GetIssues(ctx)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}

	nextSorting := util.Iif(res.IssueCount > 0, res.MaxSorting+1, 0)
	return db.WithTx(ctx, func(ctx context.Context) error {
		for i, issue := range issues {
			issue.ProjectBoardID = newColumn.ID
			issue.Sorting = nextSorting + int64(i)
			if _, err := db.GetEngine(ctx).ID(issue.ID).Cols("project_board_id", "sorting").Update(issue); err != nil {
				return err
			}
		}
		return nil
	})
}
