package ratelimiter

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type LuaServerTimeImpl struct {
	rdb       redis.UniversalClient
	scriptSha string
	callCnt   atomic.Int64
}

func InitLuaServerTimeImpl(ctx context.Context, redisAddrs []string) (impl *LuaServerTimeImpl, err error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	bts, err := os.ReadFile("../tokenbucket_servertime.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to read ../tokenbucket_servertime.lua: %w", err)
	}
	script := string(bts)
	scriptSha := redis.NewScript(script).Hash()
	switch c := rdb.(type) {
	case *redis.Client:
		_, err = rdb.ScriptLoad(ctx, script).Result()
	case *redis.ClusterClient:
		err = c.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error {
			_, err := shard.ScriptLoad(ctx, script).Result()
			return err
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to ScriptLoad: %w", err)
	}
	return &LuaServerTimeImpl{
		rdb:       rdb,
		scriptSha: scriptSha,
	}, nil
}

func (l *LuaServerTimeImpl) Name() string {
	return "LuaServerTimeImpl"
}

func (l *LuaServerTimeImpl) Teardown() {
	l.rdb.Close()
}

func (l *LuaServerTimeImpl) Allow(ctx context.Context, key string, _ int64, conf Config) (allowed bool, err error) {
	l.callCnt.Add(1)
	res, err := l.rdb.EvalSha(ctx, l.scriptSha, []string{key},
		conf.RateSec, conf.Burst, conf.Cost, conf.TTLMs).Result()
	if err != nil {
		return false, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) != 2 {
		return false, fmt.Errorf("failed to convert rdb.EvalSha result to []any")
	}
	return arr[0] == int64(1), nil

}

func (l *LuaServerTimeImpl) RedisCalls() int64 {
	return l.callCnt.Load()
}
