// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/optional"
)

var LinguistAttributes = []string{"linguist-vendored", "linguist-generated", "linguist-language", "gitlab-language", "linguist-documentation", "linguist-detectable"}

// GitAttribute exposes an attribute from the .gitattribute file
type GitAttribute string //nolint:revive

// IsSpecified returns true if the gitattribute is set and not empty
func (ca GitAttribute) IsSpecified() bool {
	return ca != "" && ca != "unspecified"
}

// String returns the value of the attribute or "" if unspecified
func (ca GitAttribute) String() string {
	if !ca.IsSpecified() {
		return ""
	}
	return string(ca)
}

// Prefix returns the value of the attribute before any question mark '?'
//
// sometimes used within gitlab-language: https://docs.gitlab.com/ee/user/project/highlighting.html#override-syntax-highlighting-for-a-file-type
func (ca GitAttribute) Prefix() string {
	s := ca.String()
	if i := strings.IndexByte(s, '?'); i >= 0 {
		return s[:i]
	}
	return s
}

// Bool returns true if "set"/"true", false if "unset"/"false", none otherwise
func (ca GitAttribute) Bool() optional.Option[bool] {
	switch ca {
	case "set", "true":
		return optional.Some(true)
	case "unset", "false":
		return optional.Some(false)
	}
	return optional.None[bool]()
}

// GitAttributeFirst returns the first specified attribute
//
// If treeish is empty, the gitattribute will be read from the current repo (which MUST be a working directory and NOT bare).
func (repo *Repository) GitAttributeFirst(treeish, filename string, attributes ...string) (GitAttribute, error) {
	values, err := repo.GitAttributes(treeish, filename, attributes...)
	if err != nil {
		return "", err
	}
	for _, a := range attributes {
		if values[a].IsSpecified() {
			return values[a], nil
		}
	}
	return "", nil
}

func (repo *Repository) gitCheckAttrCommand(treeish string, attributes ...string) (*Command, *RunOpts, context.CancelFunc, error) {
	if len(attributes) == 0 {
		return nil, nil, nil, fmt.Errorf("no provided attributes to check-attr")
	}

	env := os.Environ()
	var deleteTemporaryFile context.CancelFunc

	// git < 2.40 cannot run check-attr on bare repo, but needs INDEX + WORK_TREE
	hasIndex := treeish == ""
	if !hasIndex && !SupportCheckAttrOnBare {
		indexFilename, worktree, cancel, err := repo.ReadTreeToTemporaryIndex(treeish)
		if err != nil {
			return nil, nil, nil, err
		}
		deleteTemporaryFile = cancel

		env = append(env, "GIT_INDEX_FILE="+indexFilename, "GIT_WORK_TREE="+worktree)

		hasIndex = true

		// clear treeish to read from provided index/work_tree
		treeish = ""
	}
	ctx, cancel := context.WithCancel(repo.Ctx)
	if deleteTemporaryFile != nil {
		ctxCancel := cancel
		cancel = func() {
			ctxCancel()
			deleteTemporaryFile()
		}
	}

	cmd := NewCommand(ctx, "check-attr", "-z")

	if hasIndex {
		cmd.AddArguments("--cached")
	}

	if len(treeish) > 0 {
		cmd.AddArguments("--source")
		cmd.AddDynamicArguments(treeish)
	}
	cmd.AddDynamicArguments(attributes...)

	// Version 2.43.1 has a bug where the behavior of `GIT_FLUSH` is flipped.
	// Ref: https://lore.kernel.org/git/CABn0oJvg3M_kBW-u=j3QhKnO=6QOzk-YFTgonYw_UvFS1NTX4g@mail.gmail.com
	if InvertedGitFlushEnv {
		env = append(env, "GIT_FLUSH=0")
	} else {
		env = append(env, "GIT_FLUSH=1")
	}

	return cmd, &RunOpts{
		Env: env,
		Dir: repo.Path,
	}, cancel, nil
}

// GitAttributes returns gitattribute.
//
// If treeish is empty, the gitattribute will be read from the current repo (which MUST be a working directory and NOT bare).
func (repo *Repository) GitAttributes(treeish, filename string, attributes ...string) (map[string]GitAttribute, error) {
	cmd, runOpts, cancel, err := repo.gitCheckAttrCommand(treeish, attributes...)
	if err != nil {
		return nil, err
	}
	defer cancel()

	stdOut := new(bytes.Buffer)
	runOpts.Stdout = stdOut

	stdErr := new(bytes.Buffer)
	runOpts.Stderr = stdErr

	cmd.AddDashesAndList(filename)

	if err := cmd.Run(runOpts); err != nil {
		return nil, fmt.Errorf("failed to run check-attr: %w\n%s\n%s", err, stdOut.String(), stdErr.String())
	}

	// FIXME: This is incorrect on versions < 1.8.5
	fields := bytes.Split(stdOut.Bytes(), []byte{'\000'})

	if len(fields)%3 != 1 {
		return nil, fmt.Errorf("wrong number of fields in return from check-attr")
	}

	values := make(map[string]GitAttribute, len(attributes))
	for ; len(fields) >= 3; fields = fields[3:] {
		// filename := string(fields[0])
		attribute := string(fields[1])
		value := string(fields[2])
		values[attribute] = GitAttribute(value)
	}
	return values, nil
}

type attributeTriple struct {
	Filename  string
	Attribute string
	Value     string
}

type nulSeparatedAttributeWriter struct {
	tmp        []byte
	attributes chan attributeTriple
	closed     chan struct{}
	working    attributeTriple
	pos        int
}

func (wr *nulSeparatedAttributeWriter) Write(p []byte) (n int, err error) {
	l, read := len(p), 0

	nulIdx := bytes.IndexByte(p, '\x00')
	for nulIdx >= 0 {
		wr.tmp = append(wr.tmp, p[:nulIdx]...)
		switch wr.pos {
		case 0:
			wr.working = attributeTriple{
				Filename: string(wr.tmp),
			}
		case 1:
			wr.working.Attribute = string(wr.tmp)
		case 2:
			wr.working.Value = string(wr.tmp)
		}
		wr.tmp = wr.tmp[:0]
		wr.pos++
		if wr.pos > 2 {
			wr.attributes <- wr.working
			wr.pos = 0
		}
		read += nulIdx + 1
		if l > read {
			p = p[nulIdx+1:]
			nulIdx = bytes.IndexByte(p, '\x00')
		} else {
			return l, nil
		}
	}
	wr.tmp = append(wr.tmp, p...)
	return len(p), nil
}

func (wr *nulSeparatedAttributeWriter) Close() error {
	select {
	case <-wr.closed:
		return nil
	default:
	}
	close(wr.attributes)
	close(wr.closed)
	return nil
}

// GitAttributeChecker creates an AttributeChecker for the given repository and provided commit ID.
//
// If treeish is empty, the gitattribute will be read from the current repo (which MUST be a working directory and NOT bare).
func (repo *Repository) GitAttributeChecker(treeish string, attributes ...string) (AttributeChecker, error) {
	cmd, runOpts, cancel, err := repo.gitCheckAttrCommand(treeish, attributes...)
	if err != nil {
		return AttributeChecker{}, err
	}

	ac := AttributeChecker{
		attributeNumber: len(attributes),
		ctx:             cmd.parentContext,
		cancel:          cancel, // will be cancelled on Close
	}

	stdinReader, stdinWriter, err := os.Pipe()
	if err != nil {
		ac.cancel()
		return AttributeChecker{}, err
	}
	ac.stdinWriter = stdinWriter // will be closed on Close

	lw := new(nulSeparatedAttributeWriter)
	lw.attributes = make(chan attributeTriple, len(attributes))
	lw.closed = make(chan struct{})
	ac.attributesCh = lw.attributes

	cmd.AddArguments("--stdin")
	go func() {
		defer stdinReader.Close()
		defer lw.Close()

		stdErr := new(bytes.Buffer)
		runOpts.Stdin = stdinReader
		runOpts.Stdout = lw
		runOpts.Stderr = stdErr
		err := cmd.Run(runOpts)

		if err != nil && //                       If there is an error we need to return but:
			cmd.parentContext.Err() != err && //  1. Ignore the context error if the context is cancelled or exceeds the deadline (RunWithContext could return c.ctx.Err() which is Canceled or DeadlineExceeded)
			err.Error() != "signal: killed" { // 2. We should not pass up errors due to the program being killed
			log.Error("failed to run attr-check. Error: %v\nStderr: %s", err, stdErr.String())
		}
	}()

	return ac, nil
}

type AttributeChecker struct {
	ctx             context.Context
	cancel          context.CancelFunc
	stdinWriter     *os.File
	attributeNumber int
	attributesCh    <-chan attributeTriple
}

func (ac AttributeChecker) CheckPath(path string) (map[string]GitAttribute, error) {
	if err := ac.ctx.Err(); err != nil {
		return nil, err
	}

	if _, err := ac.stdinWriter.Write([]byte(path + "\x00")); err != nil {
		return nil, err
	}

	rs := make(map[string]GitAttribute)
	for i := 0; i < ac.attributeNumber; i++ {
		select {
		case attr, ok := <-ac.attributesCh:
			if !ok {
				return nil, ac.ctx.Err()
			}
			rs[attr.Attribute] = GitAttribute(attr.Value)
		case <-ac.ctx.Done():
			return nil, ac.ctx.Err()
		}
	}
	return rs, nil
}

func (ac AttributeChecker) Close() error {
	ac.cancel()
	return ac.stdinWriter.Close()
}
