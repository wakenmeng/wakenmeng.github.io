package ratelimiter

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type LuaImpl struct {
	rdb       *redis.Client
	scriptSha string
	callCnt   atomic.Int64
}

func InitLuaImpl(ctx context.Context, redisAddr string) (*LuaImpl, error) {
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to call redis.Ping: %w", err)
	}
	bts, err := os.ReadFile("../tokenbucket.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to read ../tokenbucket.lua: %w", err)
	}
	scriptSha, err := rdb.ScriptLoad(ctx, string(bts)).Result()
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
	l.callCnt.Add(1)
	res, err := l.rdb.EvalSha(ctx, l.scriptSha, []string{key}, nowMs, conf.RateSec, conf.Burst, conf.Cost, conf.TtlMs).Result()
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
