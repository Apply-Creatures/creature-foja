// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build !gogit

package git

import (
	"bufio"
	"bytes"
	"io"
	"math"
	"strings"

	"code.gitea.io/gitea/modules/analyze"
	"code.gitea.io/gitea/modules/log"

	"github.com/go-enry/go-enry/v2"
)

// GetLanguageStats calculates language stats for git repository at specified commit
func (repo *Repository) GetLanguageStats(commitID string) (map[string]int64, error) {
	// We will feed the commit IDs in order into cat-file --batch, followed by blobs as necessary.
	// so let's create a batch stdin and stdout
	batchStdinWriter, batchReader, cancel := repo.CatFileBatch(repo.Ctx)
	defer cancel()

	writeID := func(id string) error {
		_, err := batchStdinWriter.Write([]byte(id + "\n"))
		return err
	}

	if err := writeID(commitID); err != nil {
		return nil, err
	}
	shaBytes, typ, size, err := ReadBatchLine(batchReader)
	if typ != "commit" {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, ErrNotExist{commitID, ""}
	}

	sha, err := NewIDFromString(string(shaBytes))
	if err != nil {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, ErrNotExist{commitID, ""}
	}

	commit, err := CommitFromReader(repo, sha, io.LimitReader(batchReader, size))
	if err != nil {
		log.Debug("Unable to get commit for: %s. Err: %v", commitID, err)
		return nil, err
	}
	if _, err = batchReader.Discard(1); err != nil {
		return nil, err
	}

	tree := commit.Tree

	entries, err := tree.ListEntriesRecursiveWithSize()
	if err != nil {
		return nil, err
	}

	checker, deferable := repo.CheckAttributeReader(commitID)
	defer deferable()

	contentBuf := bytes.Buffer{}
	var content []byte

	// sizes contains the current calculated size of all files by language
	sizes := make(map[string]int64)
	// by default we will only count the sizes of programming languages or markup languages
	// unless they are explicitly set using linguist-language
	includedLanguage := map[string]bool{}
	// or if there's only one language in the repository
	firstExcludedLanguage := ""
	firstExcludedLanguageSize := int64(0)

	for _, f := range entries {
		select {
		case <-repo.Ctx.Done():
			return sizes, repo.Ctx.Err()
		default:
		}

		contentBuf.Reset()
		content = contentBuf.Bytes()

		if f.Size() == 0 {
			continue
		}

		isVendored := LinguistBoolAttrib{}
		isGenerated := LinguistBoolAttrib{}
		isDocumentation := LinguistBoolAttrib{}
		isDetectable := LinguistBoolAttrib{}

		if checker != nil {
			attrs, err := checker.CheckPath(f.Name())
			if err == nil {
				if vendored, has := attrs["linguist-vendored"]; has {
					isVendored = LinguistBoolAttrib{Value: vendored}
				}
				if generated, has := attrs["linguist-generated"]; has {
					isGenerated = LinguistBoolAttrib{Value: generated}
				}
				if documentation, has := attrs["linguist-documentation"]; has {
					isDocumentation = LinguistBoolAttrib{Value: documentation}
				}
				if detectable, has := attrs["linguist-detectable"]; has {
					isDetectable = LinguistBoolAttrib{Value: detectable}
				}
				if language, has := attrs["linguist-language"]; has && language != "unspecified" && language != "" {
					// group languages, such as Pug -> HTML; SCSS -> CSS
					group := enry.GetLanguageGroup(language)
					if len(group) != 0 {
						language = group
					}

					// this language will always be added to the size
					sizes[language] += f.Size()
					continue
				} else if language, has := attrs["gitlab-language"]; has && language != "unspecified" && language != "" {
					// strip off a ? if present
					if idx := strings.IndexByte(language, '?'); idx >= 0 {
						language = language[:idx]
					}
					if len(language) != 0 {
						// group languages, such as Pug -> HTML; SCSS -> CSS
						group := enry.GetLanguageGroup(language)
						if len(group) != 0 {
							language = group
						}

						// this language will always be added to the size
						sizes[language] += f.Size()
						continue
					}
				}

			}
		}

		if isDetectable.IsFalse() || isVendored.IsTrue() || isDocumentation.IsTrue() ||
			(!isVendored.IsFalse() && analyze.IsVendor(f.Name())) ||
			enry.IsDotFile(f.Name()) ||
			enry.IsConfiguration(f.Name()) ||
			(!isDocumentation.IsFalse() && enry.IsDocumentation(f.Name())) {
			continue
		}

		// If content can not be read or file is too big just do detection by filename

		if f.Size() <= bigFileSize {
			if err := writeID(f.ID.String()); err != nil {
				return nil, err
			}
			_, _, size, err := ReadBatchLine(batchReader)
			if err != nil {
				log.Debug("Error reading blob: %s Err: %v", f.ID.String(), err)
				return nil, err
			}

			sizeToRead := size
			discard := int64(1)
			if size > fileSizeLimit {
				sizeToRead = fileSizeLimit
				discard = size - fileSizeLimit + 1
			}

			_, err = contentBuf.ReadFrom(io.LimitReader(batchReader, sizeToRead))
			if err != nil {
				return nil, err
			}
			content = contentBuf.Bytes()
			err = discardFull(batchReader, discard)
			if err != nil {
				return nil, err
			}
		}
		if !isGenerated.IsTrue() && enry.IsGenerated(f.Name(), content) {
			continue
		}

		// FIXME: Why can't we split this and the IsGenerated tests to avoid reading the blob unless absolutely necessary?
		// - eg. do the all the detection tests using filename first before reading content.
		language := analyze.GetCodeLanguage(f.Name(), content)
		if language == "" {
			continue
		}

		// group languages, such as Pug -> HTML; SCSS -> CSS
		group := enry.GetLanguageGroup(language)
		if group != "" {
			language = group
		}

		included, checked := includedLanguage[language]
		if !checked {
			langType := enry.GetLanguageType(language)
			included = langType == enry.Programming || langType == enry.Markup
			if !included {
				if isDetectable.IsTrue() {
					included = true
				} else {
					continue
				}
			}
			includedLanguage[language] = included
		}
		if included {
			sizes[language] += f.Size()
		} else if len(sizes) == 0 && (firstExcludedLanguage == "" || firstExcludedLanguage == language) {
			firstExcludedLanguage = language
			firstExcludedLanguageSize += f.Size()
		}
		continue
	}

	// If there are no included languages add the first excluded language
	if len(sizes) == 0 && firstExcludedLanguage != "" {
		sizes[firstExcludedLanguage] = firstExcludedLanguageSize
	}

	return mergeLanguageStats(sizes), nil
}

func discardFull(rd *bufio.Reader, discard int64) error {
	if discard > math.MaxInt32 {
		n, err := rd.Discard(math.MaxInt32)
		discard -= int64(n)
		if err != nil {
			return err
		}
	}
	for discard > 0 {
		n, err := rd.Discard(int(discard))
		discard -= int64(n)
		if err != nil {
			return err
		}
	}
	return nil
}
