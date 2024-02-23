// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"time"

	"code.gitea.io/gitea/modules/container"
	"code.gitea.io/gitea/modules/log"
)

// UI settings
var UI = struct {
	ExplorePagingNum        int
	SitemapPagingNum        int
	IssuePagingNum          int
	RepoSearchPagingNum     int
	MembersPagingNum        int
	FeedMaxCommitNum        int
	FeedPagingNum           int
	PackagesPagingNum       int
	GraphMaxCommitNum       int
	CodeCommentLines        int
	ReactionMaxUserNum      int
	MaxDisplayFileSize      int64
	ShowUserEmail           bool
	DefaultShowFullName     bool
	DefaultTheme            string
	Themes                  []string
	Reactions               []string
	ReactionsLookup         container.Set[string] `ini:"-"`
	CustomEmojis            []string
	CustomEmojisMap         map[string]string `ini:"-"`
	SearchRepoDescription   bool
	OnlyShowRelevantRepos   bool
	ExploreDefaultSort      string `ini:"EXPLORE_PAGING_DEFAULT_SORT"`
	PreferredTimestampTense string

	AmbiguousUnicodeDetection bool
	SkipEscapeContexts        []string

	Notification struct {
		MinTimeout            time.Duration
		TimeoutStep           time.Duration
		MaxTimeout            time.Duration
		EventSourceUpdateTime time.Duration
	} `ini:"ui.notification"`

	SVG struct {
		Enabled bool `ini:"ENABLE_RENDER"`
	} `ini:"ui.svg"`

	CSV struct {
		MaxFileSize int64
	} `ini:"ui.csv"`

	Admin struct {
		UserPagingNum   int
		RepoPagingNum   int
		NoticePagingNum int
		OrgPagingNum    int
	} `ini:"ui.admin"`
	User struct {
		RepoPagingNum int
	} `ini:"ui.user"`
	Meta struct {
		Author      string
		Description string
		Keywords    string
	} `ini:"ui.meta"`
}{
	ExplorePagingNum:        20,
	SitemapPagingNum:        20,
	IssuePagingNum:          20,
	RepoSearchPagingNum:     20,
	MembersPagingNum:        20,
	FeedMaxCommitNum:        5,
	FeedPagingNum:           20,
	PackagesPagingNum:       20,
	GraphMaxCommitNum:       100,
	CodeCommentLines:        4,
	ReactionMaxUserNum:      10,
	MaxDisplayFileSize:      8388608,
	DefaultTheme:            `forgejo-auto`,
	Themes:                  []string{`forgejo-auto`, `forgejo-light`, `forgejo-dark`, `gitea-auto`, `gitea-light`, `gitea-dark`, `forgejo-auto-deuteranopia-protanopia`, `forgejo-light-deuteranopia-protanopia`, `forgejo-dark-deuteranopia-protanopia`, `forgejo-auto-tritanopia`, `forgejo-light-tritanopia`, `forgejo-dark-tritanopia`},
	Reactions:               []string{`+1`, `-1`, `laugh`, `hooray`, `confused`, `heart`, `rocket`, `eyes`},
	CustomEmojis:            []string{`git`, `gitea`, `codeberg`, `gitlab`, `github`, `gogs`, `forgejo`},
	CustomEmojisMap:         map[string]string{"git": ":git:", "gitea": ":gitea:", "codeberg": ":codeberg:", "gitlab": ":gitlab:", "github": ":github:", "gogs": ":gogs:", "forgejo": ":forgejo:"},
	PreferredTimestampTense: "mixed",

	AmbiguousUnicodeDetection: true,
	SkipEscapeContexts:        []string{},

	Notification: struct {
		MinTimeout            time.Duration
		TimeoutStep           time.Duration
		MaxTimeout            time.Duration
		EventSourceUpdateTime time.Duration
	}{
		MinTimeout:            10 * time.Second,
		TimeoutStep:           10 * time.Second,
		MaxTimeout:            60 * time.Second,
		EventSourceUpdateTime: 10 * time.Second,
	},
	SVG: struct {
		Enabled bool `ini:"ENABLE_RENDER"`
	}{
		Enabled: true,
	},
	CSV: struct {
		MaxFileSize int64
	}{
		MaxFileSize: 524288,
	},
	Admin: struct {
		UserPagingNum   int
		RepoPagingNum   int
		NoticePagingNum int
		OrgPagingNum    int
	}{
		UserPagingNum:   50,
		RepoPagingNum:   50,
		NoticePagingNum: 25,
		OrgPagingNum:    50,
	},
	User: struct {
		RepoPagingNum int
	}{
		RepoPagingNum: 15,
	},
	Meta: struct {
		Author      string
		Description string
		Keywords    string
	}{
		Author:      "Forgejo – Beyond coding. We forge.",
		Description: "Forgejo is a self-hosted lightweight software forge. Easy to install and low maintenance, it just does the job.",
		Keywords:    "git,forge,forgejo",
	},
}

func loadUIFrom(rootCfg ConfigProvider) {
	mustMapSetting(rootCfg, "ui", &UI)
	sec := rootCfg.Section("ui")
	UI.ShowUserEmail = sec.Key("SHOW_USER_EMAIL").MustBool(true)
	UI.DefaultShowFullName = sec.Key("DEFAULT_SHOW_FULL_NAME").MustBool(false)
	UI.SearchRepoDescription = sec.Key("SEARCH_REPO_DESCRIPTION").MustBool(true)

	if UI.PreferredTimestampTense != "mixed" && UI.PreferredTimestampTense != "absolute" {
		log.Fatal("ui.PREFERRED_TIMESTAMP_TENSE must be either 'mixed' or 'absolute'")
	}

	// OnlyShowRelevantRepos=false is important for many private/enterprise instances,
	// because many private repositories do not have "description/topic", users just want to search by their names.
	UI.OnlyShowRelevantRepos = sec.Key("ONLY_SHOW_RELEVANT_REPOS").MustBool(false)

	UI.ReactionsLookup = make(container.Set[string])
	for _, reaction := range UI.Reactions {
		UI.ReactionsLookup.Add(reaction)
	}
	UI.CustomEmojisMap = make(map[string]string)
	for _, emoji := range UI.CustomEmojis {
		UI.CustomEmojisMap[emoji] = ":" + emoji + ":"
	}
}
