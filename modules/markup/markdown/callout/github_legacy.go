// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package callout

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// Transformer for GitHub's legacy callout markup.
type GitHubLegacyCalloutTransformer struct{}

func (g *GitHubLegacyCalloutTransformer) Transform(node *ast.Document, reader text.Reader, pc parser.Context) {
	supportedCalloutTypes := map[string]bool{"Note": true, "Warning": true}

	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch v := n.(type) {
		case *ast.Blockquote:
			// The first paragraph contains the callout type.
			firstParagraph := v.FirstChild()
			if firstParagraph.ChildCount() < 1 {
				return ast.WalkContinue, nil
			}

			// In the legacy GitHub callout markup, the first node of the first
			// paragraph should be an emphasis.
			calloutNode, ok := firstParagraph.FirstChild().(*ast.Emphasis)
			if !ok {
				return ast.WalkContinue, nil
			}
			calloutText := string(calloutNode.Text(reader.Source()))
			calloutType := strings.ToLower(calloutText)
			// We only support "Note" and "Warning" callouts in legacy mode,
			// match only those.
			if _, has := supportedCalloutTypes[calloutText]; !has {
				return ast.WalkContinue, nil
			}

			// Set the attention attribute on the emphasis
			calloutNode.SetAttributeString("class", []byte("attention-"+calloutType))

			// color the blockquote
			v.SetAttributeString("class", []byte("gt-py-3 attention attention-"+calloutType))

			// Prepend callout icon before the callout node itself
			firstParagraph.InsertBefore(firstParagraph, calloutNode, NewAttention(calloutType))
		}

		return ast.WalkContinue, nil
	})
}
