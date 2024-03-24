// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"path/filepath"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_nulSeparatedAttributeWriter_ReadAttribute(t *testing.T) {
	wr := &nulSeparatedAttributeWriter{
		attributes: make(chan attributeTriple, 5),
	}

	testStr := ".gitignore\"\n\x00linguist-vendored\x00unspecified\x00"

	n, err := wr.Write([]byte(testStr))

	assert.Len(t, testStr, n)
	assert.NoError(t, err)
	select {
	case attr := <-wr.attributes:
		assert.Equal(t, ".gitignore\"\n", attr.Filename)
		assert.Equal(t, "linguist-vendored", attr.Attribute)
		assert.Equal(t, "unspecified", attr.Value)
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "took too long to read an attribute from the list")
	}
	// Write a second attribute again
	n, err = wr.Write([]byte(testStr))

	assert.Len(t, testStr, n)
	assert.NoError(t, err)

	select {
	case attr := <-wr.attributes:
		assert.Equal(t, ".gitignore\"\n", attr.Filename)
		assert.Equal(t, "linguist-vendored", attr.Attribute)
		assert.Equal(t, "unspecified", attr.Value)
	case <-time.After(100 * time.Millisecond):
		assert.FailNow(t, "took too long to read an attribute from the list")
	}

	// Write a partial attribute
	_, err = wr.Write([]byte("incomplete-file"))
	assert.NoError(t, err)
	_, err = wr.Write([]byte("name\x00"))
	assert.NoError(t, err)

	select {
	case <-wr.attributes:
		assert.FailNow(t, "There should not be an attribute ready to read")
	case <-time.After(100 * time.Millisecond):
	}
	_, err = wr.Write([]byte("attribute\x00"))
	assert.NoError(t, err)
	select {
	case <-wr.attributes:
		assert.FailNow(t, "There should not be an attribute ready to read")
	case <-time.After(100 * time.Millisecond):
	}

	_, err = wr.Write([]byte("value\x00"))
	assert.NoError(t, err)

	attr := <-wr.attributes
	assert.Equal(t, "incomplete-filename", attr.Filename)
	assert.Equal(t, "attribute", attr.Attribute)
	assert.Equal(t, "value", attr.Value)

	_, err = wr.Write([]byte("shouldbe.vendor\x00linguist-vendored\x00set\x00shouldbe.vendor\x00linguist-generated\x00unspecified\x00shouldbe.vendor\x00linguist-language\x00unspecified\x00"))
	assert.NoError(t, err)
	attr = <-wr.attributes
	assert.NoError(t, err)
	assert.EqualValues(t, attributeTriple{
		Filename:  "shouldbe.vendor",
		Attribute: "linguist-vendored",
		Value:     "set",
	}, attr)
	attr = <-wr.attributes
	assert.NoError(t, err)
	assert.EqualValues(t, attributeTriple{
		Filename:  "shouldbe.vendor",
		Attribute: "linguist-generated",
		Value:     "unspecified",
	}, attr)
	attr = <-wr.attributes
	assert.NoError(t, err)
	assert.EqualValues(t, attributeTriple{
		Filename:  "shouldbe.vendor",
		Attribute: "linguist-language",
		Value:     "unspecified",
	}, attr)
}

func TestGitAttributeBareNonBare(t *testing.T) {
	if !SupportCheckAttrOnBare {
		t.Skip("git check-attr supported on bare repo starting with git 2.40")
	}

	repoPath := filepath.Join(testReposDir, "language_stats_repo")
	gitRepo, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer gitRepo.Close()

	for _, commitID := range []string{
		"8fee858da5796dfb37704761701bb8e800ad9ef3",
		"341fca5b5ea3de596dc483e54c2db28633cd2f97",
	} {
		t.Run("GitAttributeChecker/"+commitID, func(t *testing.T) {
			bareChecker, err := gitRepo.GitAttributeChecker(commitID, LinguistAttributes...)
			assert.NoError(t, err)
			t.Cleanup(func() { bareChecker.Close() })

			bareStats, err := bareChecker.CheckPath("i-am-a-python.p")
			assert.NoError(t, err)

			defer test.MockVariableValue(&SupportCheckAttrOnBare, false)()
			cloneChecker, err := gitRepo.GitAttributeChecker(commitID, LinguistAttributes...)
			assert.NoError(t, err)
			t.Cleanup(func() { cloneChecker.Close() })
			cloneStats, err := cloneChecker.CheckPath("i-am-a-python.p")
			assert.NoError(t, err)

			assert.EqualValues(t, cloneStats, bareStats)
		})

		t.Run("GitAttributes/"+commitID, func(t *testing.T) {
			bareStats, err := gitRepo.GitAttributes(commitID, "i-am-a-python.p", LinguistAttributes...)
			assert.NoError(t, err)

			defer test.MockVariableValue(&SupportCheckAttrOnBare, false)()
			cloneStats, err := gitRepo.GitAttributes(commitID, "i-am-a-python.p", LinguistAttributes...)
			assert.NoError(t, err)

			assert.EqualValues(t, cloneStats, bareStats)
		})
	}
}

func TestGitAttributes(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "language_stats_repo")
	gitRepo, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer gitRepo.Close()

	attr, err := gitRepo.GitAttributes("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", LinguistAttributes...)
	assert.NoError(t, err)
	assert.EqualValues(t, map[string]GitAttribute{
		"gitlab-language":        "unspecified",
		"linguist-detectable":    "unspecified",
		"linguist-documentation": "unspecified",
		"linguist-generated":     "unspecified",
		"linguist-language":      "Python",
		"linguist-vendored":      "unspecified",
	}, attr)

	attr, err = gitRepo.GitAttributes("341fca5b5ea3de596dc483e54c2db28633cd2f97", "i-am-a-python.p", LinguistAttributes...)
	assert.NoError(t, err)
	assert.EqualValues(t, map[string]GitAttribute{
		"gitlab-language":        "unspecified",
		"linguist-detectable":    "unspecified",
		"linguist-documentation": "unspecified",
		"linguist-generated":     "unspecified",
		"linguist-language":      "Cobra",
		"linguist-vendored":      "unspecified",
	}, attr)
}

func TestGitAttributeFirst(t *testing.T) {
	repoPath := filepath.Join(testReposDir, "language_stats_repo")
	gitRepo, err := openRepositoryWithDefaultContext(repoPath)
	require.NoError(t, err)
	defer gitRepo.Close()

	t.Run("first is specified", func(t *testing.T) {
		language, err := gitRepo.GitAttributeFirst("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", "linguist-language", "gitlab-language")
		assert.NoError(t, err)
		assert.Equal(t, "Python", language.String())
	})

	t.Run("second is specified", func(t *testing.T) {
		language, err := gitRepo.GitAttributeFirst("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", "gitlab-language", "linguist-language")
		assert.NoError(t, err)
		assert.Equal(t, "Python", language.String())
	})

	t.Run("none is specified", func(t *testing.T) {
		language, err := gitRepo.GitAttributeFirst("8fee858da5796dfb37704761701bb8e800ad9ef3", "i-am-a-python.p", "linguist-detectable", "gitlab-language", "non-existing")
		assert.NoError(t, err)
		assert.Equal(t, "", language.String())
	})
}

func TestGitAttributeStruct(t *testing.T) {
	assert.Equal(t, "", GitAttribute("").String())
	assert.Equal(t, "", GitAttribute("unspecified").String())

	assert.Equal(t, "python", GitAttribute("python").String())

	assert.Equal(t, "text?token=Error", GitAttribute("text?token=Error").String())
	assert.Equal(t, "text", GitAttribute("text?token=Error").Prefix())
}
