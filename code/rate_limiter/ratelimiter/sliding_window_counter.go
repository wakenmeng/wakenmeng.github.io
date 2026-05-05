package ratelimiter

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

// Sliding Window Counter
type SWCImpl struct {
	rdb       redis.UniversalClient
	scriptSha string
	callCnt   atomic.Int64
}

func InitSWCImpl(ctx context.Context, redisAddrs []string) (*SWCImpl, error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	scriptSha, err := redisLoadScript(ctx, rdb, "../sliding_window_counter.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to ScriptLoad: %w", err)
	}
	return &SWCImpl{
		rdb:       rdb,
		scriptSha: scriptSha,
		callCnt:   atomic.Int64{},
	}, nil
}

func (i *SWCImpl) Name() string {
	return "SlidingWindowCounter_Impl"
}

func (i *SWCImpl) Allow(ctx context.Context, key string, nowMs int64, config Config) (allowed bool, err error) {
	if err := config.Validate(); err != nil {
		return false, err
	}
	i.callCnt.Add(1)
	res, err := i.rdb.EvalSha(ctx, i.scriptSha, []string{key},
		config.RateSec, 1000, config.Cost, config.TTLMs).Result()
	if err != nil {
		return false, err
	}
	v, ok := res.(int64)
	if !ok {
		return false, fmt.Errorf("failed to convert rdb.EvalSha result to int64")
	}
	return v == int64(1), nil
}

func (i *SWCImpl) RedisCalls() int64 {
	return i.callCnt.Load()
}

func (i *SWCImpl) Teardown() {
	i.rdb.Close()
}
