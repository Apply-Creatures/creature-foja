// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPktLine(t *testing.T) {
	ctx := context.Background()

	t.Run("Read", func(t *testing.T) {
		s := strings.NewReader("0000")
		r := bufio.NewReader(s)
		result, err := readPktLine(ctx, r, pktLineTypeFlush)
		assert.NoError(t, err)
		assert.Equal(t, pktLineTypeFlush, result.Type)

		s = strings.NewReader("0006a\n")
		r = bufio.NewReader(s)
		result, err = readPktLine(ctx, r, pktLineTypeData)
		assert.NoError(t, err)
		assert.Equal(t, pktLineTypeData, result.Type)
		assert.Equal(t, []byte("a\n"), result.Data)

		s = strings.NewReader("0004")
		r = bufio.NewReader(s)
		result, err = readPktLine(ctx, r, pktLineTypeData)
		assert.Error(t, err)
		assert.Nil(t, result)

		data := strings.Repeat("x", 65516)
		r = bufio.NewReader(strings.NewReader("fff0" + data))
		result, err = readPktLine(ctx, r, pktLineTypeData)
		assert.NoError(t, err)
		assert.Equal(t, pktLineTypeData, result.Type)
		assert.Equal(t, []byte(data), result.Data)

		r = bufio.NewReader(strings.NewReader("fff1a"))
		result, err = readPktLine(ctx, r, pktLineTypeData)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("Write", func(t *testing.T) {
		w := bytes.NewBuffer([]byte{})
		err := writeFlushPktLine(ctx, w)
		assert.NoError(t, err)
		assert.Equal(t, []byte("0000"), w.Bytes())

		w.Reset()
		err = writeDataPktLine(ctx, w, []byte("a\nb"))
		assert.NoError(t, err)
		assert.Equal(t, []byte("0007a\nb"), w.Bytes())

		w.Reset()
		data := bytes.Repeat([]byte{0x05}, 288)
		err = writeDataPktLine(ctx, w, data)
		assert.NoError(t, err)
		assert.Equal(t, append([]byte("0124"), data...), w.Bytes())

		w.Reset()
		err = writeDataPktLine(ctx, w, nil)
		assert.Error(t, err)
		assert.Empty(t, w.Bytes())

		w.Reset()
		data = bytes.Repeat([]byte{0x64}, 65516)
		err = writeDataPktLine(ctx, w, data)
		assert.NoError(t, err)
		assert.Equal(t, append([]byte("fff0"), data...), w.Bytes())

		w.Reset()
		err = writeDataPktLine(ctx, w, bytes.Repeat([]byte{0x64}, 65516+1))
		assert.Error(t, err)
		assert.Empty(t, w.Bytes())
	})
}
