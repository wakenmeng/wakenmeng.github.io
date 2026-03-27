package ratelimiter

import "context"

type Config struct {
	Burst   int64
	RateSec int64
	Cost    int64
	TtlMs   int64
}

type Impl interface {
	Name() string
	Allow(ctx context.Context, key string, nowMs int64, config Config) (allowed bool, err error)
	RedisCalls() int64
	Teardown()
}

func InitAll(ctx context.Context, redisAddr string) ([]Impl, error) {
	luaI, err := InitLuaImpl(ctx, redisAddr)
	if err != nil {
		return nil, err
	}
	mulI, err := InitMultiCmdImpl(ctx, redisAddr)
	if err != nil {
		return nil, err
	}
	watI, err := InitWatchImpl(ctx, redisAddr)
	if err != nil {
		return nil, err
	}
	return []Impl{luaI, mulI, watI}, nil
}
