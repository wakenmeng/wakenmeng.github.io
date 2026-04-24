package ratelimiter

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func newRedisClient(ctx context.Context, addrs []string) (cli redis.UniversalClient, err error) {
	cli = redis.NewUniversalClient(&redis.UniversalOptions{Addrs: addrs})
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to call redis.Ping: %w", err)
	}
	return cli, err
}
