package bench

import (
	"context"

	"github.com/redis/go-redis/v9"
)

func flushAll(ctx context.Context, rdb redis.UniversalClient) error {
	switch c := rdb.(type) {
	case *redis.ClusterClient:
		return c.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error {
			return shard.FlushAll(ctx).Err()
		})
	default:
		return c.FlushDB(ctx).Err()
	}
}
