package ratelimiter

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type GCRAImpl struct {
	rdb       redis.UniversalClient
	scriptSha string
	callCnt   atomic.Int64
}

func InitGCRAImpl(ctx context.Context, redisAddrs []string) (*GCRAImpl, error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	scriptSha, err := redisLoadScript(ctx, rdb, "../gcra.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to ScriptLoad: %w", err)
	}
	return &GCRAImpl{
		rdb:       rdb,
		scriptSha: scriptSha,
		callCnt:   atomic.Int64{},
	}, nil
}

func (i *GCRAImpl) Name() string {
	return "GCRA_impl"
}

func (i *GCRAImpl) Allow(ctx context.Context, key string, nowMs int64, config Config) (allowed bool, err error) {
	if err := config.Validate(); err != nil {
		return false, err
	}
	i.callCnt.Add(1)
	res, err := i.rdb.EvalSha(ctx, i.scriptSha, []string{key},
		config.RateSec, config.Burst, config.Cost, config.TTLMs).Result()
	if err != nil {
		return false, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) != 2 {
		return false, fmt.Errorf("failed to convert rdb.EvalSha result to []any")
	}
	return arr[0] == int64(1), nil
}

func (i *GCRAImpl) RedisCalls() int64 {
	return i.callCnt.Load()
}

func (i *GCRAImpl) Teardown() {
	i.rdb.Close()
}
