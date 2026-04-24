package ratelimiter

import "context"

type Config struct {
	Burst   int64
	RateSec int64
	Cost    int64
	TTLMs   int64
}

type Impl interface {
	Name() string
	Allow(ctx context.Context, key string, nowMs int64, config Config) (allowed bool, err error)
	RedisCalls() int64
	Teardown()
}

func InitAll(ctx context.Context, redisAddrs []string) ([]Impl, error) {
	luaI, err := InitLuaImpl(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}
	mulI, err := InitMultiCmdImpl(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}
	watI, err := InitWatchImpl(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}
	luaSrvI, err := InitLuaServerTimeImpl(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}

	twoI, err := InitTwoTierImpl(ctx, redisAddrs, 100)
	if err != nil {
		return nil, err
	}

	return []Impl{luaI, mulI, watI, luaSrvI, twoI}, nil
}
