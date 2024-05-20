// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package queue

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"code.gitea.io/gitea/modules/nosql"
	"code.gitea.io/gitea/modules/setting"

	"github.com/stretchr/testify/assert"
)

const defaultTestRedisServer = "127.0.0.1:6379"

func testRedisHost() string {
	value := os.Getenv("TEST_REDIS_SERVER")
	if value != "" {
		return value
	}

	return defaultTestRedisServer
}

func waitRedisReady(conn string, dur time.Duration) (ready bool) {
	ctxTimed, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	for t := time.Now(); ; time.Sleep(50 * time.Millisecond) {
		ret := nosql.GetManager().GetRedisClient(conn).Ping(ctxTimed)
		if ret.Err() == nil {
			return true
		}
		if time.Since(t) > dur {
			return false
		}
	}
}

func redisServerCmd(t *testing.T) *exec.Cmd {
	redisServerProg, err := exec.LookPath("redis-server")
	if err != nil {
		return nil
	}
	c := &exec.Cmd{
		Path:   redisServerProg,
		Args:   []string{redisServerProg, "--bind", "127.0.0.1", "--port", "6379"},
		Dir:    t.TempDir(),
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return c
}

func TestBaseRedis(t *testing.T) {
	redisAddress := "redis://" + testRedisHost() + "/0"
	queueSettings := setting.QueueSettings{
		Length:  10,
		ConnStr: redisAddress,
	}

	var redisServer *exec.Cmd
	if !waitRedisReady(redisAddress, 0) {
		redisServer = redisServerCmd(t)

		if redisServer == nil {
			t.Skip("redis-server not found in Forgejo test yet")
			return
		}

		assert.NoError(t, redisServer.Start())
		if !assert.True(t, waitRedisReady(redisAddress, 5*time.Second), "start redis-server") {
			return
		}
	}

	defer func() {
		if redisServer != nil {
			_ = redisServer.Process.Signal(os.Interrupt)
			_ = redisServer.Wait()
		}
	}()

	testQueueBasic(t, newBaseRedisSimple, toBaseConfig("baseRedis", queueSettings), false)
	testQueueBasic(t, newBaseRedisUnique, toBaseConfig("baseRedisUnique", queueSettings), true)
}

func TestBaseRedisWithPrefix(t *testing.T) {
	redisAddress := "redis://" + testRedisHost() + "/0?prefix=forgejo:queue:"
	queueSettings := setting.QueueSettings{
		Length:  10,
		ConnStr: redisAddress,
	}

	var redisServer *exec.Cmd
	if !waitRedisReady(redisAddress, 0) {
		redisServer = redisServerCmd(t)

		if redisServer == nil {
			t.Skip("redis-server not found in Forgejo test yet")
			return
		}

		assert.NoError(t, redisServer.Start())
		if !assert.True(t, waitRedisReady(redisAddress, 5*time.Second), "start redis-server") {
			return
		}
	}

	defer func() {
		if redisServer != nil {
			_ = redisServer.Process.Signal(os.Interrupt)
			_ = redisServer.Wait()
		}
	}()

	testQueueBasic(t, newBaseRedisSimple, toBaseConfig("baseRedis", queueSettings), false)
	testQueueBasic(t, newBaseRedisUnique, toBaseConfig("baseRedisUnique", queueSettings), true)
}
