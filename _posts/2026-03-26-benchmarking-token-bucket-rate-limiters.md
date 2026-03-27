---
layout: post
title: "Benchmarking Token Bucket Rate Limiters"
date: 2026-03-26
author: Waken
tags: [redis, lua, backend, ratelimit, golang, benchmark]
toc: true
---

Implementing a distributed rate limiter with Redis? Three approaches: Lua scripts, multi-command client-side calculation, and WATCH optimistic locking. I benchmarked all three. Here's what I found.
<!-- more -->

## The Three Approaches

A token bucket rate limiter needs **read-compute-write** on shared state. If these steps aren't atomic, concurrent requests can read stale state and over-grant tokens.

I tested three ways to solve this:

**Lua EvalSha** — The entire token bucket logic runs as a Lua script on the Redis server. Since Redis is single-threaded, the script executes atomically. One round trip.

**Multi-Cmd (Client-Side Calculation)** — Read state with `HMGET`, calculate refill in Go, write back with `HMSET` + `EXPIRE`. Straightforward, but those three separate commands leave a window for concurrent reads to see stale state.

**WATCH + MULTI/EXEC (Optimistic Locking)** — Redis monitors the key with `WATCH`. If another client modifies the key before `EXEC`, the transaction aborts and we retry. Correct, but retries add up under contention.

| Approach | Atomicity | Round Trips |
|----------|-----------|-------------|
| **Lua EvalSha** | Atomic | 1 |
| **Multi-Cmd** | Not atomic | 3 |
| **WATCH + MULTI/EXEC** | Optimistic lock | 3+ |

## Benchmark Results

### Performance (hot_key, 64 goroutines)

| Impl | ns/op | redis_calls/req | allocs/op |
|------|-------|-----------------|-----------|
| **LuaEvalSha** | **20,658** | **1.0** | 14 |
| MultiCmd | 54,528 | 3.0 | 28 |
| WatchMultiExec | 1,166,537 | **50.7** | 1,070 |

Lua is **2.6x faster** than MultiCmd and **56x faster** than Watch under contention. The `redis_calls/req` column explains why Watch struggles — 50 Redis calls per request means ~16 retries on average.

### Correctness (burst=10, rate=10/s, 3 seconds, theoretical max = 40)

Speed is only half the story. A rate limiter that's fast but lets too many requests through is worse than useless. I ran a correctness test: with `burst=10` and `rate=10/s` over 3 seconds, at most 40 requests should be allowed.

| Impl | allowed | over_grant |
|------|---------|------------|
| **LuaEvalSha** | **40** | **0.00%** |
| MultiCmd | 123 | **207.50%** |
| WatchMultiExec | 38 | **-5.00%** |

- **Lua**: Exactly 40. Perfect enforcement.
- **MultiCmd**: 3x the limit. Race conditions cause massive over-granting.
- **Watch**: Correct, but 1526 retry errors reduce throughput.

### Bottom Line

| | Correctness | Throughput | Complexity |
|---|---|---|---|
| **Lua EvalSha** | exact | best | low |
| Multi-Cmd | over-grants (race) | medium | low |
| Watch | correct (but under-throughput) | worst under contention | high |

**Lua is the clear winner.** Read on for the details.

---

## How to Run Lua Scripts in Redis

If you haven't used Lua in Redis before, the idea is simple: you send a script to Redis, and Redis executes it atomically. **The entire script runs as a single operation** — no other command can interleave. This is what makes it perfect for read-compute-write patterns like rate limiting.

### EVAL

```
EVAL "return redis.call('SET', KEYS[1], ARGV[1])" 1 mykey myvalue
```

`EVAL` takes the script source, the number of keys, followed by keys and arguments.

### SCRIPT LOAD + EVALSHA

Sending the full script text on every request is wasteful. `SCRIPT LOAD` + `EVALSHA` solves this:

1. `SCRIPT LOAD` caches the script and returns its SHA1 hash
2. `EVALSHA` executes by SHA1 hash

```
> SCRIPT LOAD "return redis.call('SET', KEYS[1], ARGV[1])"
"a42059b356c875f0717db19a51f6aaa9161571a2"

> EVALSHA a42059b356c875f0717db19a51f6aaa9161571a2 1 mykey myvalue
OK
```

The SHA1 is deterministic. Same script content always produces the same hash. In Go with `go-redis`:

```go
sha, err := rdb.ScriptLoad(ctx, luaScript).Result()
// then use EvalSha for all subsequent calls
res, err := rdb.EvalSha(ctx, sha, []string{key}, args...).Result()
```

## Token Bucket in Lua

I covered the [token bucket algorithm in a previous post](/2022/07/27/rate-limiting-algorithms). The short version: a bucket holds tokens that refill at a constant rate. Each request consumes a token. When the bucket is empty, requests are rejected until tokens refill.

Here's the full Lua implementation. The client passes in the current timestamp, rate, burst capacity, cost per request, and a TTL for the key.

### The Lua Script

```lua
-- KEYS[1] = key
-- ARGV[1] = now_ms, ARGV[2] = rate_per_sec, ARGV[3] = burst
-- ARGV[4] = cost,   ARGV[5] = ttl_ms

local key   = KEYS[1]
local now   = tonumber(ARGV[1])
local rate  = tonumber(ARGV[2])
local burst = tonumber(ARGV[3])
local cost  = tonumber(ARGV[4])
local ttl   = tonumber(ARGV[5])

local tokens = redis.call("HGET", key, "tokens")
local ts     = redis.call("HGET", key, "ts")

if tokens == false or ts == false then
  tokens = burst
  ts     = now
else
  tokens = tonumber(tokens)
  ts     = tonumber(ts)
end

-- refill
local delta  = math.max(0, now - ts)
local refill = (delta / 1000.0) * rate
tokens = math.min(burst, tokens + refill)
ts     = math.max(now, ts)

local allowed = 0
local retry_after_ms = 0

if tokens >= cost then
  tokens  = tokens - cost
  allowed = 1
else
  local need = cost - tokens
  retry_after_ms = math.ceil((need / rate) * 1000.0)
end

redis.call("HSET", key, "tokens", tokens)
redis.call("HSET", key, "ts", ts)
if ttl ~= nil and ttl > 0 then
  redis.call("PEXPIRE", key, ttl)
end

return {allowed, retry_after_ms}
```

Note `ts = math.max(now, ts)`. This prevents timestamp regression when multiple clients send out-of-order timestamps. More on this in the [Lessons Learned](#lessons-learned) section.

### Go Caller

```go
func (l *LuaImpl) Allow(ctx context.Context, key string, nowMs int64, conf Config) (bool, error) {
    res, err := l.rdb.EvalSha(ctx, l.scriptSha, []string{key},
        nowMs, conf.RateSec, conf.Burst, conf.Cost, conf.TtlMs).Result()
    if err != nil {
        return false, err
    }
    arr := res.([]any)
    return arr[0] == int64(1), nil
}
```

## Implementation Details

Now let's look at the two alternative implementations and why they fall short.

### Multi-Cmd (Client-Side Calculation)

The most intuitive approach: read the state, do the math in Go, write it back.

```go
func (m *MultiCmdImpl) Allow(ctx context.Context, key string, nowMs int64, conf Config) (bool, error) {
    // 1. HMGET - read state
    vals, _ := m.rdb.HMGet(ctx, key, "tokens", "ts").Result()

    // 2. Calculate refill in Go (same math as Lua)
    delta  := max(0, nowMs - ts)
    refill := delta * conf.RateSec / 1000
    tokens  = min(tokens + refill, conf.Burst)
    allowed := tokens >= conf.Cost

    // 3. HMSET - write back
    m.rdb.HMSet(ctx, key, "tokens", max(tokens-conf.Cost, 0), "ts", nowMs)

    // 4. EXPIRE
    m.rdb.Expire(ctx, key, time.Duration(conf.TtlMs)*time.Millisecond)
    return allowed, nil
}
```

3 round trips, and the gap between `HMGET` and `HMSET` is where the trouble starts. Two goroutines can both read `tokens=1`, both decide to allow, and both write `tokens=0` — granting two requests when only one token was available.

### WATCH + MULTI/EXEC (Optimistic Locking)

Redis's answer to optimistic concurrency control. The idea: watch a key, read its value, compute, then execute a transaction. If anyone else modified the key in the meantime, Redis aborts the transaction and we retry.

```go
func (w *WatchImpl) Allow(ctx context.Context, key string, nowMs int64, conf Config) (bool, error) {
    for range 20 { // retry loop
        allowed, err := w.rdb.Watch(ctx, func(tx *redis.Tx) error {
            // HMGET under WATCH
            vals, _ := tx.HMGet(ctx, key, "tokens", "ts").Result()
            // ... same calculation as MultiCmd ...

            // TxPipelined = MULTI + commands + EXEC
            _, err := tx.TxPipelined(ctx, func(p redis.Pipeliner) error {
                if allowed {
                    p.HMSet(ctx, key, "tokens", newTokens, "ts", nowMs)
                }
                p.PExpire(ctx, key, ttl)
                return nil
            })
            return err
        }, key)
        if err == nil { return allowed, nil }
        if err != redis.TxFailedErr { return false, err }
        // TxFailedErr = someone else modified the key, retry
    }
    return false, fmt.Errorf("watch retries exhausted")
}
```

The retry loop is capped at 20 attempts. If contention is high enough that 20 retries aren't enough, the request fails. This keeps latency bounded but means some requests are dropped entirely.

## Full Benchmark Data

I tested two scenarios across four concurrency levels (1, 16, 64, 256 goroutines):

- **hot_key**: All goroutines share a single key — maximum contention
- **per_user**: Each goroutine gets its own key — zero contention, pure throughput

### hot_key (all goroutines share one key)

```
$ go test -bench='hot_key' -count=1 ./bench/

LuaEvalSha/hot_key/con_1-12      36139    34408 ns/op   1.00 redis_calls/req
LuaEvalSha/hot_key/con_16-12     69657    16942 ns/op   1.00 redis_calls/req
LuaEvalSha/hot_key/con_64-12     76624    20658 ns/op   1.00 redis_calls/req
LuaEvalSha/hot_key/con_256-12    61875    21949 ns/op   1.00 redis_calls/req

MultiCmd/hot_key/con_1-12         14787    83378 ns/op   3.00 redis_calls/req
MultiCmd/hot_key/con_16-12        22154    58020 ns/op   3.00 redis_calls/req
MultiCmd/hot_key/con_64-12        24199    54528 ns/op   3.00 redis_calls/req
MultiCmd/hot_key/con_256-12       25531    49091 ns/op   3.00 redis_calls/req

WatchMultiExec/hot_key/con_1-12    1762   688617 ns/op  18.44 redis_calls/req
WatchMultiExec/hot_key/con_16-12   1050  1227364 ns/op  50.56 redis_calls/req
WatchMultiExec/hot_key/con_64-12    895  1166537 ns/op  50.71 redis_calls/req
WatchMultiExec/hot_key/con_256-12   828  1300218 ns/op  51.51 redis_calls/req
```

Watch's `redis_calls/req` of 50.7 means each request triggered ~16 retries on average.

### per_user (no contention)

```
LuaEvalSha/per_user/con_64-12      52089   20900 ns/op   1.00 redis_calls/req
MultiCmd/per_user/con_64-12         20330   55538 ns/op   3.00 redis_calls/req
WatchMultiExec/per_user/con_64-12   17378   77726 ns/op   3.00 redis_calls/req
```

Without contention, Watch drops to 3.0 redis_calls/req (no retries needed) and performs similarly to MultiCmd. This confirms that Watch's poor hot_key performance is entirely caused by retry storms, not inherent overhead. The remaining gap between Lua and the others is purely round-trip count: 1 RTT vs 3 RTTs.

### TestOverGrant

The correctness test uses a tight rate limit (`burst=10, rate=10/s, cost=1`) and throws 64 concurrent goroutines at a single key for 3 seconds. The theoretical maximum allowed requests is `10 + 10*3 = 40`.

```
$ go test -run=TestOverGrant -count=1 -v ./bench/

=== RUN   TestOverGrant
  LuaEvalSha    | total: 138592  allowed: 40   errors: 0
                  theoretical_max: 40, over_grant: 0.00%
  MultiCmd      | total: 55134   allowed: 123  errors: 42
                  theoretical_max: 40, over_grant: 207.50%
  WatchMultiExec| total: 2907    allowed: 38   errors: 1526
                  theoretical_max: 40, over_grant: -5.00%
--- PASS: TestOverGrant (9.03s)
```

## Lessons Learned

These are bugs I hit during development. Each one is subtle enough that the implementation looks correct until you throw concurrency at it.

### 1. Timestamp Regression Bug

The initial Lua script had `ts = now` — unconditionally setting the timestamp to whatever the client passed in. This seems fine until you realize that with multiple concurrent clients, `EvalSha` calls arrive at Redis in a different order than the clients generated their timestamps:

```
Goroutine B: nowMs=1001 → processed first → writes ts=1001
Goroutine A: nowMs=1000 → processed second → writes ts=1000  (regression!)
Goroutine C: nowMs=1002 → sees ts=1000, delta=2ms instead of 1ms (inflated refill)
```

Fix: `ts = math.max(now, ts)`. This eliminated over-granting in the Lua implementation from ~100% down to 0%.

### 2. MultiCmd Write-Back Strategy

My first version of MultiCmd only wrote back (`HSET`) when the request was allowed. When denied, it skipped the write — why update state if nothing changed? But this leaves the timestamp stale. The next request sees a larger `delta`, gets more refill tokens, and is more likely to be allowed. Combined with the existing race conditions, this amplified over-granting from ~200% to ~2400%.

Fix: Always write back tokens and timestamp, regardless of allow/deny.

### 3. Watch Conflict-Throughput Tradeoff

Watch guarantees correctness (no over-granting), but the cost becomes visible under contention. The `redis_calls/req` metric tells the story:

| Concurrency | redis_calls/req |
|-------------|-----------------|
| 1 | 18.4 |
| 16 | 50.6 |
| 64 | 50.7 |
| 256 | 51.5 |

At 64+ goroutines on one key, each request averages ~50 Redis calls (about 16 retry attempts). Each retry does WATCH + HMGET + EXEC, all for a single allow/deny decision. The throughput collapses not because any individual operation is slow, but because everyone keeps stepping on each other's toes.

## What's Next

The Lua approach wins on both performance and correctness, but there's a catch we haven't fully explored: the client supplies the `now` timestamp. In a distributed system with multiple servers, clock skew and network latency can cause the timestamps to arrive out of order — which is exactly the bug we fixed with `math.max(now, ts)`. But is that fix sufficient? In the next post, I'll dig into clock skew, network delay, and whether using server-side `redis.call('TIME')` is a better approach.

*All benchmarks run on Apple M3 Pro, local Redis 7, Go 1.25. Source code: [code/rate_limiter](https://github.com/user/blog/tree/main/code/rate_limiter)*
