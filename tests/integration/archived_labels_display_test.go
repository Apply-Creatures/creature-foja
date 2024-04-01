// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestArchivedLabelVisualProperties(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		session := loginUser(t, "user2")

		// Create labels
		session.MakeRequest(t, NewRequestWithValues(t, "POST", "user2/repo1/labels/new", map[string]string{
			"_csrf":       GetCSRF(t, session, "user2/repo1/labels"),
			"title":       "active_label",
			"description": "",
			"color":       "#aa00aa",
		}), http.StatusSeeOther)
		session.MakeRequest(t, NewRequestWithValues(t, "POST", "user2/repo1/labels/new", map[string]string{
			"_csrf":       GetCSRF(t, session, "user2/repo1/labels"),
			"title":       "archived_label",
			"description": "",
			"color":       "#00aa00",
		}), http.StatusSeeOther)

		// Get ID of label to archive it
		var id string
		doc := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "user2/repo1/labels"), http.StatusOK).Body)
		doc.Find(".issue-label-list .item").Each(func(i int, s *goquery.Selection) {
			label := s.Find(".label-title .label")
			if label.Text() == "archived_label" {
				href, _ := s.Find(".label-issues a.open-issues").Attr("href")
				hrefParts := strings.Split(href, "=")
				id = hrefParts[len(hrefParts)-1]
			}
		})

		// Make label archived
		session.MakeRequest(t, NewRequestWithValues(t, "POST", "user2/repo1/labels/edit", map[string]string{
			"_csrf":       GetCSRF(t, session, "user2/repo1/labels"),
			"id":          id,
			"title":       "archived_label",
			"is_archived": "on",
			"description": "",
			"color":       "#00aa00",
		}), http.StatusSeeOther)

		// Test label properties
		doc = NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "user2/repo1/labels"), http.StatusOK).Body)
		doc.Find(".issue-label-list .item").Each(func(i int, s *goquery.Selection) {
			label := s.Find(".label-title .label")
			style, _ := label.Attr("style")

			if label.Text() == "active_label" {
				assert.False(t, label.HasClass("archived-label"))
				assert.Contains(t, style, "background-color: #aa00aaff")
			} else if label.Text() == "archived_label" {
				assert.True(t, label.HasClass("archived-label"))
				assert.Contains(t, style, "background-color: #00aa007f")
			}
		})
	})
}
