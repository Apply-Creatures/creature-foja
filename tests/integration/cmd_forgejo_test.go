// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"code.gitea.io/gitea/cmd/forgejo"

	"github.com/urfave/cli/v2"
)

func cmdForgejoCaptureOutput(t *testing.T, args []string, stdin ...string) (string, error) {
	buf := new(bytes.Buffer)

	app := cli.NewApp()
	app.Writer = buf
	app.ErrWriter = buf
	ctx := context.Background()
	ctx = forgejo.ContextSetNoInit(ctx, true)
	ctx = forgejo.ContextSetNoExit(ctx, true)
	ctx = forgejo.ContextSetStdout(ctx, buf)
	ctx = forgejo.ContextSetStderr(ctx, buf)
	if len(stdin) > 0 {
		ctx = forgejo.ContextSetStdin(ctx, strings.NewReader(strings.Join(stdin, "")))
	}
	app.Commands = []*cli.Command{
		forgejo.CmdForgejo(ctx),
	}
	err := app.Run(args)

	return buf.String(), err
}
