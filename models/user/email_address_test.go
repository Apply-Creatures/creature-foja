// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"fmt"
	"testing"

	"code.gitea.io/gitea/models/db"
	"code.gitea.io/gitea/models/unittest"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/modules/optional"

	"github.com/stretchr/testify/assert"
)

func TestGetEmailAddresses(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	emails, _ := user_model.GetEmailAddresses(db.DefaultContext, int64(1))
	if assert.Len(t, emails, 3) {
		assert.True(t, emails[0].IsPrimary)
		assert.True(t, emails[2].IsActivated)
		assert.False(t, emails[2].IsPrimary)
	}

	emails, _ = user_model.GetEmailAddresses(db.DefaultContext, int64(2))
	if assert.Len(t, emails, 2) {
		assert.True(t, emails[0].IsPrimary)
		assert.True(t, emails[0].IsActivated)
	}
}

func TestIsEmailUsed(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	isExist, _ := user_model.IsEmailUsed(db.DefaultContext, "")
	assert.True(t, isExist)
	isExist, _ = user_model.IsEmailUsed(db.DefaultContext, "user11@example.com")
	assert.True(t, isExist)
	isExist, _ = user_model.IsEmailUsed(db.DefaultContext, "user1234567890@example.com")
	assert.False(t, isExist)
}

func TestMakeEmailPrimary(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	email := &user_model.EmailAddress{
		Email: "user567890@example.com",
	}
	err := user_model.MakeEmailPrimary(db.DefaultContext, email)
	assert.Error(t, err)
	assert.EqualError(t, err, user_model.ErrEmailAddressNotExist{Email: email.Email}.Error())

	email = &user_model.EmailAddress{
		Email: "user11@example.com",
	}
	err = user_model.MakeEmailPrimary(db.DefaultContext, email)
	assert.Error(t, err)
	assert.EqualError(t, err, user_model.ErrEmailNotActivated.Error())

	email = &user_model.EmailAddress{
		Email: "user9999999@example.com",
	}
	err = user_model.MakeEmailPrimary(db.DefaultContext, email)
	assert.Error(t, err)
	assert.True(t, user_model.IsErrUserNotExist(err))

	email = &user_model.EmailAddress{
		Email: "user101@example.com",
	}
	err = user_model.MakeEmailPrimary(db.DefaultContext, email)
	assert.NoError(t, err)

	user, _ := user_model.GetUserByID(db.DefaultContext, int64(10))
	assert.Equal(t, "user101@example.com", user.Email)
}

func TestActivate(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	email := &user_model.EmailAddress{
		ID:    int64(1),
		UID:   int64(1),
		Email: "user11@example.com",
	}
	assert.NoError(t, user_model.ActivateEmail(db.DefaultContext, email))

	emails, _ := user_model.GetEmailAddresses(db.DefaultContext, int64(1))
	assert.Len(t, emails, 3)
	assert.True(t, emails[0].IsActivated)
	assert.True(t, emails[0].IsPrimary)
	assert.False(t, emails[1].IsPrimary)
	assert.True(t, emails[2].IsActivated)
	assert.False(t, emails[2].IsPrimary)
}

func TestListEmails(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Must find all users and their emails
	opts := &user_model.SearchEmailOptions{
		ListOptions: db.ListOptions{
			PageSize: 10000,
		},
	}
	emails, count, err := user_model.SearchEmails(db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.NotEqual(t, int64(0), count)
	assert.True(t, count > 5)

	contains := func(match func(s *user_model.SearchEmailResult) bool) bool {
		for _, v := range emails {
			if match(v) {
				return true
			}
		}
		return false
	}

	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 18 }))
	// 'org3' is an organization
	assert.False(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 3 }))

	// Must find no records
	opts = &user_model.SearchEmailOptions{Keyword: "NOTFOUND"}
	emails, count, err = user_model.SearchEmails(db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Must find users 'user2', 'user28', etc.
	opts = &user_model.SearchEmailOptions{Keyword: "user2"}
	emails, count, err = user_model.SearchEmails(db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.NotEqual(t, int64(0), count)
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 2 }))
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.UID == 27 }))

	// Must find only primary addresses (i.e. from the `user` table)
	opts = &user_model.SearchEmailOptions{IsPrimary: optional.Some(true)}
	emails, _, err = user_model.SearchEmails(db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return s.IsPrimary }))
	assert.False(t, contains(func(s *user_model.SearchEmailResult) bool { return !s.IsPrimary }))

	// Must find only inactive addresses (i.e. not validated)
	opts = &user_model.SearchEmailOptions{IsActivated: optional.Some(false)}
	emails, _, err = user_model.SearchEmails(db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.True(t, contains(func(s *user_model.SearchEmailResult) bool { return !s.IsActivated }))
	assert.False(t, contains(func(s *user_model.SearchEmailResult) bool { return s.IsActivated }))

	// Must find more than one page, but retrieve only one
	opts = &user_model.SearchEmailOptions{
		ListOptions: db.ListOptions{
			PageSize: 5,
			Page:     1,
		},
	}
	emails, count, err = user_model.SearchEmails(db.DefaultContext, opts)
	assert.NoError(t, err)
	assert.Len(t, emails, 5)
	assert.Greater(t, count, int64(len(emails)))
}

func TestEmailAddressValidate(t *testing.T) {
	kases := map[string]error{
		"abc@gmail.com":                  nil,
		"132@hotmail.com":                nil,
		"1-3-2@test.org":                 nil,
		"1.3.2@test.org":                 nil,
		"a_123@test.org.cn":              nil,
		`first.last@iana.org`:            nil,
		`first!last@iana.org`:            nil,
		`first#last@iana.org`:            nil,
		`first$last@iana.org`:            nil,
		`first%last@iana.org`:            nil,
		`first&last@iana.org`:            nil,
		`first'last@iana.org`:            nil,
		`first*last@iana.org`:            nil,
		`first+last@iana.org`:            nil,
		`first/last@iana.org`:            nil,
		`first=last@iana.org`:            nil,
		`first?last@iana.org`:            nil,
		`first^last@iana.org`:            nil,
		"first`last@iana.org":            nil,
		`first{last@iana.org`:            nil,
		`first|last@iana.org`:            nil,
		`first}last@iana.org`:            nil,
		`first~last@iana.org`:            nil,
		`first;last@iana.org`:            user_model.ErrEmailCharIsNotSupported{`first;last@iana.org`},
		".233@qq.com":                    user_model.ErrEmailInvalid{".233@qq.com"},
		"!233@qq.com":                    nil,
		"#233@qq.com":                    nil,
		"$233@qq.com":                    nil,
		"%233@qq.com":                    nil,
		"&233@qq.com":                    nil,
		"'233@qq.com":                    nil,
		"*233@qq.com":                    nil,
		"+233@qq.com":                    nil,
		"-233@qq.com":                    user_model.ErrEmailInvalid{"-233@qq.com"},
		"/233@qq.com":                    nil,
		"=233@qq.com":                    nil,
		"?233@qq.com":                    nil,
		"^233@qq.com":                    nil,
		"_233@qq.com":                    nil,
		"`233@qq.com":                    nil,
		"{233@qq.com":                    nil,
		"|233@qq.com":                    nil,
		"}233@qq.com":                    nil,
		"~233@qq.com":                    nil,
		";233@qq.com":                    user_model.ErrEmailCharIsNotSupported{";233@qq.com"},
		"Foo <foo@bar.com>":              user_model.ErrEmailCharIsNotSupported{"Foo <foo@bar.com>"},
		string([]byte{0xE2, 0x84, 0xAA}): user_model.ErrEmailCharIsNotSupported{string([]byte{0xE2, 0x84, 0xAA})},
	}
	for kase, err := range kases {
		t.Run(kase, func(t *testing.T) {
			assert.EqualValues(t, err, user_model.ValidateEmail(kase))
		})
	}
}

func TestGetActivatedEmailAddresses(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	testCases := []struct {
		UID      int64
		expected []*user_model.ActivatedEmailAddress
	}{
		{
			UID:      1,
			expected: []*user_model.ActivatedEmailAddress{{ID: 9, Email: "user1@example.com"}, {ID: 33, Email: "user1-2@example.com"}, {ID: 34, Email: "user1-3@example.com"}},
		},
		{
			UID:      2,
			expected: []*user_model.ActivatedEmailAddress{{ID: 3, Email: "user2@example.com"}},
		},
		{
			UID:      4,
			expected: []*user_model.ActivatedEmailAddress{{ID: 11, Email: "user4@example.com"}},
		},
		{
			UID:      11,
			expected: []*user_model.ActivatedEmailAddress{},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("User %d", testCase.UID), func(t *testing.T) {
			emails, err := user_model.GetActivatedEmailAddresses(db.DefaultContext, testCase.UID)
			assert.NoError(t, err)
			assert.Equal(t, testCase.expected, emails)
		})
	}
}
