package mock

import (
	"context"
	"errors"

	redis "github.com/redis/go-redis/v9"
)

// InMemoryMockRedis is a very primitive in-memory redis-like feature. The main
// purpose of this struct is to give some backend to mocked unit tests.
type InMemoryMockRedis struct {
	queues map[string][][]byte
}

func NewInMemoryMockRedis() InMemoryMockRedis {
	return InMemoryMockRedis{
		queues: map[string][][]byte{},
	}
}

func (r *InMemoryMockRedis) LLen(ctx context.Context, key string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	cmd.SetVal(int64(len(r.queues[key])))
	return cmd
}

func (r *InMemoryMockRedis) SAdd(ctx context.Context, key string, content []byte) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)

	for _, value := range r.queues[key] {
		if string(value) == string(content) {
			cmd.SetVal(0)

			return cmd
		}
	}

	r.queues[key] = append(r.queues[key], content)

	cmd.SetVal(1)

	return cmd
}

func (r *InMemoryMockRedis) RPush(ctx context.Context, key string, content []byte) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)

	r.queues[key] = append(r.queues[key], content)

	cmd.SetVal(1)

	return cmd
}

func (r *InMemoryMockRedis) LPop(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)

	queue, found := r.queues[key]
	if !found {
		cmd.SetErr(errors.New("queue not found"))

		return cmd
	}

	if len(queue) < 1 {
		cmd.SetErr(errors.New("queue is empty"))

		return cmd
	}

	value, rest := queue[0], queue[1:]

	r.queues[key] = rest

	cmd.SetVal(string(value))

	return cmd
}

func (r *InMemoryMockRedis) SRem(ctx context.Context, key string, content []byte) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)

	queue, found := r.queues[key]
	if !found {
		cmd.SetErr(errors.New("queue not found"))

		return cmd
	}

	if len(queue) < 1 {
		cmd.SetErr(errors.New("queue is empty"))

		return cmd
	}

	newList := [][]byte{}

	for _, value := range queue {
		if string(value) != string(content) {
			newList = append(newList, value)
		}
	}

	r.queues[key] = newList

	cmd.SetVal(1)

	return cmd
}

func (r *InMemoryMockRedis) SIsMember(ctx context.Context, key string, content []byte) *redis.BoolCmd {
	cmd := redis.NewBoolCmd(ctx)

	queue, found := r.queues[key]
	if !found {
		cmd.SetErr(errors.New("queue not found"))

		return cmd
	}

	for _, value := range queue {
		if string(value) == string(content) {
			cmd.SetVal(true)

			return cmd
		}
	}

	cmd.SetVal(false)

	return cmd
}
