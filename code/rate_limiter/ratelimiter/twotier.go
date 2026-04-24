package ratelimiter

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/redis/go-redis/v9"
)

type TwoTierImpl struct {
	local     sync.Map // key -> *localBucket
	remote    redis.UniversalClient
	scriptSha string
	callCnt   atomic.Int64
	batchSize int64
}

type localBucket struct {
	tokens      atomic.Int64 // milli_tokens
	mu          sync.Mutex
	nextRetryMs atomic.Int64 // next retry time
}

func InitTwoTierImpl(ctx context.Context, redisAddrs []string, batchSize int64) (impl *TwoTierImpl, err error) {
	rdb, err := newRedisClient(ctx, redisAddrs)
	if err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	bs, err := os.ReadFile("../tokenbucket_servertime_borrow.lua")
	if err != nil {
		return nil, fmt.Errorf("failed to read ../tokenbucket_servertime_borrow.lua: %w", err)
	}
	script := string(bs)
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
	return &TwoTierImpl{
		local:     sync.Map{},
		remote:    rdb,
		scriptSha: scriptSha,
		batchSize: batchSize,
	}, nil
}

func (t *TwoTierImpl) Allow(ctx context.Context, key string, nowMs int64, config Config) (allowed bool, err error) {
	// get from local
	lbAny, ok := t.local.Load(key)
	if !ok {
		lbAny, _ = t.local.LoadOrStore(key, &localBucket{})
	}
	lb := lbAny.(*localBucket)
	if lb.tokens.Add(-1000) >= 0 {
		return true, nil
	}
	lb.tokens.Add(1000)

	if nowMs < lb.nextRetryMs.Load() {
		return false, nil
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return false, err
	}

	if nowMs < lb.nextRetryMs.Load() {
		return false, nil
	}

	if lb.tokens.Add(-1000) >= 0 {
		return true, nil
	}
	lb.tokens.Add(1000)

	t.callCnt.Add(1)
	res, err := t.remote.EvalSha(ctx, t.scriptSha, []string{key},
		t.batchSize, config.RateSec, config.Burst, config.TTLMs).Result()
	if err != nil {
		return false, err
	}
	arr, ok := res.([]any)
	if !ok || len(arr) != 2 {
		return false, fmt.Errorf("failed to convert rdb.EvalSha result to []any")
	}
	borrowed, retry_ms := arr[0].(int64), arr[1].(int64)
	if borrowed < 1000 {
		lb.tokens.Add(borrowed)
		lb.nextRetryMs.Store(nowMs + retry_ms)
		return false, nil
	}
	lb.tokens.Add(borrowed - 1000)
	return true, nil
}

func (t *TwoTierImpl) Name() string {
	return "twotier_impl"
}

func (t *TwoTierImpl) RedisCalls() int64 {
	return t.callCnt.Load()
}

func (t *TwoTierImpl) Teardown() {
	t.remote.Close()
}
