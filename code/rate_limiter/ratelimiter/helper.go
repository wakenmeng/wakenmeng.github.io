package ratelimiter

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func newRedisClient(ctx context.Context, addrs []string) (cli redis.UniversalClient, err error) {
	cli = redis.NewUniversalClient(&redis.UniversalOptions{Addrs: addrs})
	if err := cli.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to call redis.Ping: %w", err)
	}
	return cli, err
}

func redisLoadScript(ctx context.Context, rdb redis.UniversalClient, filename string) (scriptSha string, err error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("failed to read %s err:%w", filename, err)
	}
	src := string(bs)
	scriptSha = redis.NewScript(src).Hash()

	switch c := rdb.(type) {
	case *redis.Client:
		_, err = rdb.ScriptLoad(ctx, src).Result()
	case *redis.ClusterClient:
		err = c.ForEachShard(ctx, func(ctx context.Context, client *redis.Client) error {
			_, err := client.ScriptLoad(ctx, src).Result()
			return err
		})
	}
	if err != nil {
		return "", fmt.Errorf("failed redis.ScriptLoad: %w", err)
	}
	return scriptSha, nil
}
