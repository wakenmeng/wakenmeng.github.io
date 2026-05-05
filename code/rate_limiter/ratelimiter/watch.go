package ratelimiter

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type WatchImpl struct {
	rdb     redis.UniversalClient
	callCnt atomic.Int64
}

func InitWatchImpl(ctx context.Context, redisAddrs []string) (*WatchImpl, error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}
	return &WatchImpl{rdb: rdb}, nil
}

func (w *WatchImpl) Name() string {
	return "WatchMultiExec"
}

func (w *WatchImpl) Teardown() {
	w.rdb.Close()
}

func (w *WatchImpl) watchAllow(ctx context.Context, key string,
	nowMs, burst, rateSec, cost, ttlMs int64) (allowed bool, err error) {
	w.callCnt.Add(1)
	err = w.rdb.Watch(ctx, func(tx *redis.Tx) error {
		w.callCnt.Add(1)
		vals, err := tx.HMGet(ctx, key, "tokens", "ts").Result()
		if err != nil {
			return err
		}
		if len(vals) != 2 {
			return fmt.Errorf("failed to rdb.HMGET vals with length of 2, got: %v", vals)
		}
		tokens := burst
		if vals[0] != nil {
			tokens, err = strconv.ParseInt(vals[0].(string), 10, 64)
			if err != nil {
				return err
			}
		}
		ts := nowMs
		if vals[1] != nil {
			ts, err = strconv.ParseInt(vals[1].(string), 10, 64)
			if err != nil {
				return err
			}
		}

		delta := max(0, nowMs-ts)
		refill := delta * rateSec / 1000
		tokens = min(tokens+refill, burst)
		allowed = tokens >= cost

		w.callCnt.Add(1)
		_, err = tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
			if allowed {
				p.HMSet(ctx, key, "tokens", max(tokens-cost, 0), "ts", nowMs)
			}
			p.PExpire(ctx, key, time.Duration(ttlMs)*time.Millisecond)
			return nil
		})

		return err
	}, key)
	return allowed, err
}

func (w *WatchImpl) Allow(ctx context.Context, key string,
	nowMs int64, conf Config) (bool, error) {
	if err := conf.Validate(); err != nil {
		return false, err
	}
	for range 20 {
		allowed, err := w.watchAllow(ctx, key, nowMs, conf.Burst, conf.RateSec, conf.Cost, conf.TTLMs)
		if err == nil {
			return allowed, nil
		}
		if err != redis.TxFailedErr {
			return false, fmt.Errorf("failed to redis watch: %w", err)
		}
	}
	return false, fmt.Errorf("watch retries exhausted")
}

func (w *WatchImpl) RedisCalls() int64 {
	return w.callCnt.Load()
}
