// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations //nolint:revive

import (
	"context"
	"fmt"
	"os"

	"code.gitea.io/gitea/models/forgejo/semver"
	forgejo_v1_20 "code.gitea.io/gitea/models/forgejo_migrations/v1_20"
	"code.gitea.io/gitea/modules/git"
	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/names"
)

// ForgejoVersion describes the Forgejo version table. Should have only one row with id = 1.
type ForgejoVersion struct {
	ID      int64 `xorm:"pk autoincr"`
	Version int64
}

type Migration struct {
	description string
	migrate     func(*xorm.Engine) error
}

// NewMigration creates a new migration.
func NewMigration(desc string, fn func(*xorm.Engine) error) *Migration {
	return &Migration{desc, fn}
}

// This is a sequence of additional Forgejo migrations.
// Add new migrations to the bottom of the list.
var migrations = []*Migration{
	// v0 -> v1
	NewMigration("Add Forgejo Blocked Users table", forgejo_v1_20.AddForgejoBlockedUser),
	// v1 -> v2
	NewMigration("create the forgejo_sem_ver table", forgejo_v1_20.CreateSemVerTable),
	// v2 -> v3
	NewMigration("create the forgejo_auth_token table", forgejo_v1_20.CreateAuthorizationTokenTable),
}

// GetCurrentDBVersion returns the current Forgejo database version.
func GetCurrentDBVersion(x *xorm.Engine) (int64, error) {
	if err := x.Sync(new(ForgejoVersion)); err != nil {
		return -1, fmt.Errorf("sync: %w", err)
	}

	currentVersion := &ForgejoVersion{ID: 1}
	has, err := x.Get(currentVersion)
	if err != nil {
		return -1, fmt.Errorf("get: %w", err)
	}
	if !has {
		return -1, nil
	}
	return currentVersion.Version, nil
}

// ExpectedVersion returns the expected Forgejo database version.
func ExpectedVersion() int64 {
	return int64(len(migrations))
}

// EnsureUpToDate will check if the Forgejo database is at the correct version.
func EnsureUpToDate(x *xorm.Engine) error {
	currentDB, err := GetCurrentDBVersion(x)
	if err != nil {
		return err
	}

	if currentDB < 0 {
		return fmt.Errorf("database has not been initialized")
	}

	expected := ExpectedVersion()

	if currentDB != expected {
		return fmt.Errorf(`current Forgejo database version %d is not equal to the expected version %d. Please run "forgejo [--config /path/to/app.ini] migrate" to update the database version`, currentDB, expected)
	}

	return nil
}

// Migrate Forgejo database to current version.
func Migrate(x *xorm.Engine) error {
	// Set a new clean the default mapper to GonicMapper as that is the default for .
	x.SetMapper(names.GonicMapper{})
	if err := x.Sync(new(ForgejoVersion)); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	currentVersion := &ForgejoVersion{ID: 1}
	has, err := x.Get(currentVersion)
	if err != nil {
		return fmt.Errorf("get: %w", err)
	} else if !has {
		// If the version record does not exist we think
		// it is a fresh installation and we can skip all migrations.
		currentVersion.ID = 0
		currentVersion.Version = ExpectedVersion()

		if _, err = x.InsertOne(currentVersion); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}

	v := currentVersion.Version

	// Downgrading Forgejo's database version not supported
	if v > ExpectedVersion() {
		msg := fmt.Sprintf("Your Forgejo database (migration version: %d) is for a newer version of Forgejo, you cannot use the newer database for this old Forgejo release (%d).", v, ExpectedVersion())
		msg += "\nForgejo will exit to keep your database safe and unchanged. Please use the correct Forgejo release, do not change the migration version manually (incorrect manual operation may cause data loss)."
		if !setting.IsProd {
			msg += fmt.Sprintf("\nIf you are in development and really know what you're doing, you can force changing the migration version by executing: UPDATE forgejo_version SET version=%d WHERE id=1;", ExpectedVersion())
		}
		_, _ = fmt.Fprintln(os.Stderr, msg)
		log.Fatal(msg)
		return nil
	}

	// Some migration tasks depend on the git command
	if git.DefaultContext == nil {
		if err = git.InitSimple(context.Background()); err != nil {
			return err
		}
	}

	// Migrate
	for i, m := range migrations[v:] {
		log.Info("Migration[%d]: %s", v+int64(i), m.description)
		// Reset the mapper between each migration - migrations are not supposed to depend on each other
		x.SetMapper(names.GonicMapper{})
		if err = m.migrate(x); err != nil {
			return fmt.Errorf("migration[%d]: %s failed: %w", v+int64(i), m.description, err)
		}
		currentVersion.Version = v + int64(i) + 1
		if _, err = x.ID(1).Update(currentVersion); err != nil {
			return err
		}
	}

	if err := x.Sync(new(semver.ForgejoSemVer)); err != nil {
		return fmt.Errorf("sync: %w", err)
	}

	return semver.SetVersionStringWithEngine(x, setting.ForgejoVersion)
}
