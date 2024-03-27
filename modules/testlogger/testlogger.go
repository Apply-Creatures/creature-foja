// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package testlogger

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/queue"
)

var (
	prefix    string
	SlowTest  = 10 * time.Second
	SlowFlush = 5 * time.Second
)

var WriterCloser = &testLoggerWriterCloser{}

type testLoggerWriterCloser struct {
	sync.RWMutex
	t    []testing.TB
	errs []error // stack of error, parallel to the stack of testing.TB
	err  error   // fallback if the stack is empty
}

func (w *testLoggerWriterCloser) pushT(t testing.TB) {
	w.Lock()
	w.t = append(w.t, t)
	w.errs = append(w.errs, nil)
	w.Unlock()
}

func (w *testLoggerWriterCloser) Log(level log.Level, msg string) {
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	w.printMsg(msg)
	if level >= log.ERROR {
		w.recordError(msg)
	}
}

// list of error message which will not fail the test
// ideally this list should be empty, however ensuring that it does not grow
// is already a good first step.
var ignoredErrorMessageSuffixes = []string{
	// only seen on mysql tests https://codeberg.org/forgejo/forgejo/pulls/2657#issuecomment-1693055
	`table columns using inconsistent collation, they should use "utf8mb4_0900_ai_ci". Please go to admin panel Self Check page`,

	// TestAPIDeleteReleaseByTagName
	// action notification were a commit cannot be computed (because the commit got deleted)
	`Notify() [E] an error occurred while executing the DeleteRelease actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/release-tag, rel_path: ]`,
	`Notify() [E] an error occurred while executing the PushCommits actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/release-tag, rel_path: ]`,

	// TestAPIRepoTags
	`Notify() [E] an error occurred while executing the DeleteRelease actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/gitea/22, rel_path: ]`,
	`Notify() [E] an error occurred while executing the PushCommits actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/gitea/22, rel_path: ]`,

	// TestAPIDeleteTagByName
	`Notify() [E] an error occurred while executing the DeleteRelease actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/delete-tag, rel_path: ]`,
	`Notify() [E] an error occurred while executing the PushCommits actions method: gitRepo.GetCommit: object does not exist [id: refs/tags/delete-tag, rel_path: ]`,

	// TestAPIGenerateRepo
	`Notify() [E] an error occurred while executing the CreateRepository actions method: gitRepo.GetCommit: object does not exist [id: , rel_path: ]`,

	// TestAPIPullReview
	`PullRequestReview() [E] Unsupported review webhook type`,

	// TestAPIPullReviewRequest
	`ToAPIPullRequest() [E] unable to resolve PR head ref`,

	// TestAPILFSUpload
	`Put() [E] Whilst putting LFS OID[ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb]: Failed to copy to tmpPath: ca/97/8112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb Error: content size does not match`,
	`[E] Error putting LFS MetaObject [ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb] into content store. Error: content size does not match`,
	`UploadHandler() [E] Upload does not match LFS MetaObject [ca978112ca1bbdcafac231b39a23dc4da786eff8147c4e72b9807785afee48bb]. Error: content size does not match`,
	`Put() [E] Whilst putting LFS OID[2581dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a]: Failed to copy to tmpPath: 25/81/dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a Error: content hash does not match OID`,
	`[E] Error putting LFS MetaObject [2581dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a] into content store. Error: content hash does not match OID`,
	`UploadHandler() [E] Upload does not match LFS MetaObject [2581dd7bbc1fe44726de4b7dd806a087a978b9c5aec0a60481259e34be09b06a]. Error: content hash does not match OID`,
	`UploadHandler() [E] Upload does not match LFS MetaObject [83de2e488b89a0aa1c97496b888120a28b0c1e15463a4adb8405578c540f36d4]. Error: content size does not match`,

	// TestAPILFSVerify
	`getAuthenticatedMeta() [E] Unable to get LFS OID[fb8f7d8435968c4f82a726a92395be4d16f2f63116caf36c8ad35c60831ab042] Error: LFS Meta object does not exist`,

	// TestAPIUpdateOrgAvatar
	`UpdateAvatar() [E] UploadAvatar: image.DecodeConfig: image: unknown format`,

	// TestGetAttachment
	`/data/attachments/a/0/a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a18: no such file or directory`,

	// TestBlockUser
	`BlockedUsersUnblock() [E] IsOrganization: org3 is an organization not a user`,
	`BlockedUsersBlock() [E] IsOrganization: org3 is an organization not a user`,
	`Action() [E] Cannot perform this action on an organization "unblock"`,
	`Action() [E] Cannot perform this action on an organization "block"`,

	// TestBlockActions
	`/gitea-repositories/user10/repo7.git Error: no such file or directory`,

	// TestRebuildCargo
	`RebuildCargoIndex() [E] RebuildIndex failed: GetRepositoryByOwnerAndName: repository does not exist [id: 0, uid: 0, owner_name: user2, name: _cargo-index]`,
}

func (w *testLoggerWriterCloser) recordError(msg string) {
	for _, s := range ignoredErrorMessageSuffixes {
		if strings.HasSuffix(msg, s) {
			return
		}
	}

	w.Lock()
	defer w.Unlock()

	err := w.err
	if len(w.errs) > 0 {
		err = w.errs[len(w.errs)-1]
	}

	if len(w.t) > 0 {
		// format error message to easily add it to the ignore list
		msg = fmt.Sprintf("// %s\n\t`%s`,", w.t[len(w.t)-1].Name(), msg)
	}

	err = errors.Join(err, errors.New(msg))

	if len(w.errs) > 0 {
		w.errs[len(w.errs)-1] = err
	} else {
		w.err = err
	}
}

func (w *testLoggerWriterCloser) printMsg(msg string) {
	// There was a data race problem: the logger system could still try to output logs after the runner is finished.
	// So we must ensure that the "t" in stack is still valid.
	w.RLock()
	defer w.RUnlock()

	if len(w.t) > 0 {
		t := w.t[len(w.t)-1]
		t.Log(msg)
	} else {
		// if there is no running test, the log message should be outputted to console, to avoid losing important information.
		// the "???" prefix is used to match the "===" and "+++" in PrintCurrentTest
		fmt.Fprintln(os.Stdout, "??? [TestLogger]", msg)
	}
}

func (w *testLoggerWriterCloser) popT() error {
	w.Lock()
	defer w.Unlock()

	if len(w.t) > 0 {
		w.t = w.t[:len(w.t)-1]
		err := w.errs[len(w.errs)-1]
		w.errs = w.errs[:len(w.errs)-1]
		return err
	}
	return w.err
}

func (w *testLoggerWriterCloser) Reset() error {
	w.Lock()
	if len(w.t) > 0 {
		for _, t := range w.t {
			if t == nil {
				continue
			}
			_, _ = fmt.Fprintf(os.Stdout, "Unclosed logger writer in test: %s", t.Name())
			t.Errorf("Unclosed logger writer in test: %s", t.Name())
		}
		w.t = nil
		w.errs = nil
	}
	err := w.err
	w.err = nil
	w.Unlock()
	return err
}

// PrintCurrentTest prints the current test to os.Stdout
func PrintCurrentTest(t testing.TB, skip ...int) func() {
	t.Helper()
	start := time.Now()
	actualSkip := 1
	if len(skip) > 0 {
		actualSkip = skip[0] + 1
	}
	_, filename, line, _ := runtime.Caller(actualSkip)

	if log.CanColorStdout {
		_, _ = fmt.Fprintf(os.Stdout, "=== %s (%s:%d)\n", fmt.Formatter(log.NewColoredValue(t.Name())), strings.TrimPrefix(filename, prefix), line)
	} else {
		_, _ = fmt.Fprintf(os.Stdout, "=== %s (%s:%d)\n", t.Name(), strings.TrimPrefix(filename, prefix), line)
	}
	WriterCloser.pushT(t)
	return func() {
		took := time.Since(start)
		if took > SlowTest {
			if log.CanColorStdout {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s is a slow test (took %v)\n", fmt.Formatter(log.NewColoredValue(t.Name(), log.Bold, log.FgYellow)), fmt.Formatter(log.NewColoredValue(took, log.Bold, log.FgYellow)))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s is a slow test (took %v)\n", t.Name(), took)
			}
		}
		timer := time.AfterFunc(SlowFlush, func() {
			if log.CanColorStdout {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s ... still flushing after %v ...\n", fmt.Formatter(log.NewColoredValue(t.Name(), log.Bold, log.FgRed)), SlowFlush)
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s ... still flushing after %v ...\n", t.Name(), SlowFlush)
			}
		})
		if err := queue.GetManager().FlushAll(context.Background(), time.Minute); err != nil {
			t.Errorf("Flushing queues failed with error %v", err)
		}
		timer.Stop()
		flushTook := time.Since(start) - took
		if flushTook > SlowFlush {
			if log.CanColorStdout {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s had a slow clean-up flush (took %v)\n", fmt.Formatter(log.NewColoredValue(t.Name(), log.Bold, log.FgRed)), fmt.Formatter(log.NewColoredValue(flushTook, log.Bold, log.FgRed)))
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "+++ %s had a slow clean-up flush (took %v)\n", t.Name(), flushTook)
			}
		}

		if err := WriterCloser.popT(); err != nil {
			t.Errorf("testlogger.go:recordError() FATAL ERROR: log.Error has been called: %v", err)
		}
	}
}

// Printf takes a format and args and prints the string to os.Stdout
func Printf(format string, args ...any) {
	if log.CanColorStdout {
		for i := 0; i < len(args); i++ {
			args[i] = log.NewColoredValue(args[i])
		}
	}
	_, _ = fmt.Fprintf(os.Stdout, "\t"+format, args...)
}

// NewTestLoggerWriter creates a TestLogEventWriter as a log.LoggerProvider
func NewTestLoggerWriter(name string, mode log.WriterMode) log.EventWriter {
	w := &TestLogEventWriter{}
	w.base = log.NewEventWriterBase(name, "test-log-writer", mode)
	w.writer = WriterCloser
	return w
}

// TestLogEventWriter is a logger which will write to the testing log
type TestLogEventWriter struct {
	base   *log.EventWriterBaseImpl
	writer *testLoggerWriterCloser
}

// Base implements log.EventWriter.
func (t *TestLogEventWriter) Base() *log.EventWriterBaseImpl {
	return t.base
}

// GetLevel implements log.EventWriter.
func (t *TestLogEventWriter) GetLevel() log.Level {
	return t.base.GetLevel()
}

// GetWriterName implements log.EventWriter.
func (t *TestLogEventWriter) GetWriterName() string {
	return t.base.GetWriterName()
}

// GetWriterType implements log.EventWriter.
func (t *TestLogEventWriter) GetWriterType() string {
	return t.base.GetWriterType()
}

// Run implements log.EventWriter.
func (t *TestLogEventWriter) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-t.base.Queue:
			if !ok {
				return
			}

			var errorMsg string

			switch msg := event.Msg.(type) {
			case string:
				errorMsg = msg
			case []byte:
				errorMsg = string(msg)
			case io.WriterTo:
				var buf bytes.Buffer
				if _, err := msg.WriteTo(&buf); err != nil {
					panic(err)
				}
				errorMsg = buf.String()
			default:
				errorMsg = fmt.Sprint(msg)
			}
			t.writer.Log(event.Origin.Level, errorMsg)
		}
	}
}

func init() {
	const relFilePath = "modules/testlogger/testlogger.go"
	_, filename, _, _ := runtime.Caller(0)
	if !strings.HasSuffix(filename, relFilePath) {
		panic("source code file path doesn't match expected: " + relFilePath)
	}
	prefix = strings.TrimSuffix(filename, relFilePath)
}
