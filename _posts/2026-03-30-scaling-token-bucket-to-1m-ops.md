---
title: "Scaling a Token Bucket Rate Limiter to 1M ops/sec"
date: 2026-03-30
author: Waken
tags: [redis, lua, backend, ratelimit, golang, benchmark, distributed]
toc: true
---

Last week I [benchmarked three token-bucket implementations](/posts/benchmarking-token-bucket-rate-limiters) and crowned Lua+EvalSha the winner at ~66K ops/sec on a single Redis node. That was the floor. This post walks from there to **1.7M ops/sec** on the same laptop — and the bugs I broke along the way.
<!-- more -->

## The 1M Journey

Two levers were left after the first post:

1. **Shard the state.** Move from a single Redis to a Redis Cluster so independent keys spread across shards.
2. **Cut Redis out of the hot path.** Add a local token cache that batch-borrows from Redis; most requests never touch the network.

End-state headline numbers (Apple M3 Pro, 3-master/3-replica Redis Cluster, `burst=1000, rate=500/s`, 256 goroutines, `con_256`):

| Impl | Scenario | ns/op | ops/sec | redis_calls/req |
|------|---------|------:|--------:|----------------:|
| LuaEvalSha (cluster) | per_user | 5,878 | 170K | 1.000 |
| LuaServerTime (cluster) | per_user | 6,091 | 164K | 1.000 |
| **TwoTier (cluster)** | **per_user** | **588** | **1.7M** | **0.09** |
| **TwoTier (cluster)** | **hot_key** | **119** | **8.4M** | **0.0001** |

TwoTier wins by **~10×** throughput on `per_user`, and (after a second round of fixes) also wins on `hot_key` — down to ~120 ns per request because the failure path never touches Redis. Getting there was where most of the bugs lived.

## The Two-Tier Design

```
                ┌──────────────┐
  Allow() ─▶    │ local bucket │  ── atomic.Int64 tokens (milli-tokens)
                └───────┬──────┘
                        │  miss
                        ▼
                ┌──────────────┐  EvalSha borrow(N)
                │ per-key mutex│ ───────────────────▶  Redis
                └──────────────┘                        (source of truth)
```

**Fast path (atomic):** `tokens.Add(-1000) >= 0`. No lock, no RTT. If positive, allow.

**Slow path (mutex):** on miss, a single goroutine per key holds a lock, double-checks the counter, then asks Redis for `batchSize` tokens at once via a modified Lua (`borrow` instead of `allow`). The borrowed tokens become local capacity; subsequent requests drain them atomically.

Key code: [twotier.go](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/ratelimiter/twotier.go)

```go
func (t *TwoTierImpl) Allow(ctx context.Context, key string, nowMs int64, config Config) (bool, error) {
    lbAny, ok := t.local.Load(key)
    if !ok {
        lbAny, _ = t.local.LoadOrStore(key, &localBucket{})
    }
    lb := lbAny.(*localBucket)

    // fast path: lock-free
    if lb.tokens.Add(-1000) >= 0 { return true, nil }
    lb.tokens.Add(1000)

    // short-circuit: Redis told us when the next token will be ready
    if nowMs < lb.nextRetryMs.Load() { return false, nil }

    lb.mu.Lock()
    defer lb.mu.Unlock()

    // double-check inside the lock — world may have changed while we waited
    if nowMs < lb.nextRetryMs.Load() { return false, nil }
    if lb.tokens.Add(-1000) >= 0 { return true, nil }
    lb.tokens.Add(1000)

    t.callCnt.Add(1)
    res, err := t.remote.EvalSha(ctx, t.scriptSha, []string{key},
        t.batchSize, config.RateSec, config.Burst, config.TTLMs).Result()
    if err != nil { return false, err }
    arr := res.([]any)
    borrowed, retryMs := arr[0].(int64), arr[1].(int64)

    if borrowed < 1000 {
        lb.tokens.Add(borrowed)             // accumulate fractional leftover
        lb.nextRetryMs.Store(nowMs + retryMs)
        return false, nil
    }
    lb.tokens.Add(borrowed - 1000)
    return true, nil
}
```

The Lua script — [tokenbucket_servertime_borrow.lua](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/tokenbucket_servertime_borrow.lua) — is nearly identical to the `allow` version, except it returns `{borrowed, retry_ms}` instead of a boolean: `borrowed = min(tokens, need)` milli-tokens, plus the estimated ms until another token is available. The caller takes what it can get, and uses the estimate to avoid hitting Redis during the cooldown.

Four things worth pausing on:

- **`Load` first, then `LoadOrStore` on miss.** `LoadOrStore` always constructs its value argument, even if the key exists — one alloc per request on the hot path. `Load` does a lookup only.
- **Double-check inside the mutex** prevents the thundering herd: without it, every goroutine that saw an empty bucket would serialize behind the lock and each issue its own `EvalSha`. With it, only the first borrow hits Redis; the rest find tokens already refilled and skip the RTT.
- **`nextRetryMs` short-circuits rejections.** When Redis tells us it's empty for another 200 ms, we spend that 200 ms rejecting locally at ~100 ns/op — no lock, no RTT. This is what takes `hot_key` from 120,000 ns/op to 120 ns/op.
- **`batchSize` is the knob.** Bigger batch → fewer Redis calls but more local over-grant window. I used 100 for the benchmarks.

That's the sketch. The rest of this post is about what broke.

## Pitfalls & Debugging

### 1. The float-truncation leak (`allowed=10` vs `allowed=40`)

The first symptom: `TestOverGrant` with `burst=10, rate=10/s, 3 seconds` passed for every implementation except TwoTier. LuaServerTime saw `allowed=40` (the theoretical max); TwoTier saw `allowed=10` — exactly the burst, with no refill at all.

I spent an hour hunting the bug in Go before realizing the leak was in Lua.

The `borrow` script does this:

```lua
local need = tonumber(ARGV[1])          -- e.g. 100 (batch size)
local rate = tonumber(ARGV[2])          -- e.g. 10/s
...
local refill = (delta * rate) / 1000.0  -- ms → tokens. Fractional!
tokens = math.min(burst, tokens + refill)

local borrowed = math.min(tokens, need)
return borrowed                         -- e.g. 0.35
```

Redis's Lua runtime returns floats, but `go-redis` converts them to `int64` by **truncation**. `0.35` → `0`. Every borrow call with a fractional result silently lost tokens. Over 3 seconds at 10 tokens/sec, the bucket was supposed to refill 30 tokens; instead it refilled ~0.

LuaServerTime avoided this because it returns an *integer* pair `{allowed, retry_after_ms}` — the fractional tokens stay inside Redis, never crossing the language boundary.

**The fix: scale everything to milli-tokens.** 1 token = 1000 milli-tokens. Rate, burst, cost, and all returns are integers. [tokenbucket_servertime_borrow.lua](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/tokenbucket_servertime_borrow.lua#L7-L10):

```lua
local need  = tonumber(ARGV[1]) * 1000
local rate  = tonumber(ARGV[2]) * 1000
local burst = tonumber(ARGV[3]) * 1000
```

On the Go side, `lb.tokens` is an `atomic.Int64` of milli-tokens, and every `Add` uses `1000` as the unit ([twotier.go](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/ratelimiter/twotier.go)). One request costs 1000 milli-tokens; `borrowed` comes back in milli-tokens; the leftover after a partial borrow is `lb.tokens.Add(borrowed)` — milli-tokens accumulate locally and eventually round up to a whole request.

After the fix: `allowed=40`, exactly on target.

**Lesson:** any time a calculation crosses a language/protocol boundary, check the serialization contract. "Lua returns a number" is not the same as "Lua returns an int64." Scaled integer arithmetic is the standard fix — the same pattern that powers currency libraries (cents instead of dollars).

### 2. The hot_key story: from 8K to 8M ops/sec

The initial benchmark was embarrassing:

| Impl | hot_key con_64 ns/op |
|------|--------:|
| LuaEvalSha | 17,380 (~57K ops/sec) |
| TwoTier (v1) | 120,000 (~8K ops/sec) |

The whole point of local caching was to *avoid* Redis. Instead, TwoTier was **7× slower**.

**Why.** On `hot_key` every goroutine hits the same `localBucket`. The bucket empties constantly at `rate=500/s`, so every request falls through to the mutex. What plain Lua serializes at Redis (single-threaded, one RTT, ~20µs), TwoTier v1 serializes at a Go mutex — and *then still does the RTT*. Worse serialization, same RTT.

But here's the kicker: most of those queued goroutines are going to be **rejected** anyway (the bucket is empty, the borrow returns 0). If we already know rejection is the answer, the Redis round-trip is pure waste.

**The fix: short-circuit rejections.** Let Redis tell us when a retry would be worth it, cache that deadline locally, and reject without a lock or an RTT until the deadline passes. [twotier.go](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/ratelimiter/twotier.go):

```go
// fast path
if lb.tokens.Add(-1000) >= 0 { return true, nil }
lb.tokens.Add(1000)

// short-circuit: if we know Redis is empty too, reject locally
if nowMs < lb.nextRetryMs.Load() {
    return false, nil
}

lb.mu.Lock()
// ... borrow; if borrowed < 1000, compute and store nextRetryMs from Redis's reply
```

The Lua script got an extra return value, `retry_ms` — "when Redis will next have at least 1 token available" — computed on the server side where all the state is ([tokenbucket_servertime_borrow.lua](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/tokenbucket_servertime_borrow.lua)):

```lua
local retry_ms = 0
if borrowed < 1000 then
  retry_ms = math.ceil((1000 - tokens) / rate * 1000)
  if retry_ms < 1 then retry_ms = 1 end
end
return {borrowed, retry_ms}
```

Why not compute `retry_ms` in Go? Because Go only knows `rate`. Integer-divide `1000 / rate` at high rates (>1000/s) gives 0, and the cooldown collapses to "always retry" — back to stampeding Redis. Lua has floats and `math.ceil`; it can give us "at least 1 ms" cleanly, and it also has the *actual* current token count to base the estimate on.

**The bug in the fix.** First attempt of the short-circuit checked `nextRetryMs` only *before* `lb.mu.Lock()`. Benchmark result at `con_16` and `con_64`:

| hot_key | v1 (mutex only) | v2 (pre-mutex short-circuit) |
|---------|---:|---:|
| con_1   | 120K ns | **82 ns** |
| **con_16** | 120K ns | **118,090 ns** ❌ |
| **con_64** | 120K ns | **114,635 ns** ❌ |
| con_256 | 120K ns | 10,136 ns |

`con_1` dropped to 82 ns — perfect. `con_256` improved 12×. But `con_16` and `con_64` stayed stuck at 120K ns, and `redis_calls/req` was still 0.9 — almost every request was hitting Redis.

The reason: goroutines that arrived *together* all passed the pre-mutex `nextRetryMs` check when it was still zero, and queued on the mutex. The first one borrowed, set `nextRetryMs`, and released. Each subsequent goroutine had already committed to the slow path — the pre-mutex check was in the past. They all took their turn borrowing.

**Fix #2: re-check after acquiring the lock.** Classic double-check:

```go
lb.mu.Lock()
defer lb.mu.Unlock()

if nowMs < lb.nextRetryMs.Load() {  // re-check, state may have changed
    return false, nil
}
if lb.tokens.Add(-1000) >= 0 { return true, nil }
lb.tokens.Add(1000)
// only the winner borrows
```

After this fix, all concurrencies converge:

| hot_key | final | redis_calls/req | vs plain Lua |
|---------|---:|---:|---:|
| con_1   | 118 ns | 0.0001 | **190× faster** |
| con_16  | 119 ns | 0.0001 | **100× faster** |
| con_64  | 116 ns | 0.0001 | **100× faster** |
| con_256 | 119 ns | 0.0001 | **100× faster** |

**Lesson:** any "fast path out of the slow path" needs to be re-verified once you're *in* the slow path. The world changed while you waited for the lock. This is the same shape as the `lb.tokens` double-check, applied one layer out.

**Meta-lesson:** local caching does help on hot keys — but only if the *rejection* path is local too. If rejections still require Redis, the cache just adds a serialization point. Cache the "no" as well as the "yes."

### 3. Local time vs. Redis time decoupling

Even after fixing the float leak, I was seeing `context deadline exceeded` errors under high concurrency — hundreds of them per test. The bucket drains locally in microseconds; a `EvalSha` refill takes milliseconds. During that millisecond gap, all 64 goroutines (at `con_64`) hit an empty bucket. Without a mutex, they'd all fire their own `EvalSha` — 64 RTTs for what should be 1.

This is what motivated the double-check-under-mutex in the first place. The ordering matters:

```go
if lb.tokens.Add(-1000) >= 0 { return true, nil }  // fast path
lb.tokens.Add(1000)

lb.mu.Lock()                                        // serialize
if lb.tokens.Add(-1000) >= 0 { return true, nil }  // double-check
lb.tokens.Add(1000)
// only the winning goroutine borrows
```

Without the double-check, every goroutine behind the lock still fires its own borrow. With it, the first goroutine borrows, returns 100 tokens to `lb.tokens`, and the next 99 goroutines exit on the fast path inside the lock. Redis load drops by `batchSize`×.

A related symptom: context cancellation errors. The test uses `context.WithTimeout`; when the context fires, goroutines queued behind the mutex eventually wake up, find `ctx.Err() != nil`, and return. That's expected — but I was counting them as failures. A bit of bookkeeping (check `ctx.Err()` first thing inside the lock, [twotier.go](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/ratelimiter/twotier.go)) cleaned up the noise.

**Lesson:** when your fast path is atomic and your slow path is a network call, the gap between them is where the stampede happens. Always double-check the atomic state after acquiring the lock.

### 4. `over_grant_% = 301687%` was a statistics bug

The correctness test for `per_user` showed stunning over-grant numbers after I scaled up:

```
TwoTier/per_user | allowed: 120810 | theoretical_max: 40 | over_grant: 301687%
```

TwoTier was not granting 3000× the budget. The test harness was computing the wrong theoretical max:

```go
theoreticalMax := defaultConf.Burst + defaultConf.RateSec*elapsed/1000
```

For `hot_key` this is right. For `per_user` with N distinct keys, each key gets its own budget — so the actual max is `N × (burst + rate × elapsed)`. The original formula was off by a factor of `N` (the number of goroutines, i.e. 64 or 256).

The fix added a `keyCount` function per scenario ([rate_test.go](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/bench/rate_test.go)):

```go
{"hot_key",  keyFn: ..., keyCount: func(gcnt int64) int64 { return 1 }},
{"per_user", keyFn: ..., keyCount: func(gcnt int64) int64 { return gcnt }},
```

Then `theoreticalMax = numKeys * (burst + rate*elapsed/1000)`.

After the fix, the number went *negative* — TwoTier was granting **below** the theoretical max because the batch-borrow model leaves unclaimed tokens sitting in Redis when the test ends. "Over-grant %" with a meaningful negative value is confusing, so I renamed it to **`util_%`** (saturation) — how much of the budget was actually used. Values below 100% are fine; the correctness invariant is "no implementation exceeds 100% by more than rounding."

**Lesson:** when a metric changes regime (single key → many keys), the definition often needs to change too. If your "over X" number can also mean "under Y," the metric's name is lying. Rename before you report.

### 5. Cluster `ScriptLoad` silently fails without `ForEachShard`

Redis Cluster with 3 masters. My first run threw `NOSCRIPT` errors from 2 of the 3 shards.

`rdb.ScriptLoad(ctx, script)` on a `redis.ClusterClient` sends the command to one node — in my case, whichever shard the internal dispatcher picked. The other two shards never received the script, so `EvalSha` failed for any key that hashed to them.

The fix is `ForEachShard`, which is exactly the tool the cluster client gives you for this situation — [twotier.go](https://github.com/wakenmeng/wakenmeng.github.io/blob/main/code/rate_limiter/ratelimiter/twotier.go):

```go
switch c := rdb.(type) {
case *redis.Client:
    _, err = rdb.ScriptLoad(ctx, script).Result()
case *redis.ClusterClient:
    err = c.ForEachShard(ctx, func(ctx context.Context, shard *redis.Client) error {
        _, err := shard.ScriptLoad(ctx, script).Result()
        return err
    })
}
```

**Lesson:** `UniversalClient` papers over single-node and cluster differences for command dispatch, but script loading is a per-shard state mutation. Whenever you're mutating node-local state (scripts, config, function registrations), iterate shards explicitly.

### 6. Amdahl's Law — and the benchmark that disappeared

An early run of the cluster benchmark showed a curious gap. Going from single-node to 3-shard cluster on `per_user`:

- Lua: 2.4× speedup — close to the 3× theoretical upper bound.
- LuaServerTime: only **1.8×**.

Same cluster, same test shape. Why the gap?

I had a tidy story: LuaServerTime calls `redis.call("TIME")` to get a server timestamp — a syscall that takes a few microseconds per request. That cost is **serial overhead**: it doesn't parallelize when you add shards. Plain Lua passes the timestamp in as an argument, no syscall. By Amdahl's Law,

```
speedup = 1 / ((1 - P) + P/N)
```

A 1.8× speedup at N=3 implies P ≈ 0.67 — i.e. about a third of LuaServerTime's per-request cost is the `TIME` syscall. Pretty clean.

**Then I re-ran with a longer benchtime.** Both implementations now showed ~2.88× cluster speedup, identical to two decimal places.

The "Amdahl gap" was a measurement artifact from a too-short `benchtime`. The first run's short elapsed time let startup / first-request variance dominate LuaServerTime's numbers. A longer run washed it out.

Keeping both the bad story and the good one in the post because the bad one was instructive:

1. **Amdahl's Law is still the right mental model.** When horizontal scaling delivers less than N×, the first question is always *what's the serial fraction?* The answer is often something small you forgot to price — a syscall, a single-threaded allocator, a cross-shard coordination hop. That intuition is right even when the specific story isn't.
2. **Never build a story from one benchmark run.** One run is a data point, not a signal. If the effect you're explaining disappears at `benchtime=10s`, you weren't measuring what you thought you were measuring.

So: the lesson I *thought* I was writing turned into a different lesson about measurement hygiene. The `TIME` syscall does have real cost — it's just not dominant enough to show up as a scaling gap in this setup.

## Headline Benchmark Data

Cluster mode (3 masters / 3 replicas), `con_256`:

```
# per_user (keys independent, budget = numKeys × burst+rate×t)
TwoTier/per_user/con_256-12        27238970      588 ns/op   0.092 redis_calls/req   ~98.5 util_%
LuaEvalSha/per_user/con_256-12       638846     5878 ns/op   1.000 redis_calls/req      —
LuaServerTime/per_user/con_256-12    648008     6091 ns/op   1.000 redis_calls/req      —
MultiCmd/per_user/con_256-12         270423    13488 ns/op   3.000 redis_calls/req     (races)
WatchMultiExec/per_user/con_256-12   201543    17807 ns/op   3.000 redis_calls/req      —

# hot_key (single shared key, budget = burst+rate×t)
TwoTier/hot_key/con_256-12        32284123      119 ns/op   0.0001 redis_calls/req   100.0 util_%
LuaEvalSha/hot_key/con_256-12       310562    11633 ns/op   1.000  redis_calls/req    99.6 util_%
LuaServerTime/hot_key/con_256-12    299128    12485 ns/op   1.000  redis_calls/req   100.0 util_%
```

- **TwoTier `per_user`** does ~1 Redis call per 11 requests (`0.09 = 1/11`) — a 10× Redis-load reduction vs plain Lua, plus 10× throughput from the local fast path.
- **TwoTier `hot_key`** is ~100× faster than plain Lua because rejections are served entirely from `lb.nextRetryMs` — no lock, no RTT. The "100% util" means the algorithm is giving out exactly the budget; the other 99.99% of ops are short-circuited rejections.
- **Plain Lua on cluster** gets ~2.88× speedup on `per_user` vs single-node (3-shard cluster ceiling is 3×). Good.
- **Single-node numbers** (for comparison, `per_user/con_256`): Lua 16,915 ns/op, LuaServerTime 17,515 ns/op, TwoTier 556 ns/op. TwoTier's local cache is doing most of the work; the Redis backend barely matters.

## What's Next

Two open fronts:

1. **Algorithm diversity.** Token bucket is one shape. GCRA (Generic Cell Rate Algorithm), sliding-window counter, and sliding-window log all have different trade-offs around burst-shaping and state size. Next post: implementing and benchmarking GCRA against token bucket.
2. **Actually deploying this.** Everything so far is a laptop benchmark. The next big step is a multi-node Go service behind a real load generator, with Prometheus/Grafana watching p99, Redis CPU, and hot-shard detection. The "1M ops/sec" number doesn't mean much until I can see it survive a shard failure and a cluster resize.

*All benchmarks on Apple M3 Pro, Redis 7, Go 1.25, local 3-master/3-replica cluster. Source: [code/rate_limiter](https://github.com/wakenmeng/wakenmeng.github.io/tree/main/code/rate_limiter).*
