package ratelimiter

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type LuaImpl struct {
	rdb       redis.UniversalClient
	scriptSha string
	callCnt   atomic.Int64
}

func InitLuaImpl(ctx context.Context, redisAddrs []string) (*LuaImpl, error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}
	scriptSha, err := redisLoadScript(ctx, rdb, "../tokenbucket.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to ScriptLoad: %w", err)
	}

	return &LuaImpl{
		rdb:       rdb,
		scriptSha: scriptSha,
	}, nil
}

func (l *LuaImpl) Name() string {
	return "LuaEvalSha"
}

func (l *LuaImpl) Teardown() {
	l.rdb.Close()
}

func (l *LuaImpl) Allow(ctx context.Context, key string, nowMs int64, conf Config) (bool, error) {
	if err := conf.Validate(); err != nil {
		return false, err
	}
	l.callCnt.Add(1)
	res, err := l.rdb.EvalSha(ctx, l.scriptSha, []string{key}, nowMs, conf.RateSec, conf.Burst, conf.Cost, conf.TTLMs).Result()
	if err != nil {
		return false, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) != 2 {
		return false, fmt.Errorf("failed to convert rdb.EvalSha result to []any")
	}
	return arr[0] == int64(1), nil
}

func (l *LuaImpl) RedisCalls() int64 {
	return l.callCnt.Load()

}
