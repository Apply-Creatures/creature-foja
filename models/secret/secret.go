// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package secret

import (
	"context"
	"errors"
	"fmt"
	"strings"

	actions_model "code.gitea.io/gitea/models/actions"
	"code.gitea.io/gitea/models/db"
	actions_module "code.gitea.io/gitea/modules/actions"
	"code.gitea.io/gitea/modules/log"
	secret_module "code.gitea.io/gitea/modules/secret"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/timeutil"
	"code.gitea.io/gitea/modules/util"

	"xorm.io/builder"
)

// Secret represents a secret
type Secret struct {
	ID          int64
	OwnerID     int64              `xorm:"INDEX UNIQUE(owner_repo_name) NOT NULL"`
	RepoID      int64              `xorm:"INDEX UNIQUE(owner_repo_name) NOT NULL DEFAULT 0"`
	Name        string             `xorm:"UNIQUE(owner_repo_name) NOT NULL"`
	Data        string             `xorm:"LONGTEXT"` // encrypted data
	CreatedUnix timeutil.TimeStamp `xorm:"created NOT NULL"`
}

// ErrSecretNotFound represents a "secret not found" error.
type ErrSecretNotFound struct {
	Name string
}

func (err ErrSecretNotFound) Error() string {
	return fmt.Sprintf("secret was not found [name: %s]", err.Name)
}

func (err ErrSecretNotFound) Unwrap() error {
	return util.ErrNotExist
}

// InsertEncryptedSecret Creates, encrypts, and validates a new secret with yet unencrypted data and insert into database
func InsertEncryptedSecret(ctx context.Context, ownerID, repoID int64, name, data string) (*Secret, error) {
	encrypted, err := secret_module.EncryptSecret(setting.SecretKey, data)
	if err != nil {
		return nil, err
	}
	secret := &Secret{
		OwnerID: ownerID,
		RepoID:  repoID,
		Name:    strings.ToUpper(name),
		Data:    encrypted,
	}
	if err := secret.Validate(); err != nil {
		return secret, err
	}
	return secret, db.Insert(ctx, secret)
}

func init() {
	db.RegisterModel(new(Secret))
}

func (s *Secret) Validate() error {
	if s.OwnerID == 0 && s.RepoID == 0 {
		return errors.New("the secret is not bound to any scope")
	}
	return nil
}

type FindSecretsOptions struct {
	db.ListOptions
	OwnerID  int64
	RepoID   int64
	SecretID int64
	Name     string
}

func (opts FindSecretsOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.OwnerID > 0 {
		cond = cond.And(builder.Eq{"owner_id": opts.OwnerID})
	}
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.SecretID != 0 {
		cond = cond.And(builder.Eq{"id": opts.SecretID})
	}
	if opts.Name != "" {
		cond = cond.And(builder.Eq{"name": strings.ToUpper(opts.Name)})
	}

	return cond
}

// UpdateSecret changes org or user reop secret.
func UpdateSecret(ctx context.Context, secretID int64, data string) error {
	encrypted, err := secret_module.EncryptSecret(setting.SecretKey, data)
	if err != nil {
		return err
	}

	s := &Secret{
		Data: encrypted,
	}
	affected, err := db.GetEngine(ctx).ID(secretID).Cols("data").Update(s)
	if affected != 1 {
		return ErrSecretNotFound{}
	}
	return err
}

func GetSecretsOfTask(ctx context.Context, task *actions_model.ActionTask) (map[string]string, error) {
	secrets := map[string]string{}

	secrets["GITHUB_TOKEN"] = task.Token
	secrets["GITEA_TOKEN"] = task.Token

	if task.Job.Run.IsForkPullRequest && task.Job.Run.TriggerEvent != actions_module.GithubEventPullRequestTarget {
		// ignore secrets for fork pull request, except GITHUB_TOKEN and GITEA_TOKEN which are automatically generated.
		// for the tasks triggered by pull_request_target event, they could access the secrets because they will run in the context of the base branch
		// see the documentation: https://docs.github.com/en/actions/using-workflows/events-that-trigger-workflows#pull_request_target
		return secrets, nil
	}

	ownerSecrets, err := db.Find[Secret](ctx, FindSecretsOptions{OwnerID: task.Job.Run.Repo.OwnerID})
	if err != nil {
		log.Error("find secrets of owner %v: %v", task.Job.Run.Repo.OwnerID, err)
		return nil, err
	}
	repoSecrets, err := db.Find[Secret](ctx, FindSecretsOptions{RepoID: task.Job.Run.RepoID})
	if err != nil {
		log.Error("find secrets of repo %v: %v", task.Job.Run.RepoID, err)
		return nil, err
	}

	for _, secret := range append(ownerSecrets, repoSecrets...) {
		v, err := secret_module.DecryptSecret(setting.SecretKey, secret.Data)
		if err != nil {
			log.Error("decrypt secret %v %q: %v", secret.ID, secret.Name, err)
			return nil, err
		}
		secrets[secret.Name] = v
	}

	return secrets, nil
}
