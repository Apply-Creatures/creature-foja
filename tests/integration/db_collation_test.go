// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"
	"time"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/modules/setting"
	"code.gitea.io/gitea/modules/test"
	"code.gitea.io/gitea/tests"

	"github.com/stretchr/testify/assert"
	"xorm.io/xorm"
)

type TestCollationTbl struct {
	ID  int64
	Txt string `xorm:"VARCHAR(10) UNIQUE"`
}

func TestDatabaseCollationSelfCheckUI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	assertSelfCheckExists := func(exists bool) {
		expectedHTTPResponse := http.StatusOK
		if !exists {
			expectedHTTPResponse = http.StatusNotFound
		}
		session := loginUser(t, "user1")
		req := NewRequest(t, "GET", "/admin/self_check")
		resp := session.MakeRequest(t, req, expectedHTTPResponse)
		htmlDoc := NewHTMLParser(t, resp.Body)

		htmlDoc.AssertElement(t, "a.item[href*='/admin/self_check']", exists)
	}

	if setting.Database.Type.IsMySQL() || setting.Database.Type.IsMSSQL() {
		assertSelfCheckExists(true)
	} else {
		assertSelfCheckExists(false)
	}
}

func TestDatabaseCollation(t *testing.T) {
	x := db.GetEngine(db.DefaultContext).(*xorm.Engine)

	// all created tables should use case-sensitive collation by default
	_, _ = x.Exec("DROP TABLE IF EXISTS test_collation_tbl")
	err := x.Sync(&TestCollationTbl{})
	assert.NoError(t, err)
	_, _ = x.Exec("INSERT INTO test_collation_tbl (txt) VALUES ('main')")
	_, _ = x.Exec("INSERT INTO test_collation_tbl (txt) VALUES ('Main')") // case-sensitive, so it inserts a new row
	_, _ = x.Exec("INSERT INTO test_collation_tbl (txt) VALUES ('main')") // duplicate, so it doesn't insert
	cnt, err := x.Count(&TestCollationTbl{})
	assert.NoError(t, err)
	assert.EqualValues(t, 2, cnt)
	_, _ = x.Exec("DROP TABLE IF EXISTS test_collation_tbl")

	// by default, SQLite3 and PostgreSQL are using case-sensitive collations, but MySQL and MSSQL are not
	// the following tests are only for MySQL and MSSQL
	if !setting.Database.Type.IsMySQL() && !setting.Database.Type.IsMSSQL() {
		t.Skip("only MySQL and MSSQL requires the case-sensitive collation check at the moment")
		return
	}

	t.Run("Default startup makes database collation case-sensitive", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		r, err := db.CheckCollations(x)
		assert.NoError(t, err)
		assert.True(t, r.IsCollationCaseSensitive(r.DatabaseCollation))
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.NotEmpty(t, r.AvailableCollation)
		assert.Empty(t, r.InconsistentCollationColumns)

		// and by the way test the helper functions
		if setting.Database.Type.IsMySQL() {
			assert.True(t, r.IsCollationCaseSensitive("utf8mb4_bin"))
			assert.True(t, r.IsCollationCaseSensitive("utf8mb4_xxx_as_cs"))
			assert.False(t, r.IsCollationCaseSensitive("utf8mb4_general_ci"))
			assert.True(t, r.CollationEquals("abc", "abc"))
			assert.True(t, r.CollationEquals("abc", "utf8mb4_abc"))
			assert.False(t, r.CollationEquals("utf8mb4_general_ci", "utf8mb4_unicode_ci"))
		} else if setting.Database.Type.IsMSSQL() {
			assert.True(t, r.IsCollationCaseSensitive("Latin1_General_CS_AS"))
			assert.False(t, r.IsCollationCaseSensitive("Latin1_General_CI_AS"))
			assert.True(t, r.CollationEquals("abc", "abc"))
			assert.False(t, r.CollationEquals("Latin1_General_CS_AS", "SQL_Latin1_General_CP1_CS_AS"))
		} else {
			assert.Fail(t, "unexpected database type")
		}
	})

	if setting.Database.Type.IsMSSQL() {
		return // skip table converting tests because MSSQL doesn't have a simple solution at the moment
	}

	t.Run("Convert tables to utf8mb4_bin", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer test.MockVariableValue(&setting.Database.CharsetCollation, "utf8mb4_bin")()
		assert.NoError(t, db.ConvertDatabaseTable())
		time.Sleep(5 * time.Second)

		r, err := db.CheckCollations(x)
		assert.NoError(t, err)
		assert.Equal(t, "utf8mb4_bin", r.DatabaseCollation)
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.Empty(t, r.InconsistentCollationColumns)

		_, _ = x.Exec("DROP TABLE IF EXISTS test_tbl")
		_, err = x.Exec("CREATE TABLE test_tbl (txt varchar(10) COLLATE utf8mb4_unicode_ci NOT NULL)")
		assert.NoError(t, err)
		r, err = db.CheckCollations(x)
		assert.NoError(t, err)
		assert.Contains(t, r.InconsistentCollationColumns, "test_tbl.txt")
	})

	t.Run("Convert tables to utf8mb4_general_ci", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer test.MockVariableValue(&setting.Database.CharsetCollation, "utf8mb4_general_ci")()
		assert.NoError(t, db.ConvertDatabaseTable())
		time.Sleep(5 * time.Second)

		r, err := db.CheckCollations(x)
		assert.NoError(t, err)
		assert.Equal(t, "utf8mb4_general_ci", r.DatabaseCollation)
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.Empty(t, r.InconsistentCollationColumns)

		_, _ = x.Exec("DROP TABLE IF EXISTS test_tbl")
		_, err = x.Exec("CREATE TABLE test_tbl (txt varchar(10) COLLATE utf8mb4_bin NOT NULL)")
		assert.NoError(t, err)
		r, err = db.CheckCollations(x)
		assert.NoError(t, err)
		assert.Contains(t, r.InconsistentCollationColumns, "test_tbl.txt")
	})

	t.Run("Convert tables to default case-sensitive collation", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		defer test.MockVariableValue(&setting.Database.CharsetCollation, "")()
		assert.NoError(t, db.ConvertDatabaseTable())
		time.Sleep(5 * time.Second)

		r, err := db.CheckCollations(x)
		assert.NoError(t, err)
		assert.True(t, r.IsCollationCaseSensitive(r.DatabaseCollation))
		assert.True(t, r.CollationEquals(r.ExpectedCollation, r.DatabaseCollation))
		assert.Empty(t, r.InconsistentCollationColumns)
	})
}
