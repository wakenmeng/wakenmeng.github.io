package bench

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	rl "waken.dev/blog/code/lua_redis/ratelimiter"
)

const REDIS_ADDR = "localhost:6379"

var defaultConf = rl.Config{
	Burst:   1000,
	RateSec: 500,
	Cost:    1,
	TtlMs:   1000 * 60 * 60 * 24, // 24 hrs
}

func BenchmarkRateLimiter(b *testing.B) {
	ctx := context.Background()
	impls, err := rl.InitAll(ctx, REDIS_ADDR)
	if err != nil {
		b.Fatal(err)
	}
	scenarios := []struct {
		name  string
		keyFn func(goroutineID int64) string
	}{
		{"hot_key", func(gid int64) string { return "rate:hot" }},
		{"per_user", func(gid int64) string { return fmt.Sprintf("rate:user:%d", gid) }},
	}
	concurrency := []int{1, 16, 64, 256}

	rdb := redis.NewClient(&redis.Options{Addr: REDIS_ADDR})
	defer rdb.Close()

	for _, im := range impls {
		for _, sc := range scenarios {
			for _, conc := range concurrency {
				b.Run(fmt.Sprintf("%s/%s/con_%d", im.Name(), sc.name, conc), func(b *testing.B) {
					_ = rdb.FlushDB(ctx).Err()

					var totalCnt, allowedCnt atomic.Int64
					callsBefore := im.RedisCalls()

					gidGen := atomic.Int64{}
					b.SetParallelism(conc)
					start := time.Now()
					b.RunParallel(func(pb *testing.PB) {
						gid := gidGen.Add(1)
						for pb.Next() {
							totalCnt.Add(1)
							ok, err := im.Allow(ctx, sc.keyFn(gid), time.Now().UnixMilli(), defaultConf)
							if err == nil && ok {
								allowedCnt.Add(1)
							}
						}
					})
					elapsed := time.Since(start)

					total := totalCnt.Load()
					allowed := allowedCnt.Load()
					callsDelta := im.RedisCalls() - callsBefore

					if total > 0 {
						b.ReportMetric(float64(callsDelta)/float64(total), "redis_calls/req")
					}

					theoreticalMax := defaultConf.Burst + defaultConf.RateSec*int64(elapsed.Seconds())
					if theoreticalMax > 0 {
						overGrant := float64(allowed-theoreticalMax) / float64(theoreticalMax) * 100
						b.ReportMetric(overGrant, "over_grant_%")
					}
					b.ReportMetric(float64(allowed), "allowed_total")
				})
			}
		}
		im.Teardown()
	}
}

func TestOverGrant(t *testing.T) {
	testSecs := 3
	conf := rl.Config{
		Burst:   10,
		RateSec: 10,
		Cost:    1,
		TtlMs:   60_000,
	}
	theoreticalMax := conf.Burst + conf.RateSec*int64(testSecs)
	conc := 64

	impls, err := rl.InitAll(context.Background(), REDIS_ADDR)
	if err != nil {
		t.Fatal(err)
	}

	rdb := redis.NewClient(&redis.Options{Addr: REDIS_ADDR})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}

	for _, im := range impls {
		key := fmt.Sprintf("test:hot:og:%s", im.Name())
		if err := rdb.Del(context.Background(), key).Err(); err != nil {
			t.Fatal(err)
		}
		var totalCnt, allowedCnt, errCnt atomic.Int64
		var wg sync.WaitGroup
		wg.Add(conc)
		ctx, cc := context.WithTimeout(context.Background(), time.Duration(testSecs)*time.Second)
		start := time.Now()
		for range conc {
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					totalCnt.Add(1)
					allowed, err := im.Allow(ctx, key, time.Now().UnixMilli(), conf)
					if err != nil {
						errCnt.Add(1)
						continue
					}
					if allowed {
						allowedCnt.Add(1)
					}
				}
			}()
		}
		wg.Wait()
		elapsed := time.Since(start)
		cc()

		t.Logf("test: %s | total: %d, allowed: %d, errors: %d, elapsed: %v",
			im.Name(), totalCnt.Load(), allowedCnt.Load(), errCnt.Load(), elapsed)
		t.Logf("  theoretical_max: %d, over_grant: %.2f%%",
			theoreticalMax, float64(allowedCnt.Load()-theoreticalMax)/float64(theoreticalMax)*100)
	}
}
