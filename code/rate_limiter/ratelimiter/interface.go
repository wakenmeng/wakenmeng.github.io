package ratelimiter

import (
	"context"
	"fmt"
)

type Config struct {
	Burst   int64
	RateSec int64
	Cost    int64
	TTLMs   int64
}

func (c Config) Validate() error {
	if c.Burst < 1 {
		return fmt.Errorf("invalid config: burst must be >= 1, got %d", c.Burst)
	}
	if c.RateSec < 1 {
		return fmt.Errorf("invalid config: rate must be >= 1, got %d", c.RateSec)
	}
	if c.Cost < 1 {
		return fmt.Errorf("invalid config: cost must be >= 1, got %d", c.Cost)
	}
	return nil
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

	gcraI, err := InitGCRAImpl(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}

	swcI, err := InitSWCImpl(ctx, redisAddrs)
	if err != nil {
		return nil, err
	}
	return []Impl{luaI, mulI, watI, luaSrvI, twoI, gcraI, swcI}, nil
}
