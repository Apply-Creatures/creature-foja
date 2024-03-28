// Copyright 2016 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package activities

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"code.gitea.io/gitea/models/db"
	issues_model "code.gitea.io/gitea/models/issues"
	"code.gitea.io/gitea/models/organization"
	repo_model "code.gitea.io/gitea/models/repo"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

type (
	// NotificationStatus is the status of the notification (read or unread)
	NotificationStatus uint8
	// NotificationSource is the source of the notification (issue, PR, commit, etc)
	NotificationSource uint8
)

const (
	// NotificationStatusUnread represents an unread notification
	NotificationStatusUnread NotificationStatus = iota + 1
	// NotificationStatusRead represents a read notification
	NotificationStatusRead
	// NotificationStatusPinned represents a pinned notification
	NotificationStatusPinned
)

const (
	// NotificationSourceIssue is a notification of an issue
	NotificationSourceIssue NotificationSource = iota + 1
	// NotificationSourcePullRequest is a notification of a pull request
	NotificationSourcePullRequest
	// NotificationSourceCommit is a notification of a commit
	NotificationSourceCommit
	// NotificationSourceRepository is a notification for a repository
	NotificationSourceRepository
)

// Notification represents a notification
type Notification struct {
	ID     int64 `xorm:"pk autoincr"`
	UserID int64 `xorm:"INDEX NOT NULL"`
	RepoID int64 `xorm:"INDEX NOT NULL"`

	Status NotificationStatus `xorm:"SMALLINT INDEX NOT NULL"`
	Source NotificationSource `xorm:"SMALLINT INDEX NOT NULL"`

	IssueID   int64  `xorm:"INDEX NOT NULL"`
	CommitID  string `xorm:"INDEX"`
	CommentID int64

	UpdatedBy int64 `xorm:"INDEX NOT NULL"`

	Issue      *issues_model.Issue    `xorm:"-"`
	Repository *repo_model.Repository `xorm:"-"`
	Comment    *issues_model.Comment  `xorm:"-"`
	User       *user_model.User       `xorm:"-"`

	CreatedUnix timeutil.TimeStamp `xorm:"created INDEX NOT NULL"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated INDEX NOT NULL"`
}

func init() {
	db.RegisterModel(new(Notification))
}

// CreateRepoTransferNotification creates  notification for the user a repository was transferred to
func CreateRepoTransferNotification(ctx context.Context, doer, newOwner *user_model.User, repo *repo_model.Repository) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		var notify []*Notification

		if newOwner.IsOrganization() {
			users, err := organization.GetUsersWhoCanCreateOrgRepo(ctx, newOwner.ID)
			if err != nil || len(users) == 0 {
				return err
			}
			for i := range users {
				notify = append(notify, &Notification{
					UserID:    i,
					RepoID:    repo.ID,
					Status:    NotificationStatusUnread,
					UpdatedBy: doer.ID,
					Source:    NotificationSourceRepository,
				})
			}
		} else {
			notify = []*Notification{{
				UserID:    newOwner.ID,
				RepoID:    repo.ID,
				Status:    NotificationStatusUnread,
				UpdatedBy: doer.ID,
				Source:    NotificationSourceRepository,
			}}
		}

		return db.Insert(ctx, notify)
	})
}

func createIssueNotification(ctx context.Context, userID int64, issue *issues_model.Issue, commentID, updatedByID int64) error {
	notification := &Notification{
		UserID:    userID,
		RepoID:    issue.RepoID,
		Status:    NotificationStatusUnread,
		IssueID:   issue.ID,
		CommentID: commentID,
		UpdatedBy: updatedByID,
	}

	if issue.IsPull {
		notification.Source = NotificationSourcePullRequest
	} else {
		notification.Source = NotificationSourceIssue
	}

	return db.Insert(ctx, notification)
}

func updateIssueNotification(ctx context.Context, userID, issueID, commentID, updatedByID int64) error {
	notification, err := GetIssueNotification(ctx, userID, issueID)
	if err != nil {
		return err
	}

	// NOTICE: Only update comment id when the before notification on this issue is read, otherwise you may miss some old comments.
	// But we need update update_by so that the notification will be reorder
	var cols []string
	if notification.Status == NotificationStatusRead {
		notification.Status = NotificationStatusUnread
		notification.CommentID = commentID
		cols = []string{"status", "update_by", "comment_id"}
	} else {
		notification.UpdatedBy = updatedByID
		cols = []string{"update_by"}
	}

	_, err = db.GetEngine(ctx).ID(notification.ID).Cols(cols...).Update(notification)
	return err
}

// GetIssueNotification return the notification about an issue
func GetIssueNotification(ctx context.Context, userID, issueID int64) (*Notification, error) {
	notification := new(Notification)
	_, err := db.GetEngine(ctx).
		Where("user_id = ?", userID).
		And("issue_id = ?", issueID).
		Get(notification)
	return notification, err
}

// LoadAttributes load Repo Issue User and Comment if not loaded
func (n *Notification) LoadAttributes(ctx context.Context) (err error) {
	if err = n.loadRepo(ctx); err != nil {
		return err
	}
	if err = n.loadIssue(ctx); err != nil {
		return err
	}
	if err = n.loadUser(ctx); err != nil {
		return err
	}
	if err = n.loadComment(ctx); err != nil {
		return err
	}
	return err
}

func (n *Notification) loadRepo(ctx context.Context) (err error) {
	if n.Repository == nil {
		n.Repository, err = repo_model.GetRepositoryByID(ctx, n.RepoID)
		if err != nil {
			return fmt.Errorf("getRepositoryByID [%d]: %w", n.RepoID, err)
		}
	}
	return nil
}

func (n *Notification) loadIssue(ctx context.Context) (err error) {
	if n.Issue == nil && n.IssueID != 0 {
		n.Issue, err = issues_model.GetIssueByID(ctx, n.IssueID)
		if err != nil {
			return fmt.Errorf("getIssueByID [%d]: %w", n.IssueID, err)
		}
		return n.Issue.LoadAttributes(ctx)
	}
	return nil
}

func (n *Notification) loadComment(ctx context.Context) (err error) {
	if n.Comment == nil && n.CommentID != 0 {
		n.Comment, err = issues_model.GetCommentByID(ctx, n.CommentID)
		if err != nil {
			if issues_model.IsErrCommentNotExist(err) {
				return issues_model.ErrCommentNotExist{
					ID:      n.CommentID,
					IssueID: n.IssueID,
				}
			}
			return err
		}
	}
	return nil
}

func (n *Notification) loadUser(ctx context.Context) (err error) {
	if n.User == nil {
		n.User, err = user_model.GetUserByID(ctx, n.UserID)
		if err != nil {
			return fmt.Errorf("getUserByID [%d]: %w", n.UserID, err)
		}
	}
	return nil
}

// GetRepo returns the repo of the notification
func (n *Notification) GetRepo(ctx context.Context) (*repo_model.Repository, error) {
	return n.Repository, n.loadRepo(ctx)
}

// GetIssue returns the issue of the notification
func (n *Notification) GetIssue(ctx context.Context) (*issues_model.Issue, error) {
	return n.Issue, n.loadIssue(ctx)
}

// HTMLURL formats a URL-string to the notification
func (n *Notification) HTMLURL(ctx context.Context) string {
	switch n.Source {
	case NotificationSourceIssue, NotificationSourcePullRequest:
		if n.Comment != nil {
			return n.Comment.HTMLURL(ctx)
		}
		return n.Issue.HTMLURL()
	case NotificationSourceCommit:
		return n.Repository.HTMLURL() + "/commit/" + url.PathEscape(n.CommitID)
	case NotificationSourceRepository:
		return n.Repository.HTMLURL()
	}
	return ""
}

// Link formats a relative URL-string to the notification
func (n *Notification) Link(ctx context.Context) string {
	switch n.Source {
	case NotificationSourceIssue, NotificationSourcePullRequest:
		if n.Comment != nil {
			return n.Comment.Link(ctx)
		}
		return n.Issue.Link()
	case NotificationSourceCommit:
		return n.Repository.Link() + "/commit/" + url.PathEscape(n.CommitID)
	case NotificationSourceRepository:
		return n.Repository.Link()
	}
	return ""
}

// APIURL formats a URL-string to the notification
func (n *Notification) APIURL() string {
	return setting.AppURL + "api/v1/notifications/threads/" + strconv.FormatInt(n.ID, 10)
}

func notificationExists(notifications []*Notification, issueID, userID int64) bool {
	for _, notification := range notifications {
		if notification.IssueID == issueID && notification.UserID == userID {
			return true
		}
	}

	return false
}

// UserIDCount is a simple coalition of UserID and Count
type UserIDCount struct {
	UserID int64
	Count  int64
}

// GetUIDsAndNotificationCounts between the two provided times
func GetUIDsAndNotificationCounts(ctx context.Context, since, until timeutil.TimeStamp) ([]UserIDCount, error) {
	sql := `SELECT user_id, count(*) AS count FROM notification ` +
		`WHERE user_id IN (SELECT user_id FROM notification WHERE updated_unix >= ? AND ` +
		`updated_unix < ?) AND status = ? GROUP BY user_id`
	var res []UserIDCount
	return res, db.GetEngine(ctx).SQL(sql, since, until, NotificationStatusUnread).Find(&res)
}

// SetIssueReadBy sets issue to be read by given user.
func SetIssueReadBy(ctx context.Context, issueID, userID int64) error {
	if err := issues_model.UpdateIssueUserByRead(ctx, userID, issueID); err != nil {
		return err
	}

	return setIssueNotificationStatusReadIfUnread(ctx, userID, issueID)
}

func setIssueNotificationStatusReadIfUnread(ctx context.Context, userID, issueID int64) error {
	notification, err := GetIssueNotification(ctx, userID, issueID)
	// ignore if not exists
	if err != nil {
		return nil
	}

	if notification.Status != NotificationStatusUnread {
		return nil
	}

	notification.Status = NotificationStatusRead

	_, err = db.GetEngine(ctx).ID(notification.ID).Cols("status").Update(notification)
	return err
}

// SetRepoReadBy sets repo to be visited by given user.
func SetRepoReadBy(ctx context.Context, userID, repoID int64) error {
	_, err := db.GetEngine(ctx).Where(builder.Eq{
		"user_id": userID,
		"status":  NotificationStatusUnread,
		"source":  NotificationSourceRepository,
		"repo_id": repoID,
	}).Cols("status").Update(&Notification{Status: NotificationStatusRead})
	return err
}

// SetNotificationStatus change the notification status
func SetNotificationStatus(ctx context.Context, notificationID int64, user *user_model.User, status NotificationStatus) (*Notification, error) {
	notification, err := GetNotificationByID(ctx, notificationID)
	if err != nil {
		return notification, err
	}

	if notification.UserID != user.ID {
		return nil, fmt.Errorf("Can't change notification of another user: %d, %d", notification.UserID, user.ID)
	}

	notification.Status = status

	_, err = db.GetEngine(ctx).ID(notificationID).Update(notification)
	return notification, err
}

// GetNotificationByID return notification by ID
func GetNotificationByID(ctx context.Context, notificationID int64) (*Notification, error) {
	notification := new(Notification)
	ok, err := db.GetEngine(ctx).
		Where("id = ?", notificationID).
		Get(notification)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, db.ErrNotExist{Resource: "notification", ID: notificationID}
	}

	return notification, nil
}

// UpdateNotificationStatuses updates the statuses of all of a user's notifications that are of the currentStatus type to the desiredStatus
func UpdateNotificationStatuses(ctx context.Context, user *user_model.User, currentStatus, desiredStatus NotificationStatus) error {
	n := &Notification{Status: desiredStatus, UpdatedBy: user.ID}
	_, err := db.GetEngine(ctx).
		Where("user_id = ? AND status = ?", user.ID, currentStatus).
		Cols("status", "updated_by", "updated_unix").
		Update(n)
	return err
}

// LoadIssuePullRequests loads all issues' pull requests if possible
func (nl NotificationList) LoadIssuePullRequests(ctx context.Context) error {
	issues := make(map[int64]*issues_model.Issue, len(nl))
	for _, notification := range nl {
		if notification.Issue != nil && notification.Issue.IsPull && notification.Issue.PullRequest == nil {
			issues[notification.Issue.ID] = notification.Issue
		}
	}

	if len(issues) == 0 {
		return nil
	}

	pulls, err := issues_model.GetPullRequestByIssueIDs(ctx, util.KeysOfMap(issues))
	if err != nil {
		return err
	}

	for _, pull := range pulls {
		if issue := issues[pull.IssueID]; issue != nil {
			issue.PullRequest = pull
			issue.PullRequest.Issue = issue
		}
	}

	return nil
}
