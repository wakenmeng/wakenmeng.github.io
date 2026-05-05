package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type MultiCmdImpl struct {
	rdb     redis.UniversalClient
	callCnt atomic.Int64
}

func InitMultiCmdImpl(ctx context.Context, redisAddrs []string) (*MultiCmdImpl, error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}
	return &MultiCmdImpl{rdb: rdb}, nil
}

func (m *MultiCmdImpl) Name() string {
	return "MultiCmd"
}

func (m *MultiCmdImpl) Teardown() {
	m.rdb.Close()
}

func (m *MultiCmdImpl) Allow(ctx context.Context, key string, nowMs int64, conf Config) (bool, error) {
	if err := conf.Validate(); err != nil {
		return false, err
	}
	m.callCnt.Add(1)
	vals, err := m.rdb.HMGet(ctx, key, "tokens", "ts").Result()
	if err != nil {
		return false, err
	}
	if len(vals) != 2 {
		return false, fmt.Errorf("failed to rdb.HMGET vals with length of 2, got: %v", vals)
	}
	tokens := conf.Burst
	if vals[0] != nil {
		tokens, err = strconv.ParseInt(vals[0].(string), 10, 64)
		if err != nil {
			return false, err
		}
	}
	ts := nowMs
	if vals[1] != nil {
		ts, err = strconv.ParseInt(vals[1].(string), 10, 64)
		if err != nil {
			return false, err
		}
	}

	delta := max(0, nowMs-ts)
	refill := delta * conf.RateSec / 1000
	tokens = min(tokens+refill, conf.Burst)
	allowed := tokens >= conf.Cost

	m.callCnt.Add(1)
	if err := m.rdb.HMSet(ctx, key, "tokens", max(tokens-conf.Cost, 0), "ts", nowMs).Err(); err != nil {
		return false, fmt.Errorf("failed to rdb.HMSet: %w", err)
	}

	m.callCnt.Add(1)
	if err := m.rdb.Expire(ctx, key, time.Duration(conf.TTLMs)*time.Millisecond).Err(); err != nil {
		return false, fmt.Errorf("failed to rdb.Expire: %w", err)
	}
	return allowed, nil
}

func (m *MultiCmdImpl) RedisCalls() int64 {
	return m.callCnt.Load()
}
