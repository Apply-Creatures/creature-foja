// Copyright Earl Warren <contact@earl-warren.org>
// SPDX-License-Identifier: MIT

package templates

import (
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_StringUtils_HasPrefix(t *testing.T) {
	su := &StringUtils{}
	assert.True(t, su.HasPrefix("ABC", "A"))
	assert.False(t, su.HasPrefix("ABC", "B"))
	assert.True(t, su.HasPrefix(template.HTML("ABC"), "A"))
	assert.False(t, su.HasPrefix(template.HTML("ABC"), "B"))
	assert.False(t, su.HasPrefix(123, "B"))
}
