// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package callout

import (
	"github.com/yuin/goldmark/ast"
)

// Attention is an inline for an attention
type Attention struct {
	ast.BaseInline
	AttentionType string
}

// Dump implements Node.Dump.
func (n *Attention) Dump(source []byte, level int) {
	m := map[string]string{}
	m["AttentionType"] = n.AttentionType
	ast.DumpHelper(n, source, level, m, nil)
}

// KindAttention is the NodeKind for Attention
var KindAttention = ast.NewNodeKind("Attention")

// Kind implements Node.Kind.
func (n *Attention) Kind() ast.NodeKind {
	return KindAttention
}

// NewAttention returns a new Attention node.
func NewAttention(attentionType string) *Attention {
	return &Attention{
		BaseInline:    ast.BaseInline{},
		AttentionType: attentionType,
	}
}
