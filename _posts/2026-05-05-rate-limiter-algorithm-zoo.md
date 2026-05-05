---
title: "Rate Limiter Algorithm Zoo: GCRA, Sliding Window, and the Off-by-One Tax"
date: 2026-05-05
author: Waken
tags: [redis, lua, backend, ratelimit, golang, benchmark, distributed, gcra]
toc: true
---

The previous two posts in this series covered [token bucket basics](/posts/benchmarking-token-bucket-rate-limiters/) and [scaling to 1.7M ops/sec](/posts/scaling-token-bucket-to-1m-ops/). Both stayed inside one algorithm family. This post goes wider: **GCRA** and **Sliding Window Counter** — two industry-standard alternatives, with derivations and some tricky off-by-one bugs each one comes with.
<!-- more -->

## Where Token Bucket Sits in the Algorithm Space

Every rate limiter answers the same question: *over some time window, are we under budget?* Algorithms differ on three axes:

1. **How is "the time window" expressed?** Continuous (token bucket, GCRA) vs. fixed-window grid vs. sliding window
2. **How is "consumption so far" stored?** Counters, timestamps, or token balances
3. **What's the trade between accuracy and state size?**

| Algorithm | State / key | Accuracy | Burst param | Typical use |
|---|---|---|---|---|
| Token Bucket | 2 fields (`tokens, ts`) | Exact | ✅ explicit | The default everywhere |
| **GCRA** | **1 field (`TAT`)** | Exact | ✅ implicit τ | redis-cell, AWS API GW, Stripe |
| Sliding Window Counter | 3 fields (`win, cur, prev`) | ~3% (or worse under bursts) | ❌ | Cloudflare, "N per W seconds" rules |
| Sliding Window Log | O(N) timestamps | Exact | ❌ | Login throttling, low-rate audit |

The first one we covered already. This post does GCRA and SWC, ending with the production decision map.

## Part 1 — GCRA: Token Bucket's Single-Field Twin

**[GCRA](https://en.wikipedia.org/wiki/Generic_cell_rate_algorithm)** (Generic Cell Rate Algorithm) comes from 1990s ATM networks (ITU-T I.371). The constraint there: hardware traffic shapers had to fit in tiny FPGAs. So every byte of state mattered. Token bucket's `(tokens, ts)` was too expensive — they wanted **one number**.

### The Insight

Token bucket asks "how many tokens are in the bucket *right now*?" GCRA flips the question: "**when would the next request *theoretically* arrive if traffic followed the rate exactly?**"

That single timestamp — **TAT** (Theoretical Arrival Time) — replaces both `tokens` and `ts`.

### The Algorithm (3 lines)

Given:
- `T = 1/rate` — emission interval (seconds per request)
- `τ = (burst − 1) × T` — tolerance (more on the `−1` later)

On each request at time `now`:

```
if now >= TAT - τ:
    accept
    TAT = max(now, TAT) + T
else:
    reject
    retry_after = TAT - τ - now
```

That's it. **Single field, one max, one comparison, one add.**

### Why It's Equivalent to Token Bucket

Think of `TAT` as "the future point in time when the bucket would be full again, assuming we keep accepting requests at the configured rate from now."

- After `n` accepts (starting from idle), `TAT = now + n×T`.
- We allow more if `TAT - now ≤ τ`, i.e., we haven't pulled too far into the future.
- Each accept advances `TAT` by `T`, "reserving" the next slot.
- Once the system idles past `TAT`, `max(now, TAT)` resets — the bucket "refills" without an explicit refill computation.

The math works out to *exactly* token bucket's behavior, with half the state.

### Verified Equivalence in TestOverGrant

3 seconds, `burst=10, rate=10/s`, theoretical max = 40:

| Impl | allowed | over_grant |
|------|---------|------------|
| LuaEvalSha (token bucket) | 40 | 0.00% |
| **GCRA** | **39** | **−2.50%** |

The −2.5% is boundary noise (the test ends at 3.0009s; the 30th refill might or might not land before context cancellation). Same noise band as token bucket itself in different runs.

### The Off-by-One Tax

The `(burst − 1)` in `τ` is not a typo. Getting this wrong cost me an evening. Here's the bestiary I produced before landing on the right combination:

| τ formula | Comparison | Burst observed | Total accepts (3s test) |
|-----------|-----------|---------------:|---------:|
| `burst × T` | `>=` | 11 | 41 ❌ |
| `burst × T` | `>` | 10 | 41 ❌ |
| `(burst−1) × T` | `>` | 9 | 39 ❌ |
| **`(burst−1) × T`** | **`>=`** | **10** | **40** ✓ |

**Only one of four combinations matches token bucket's behavior.** The others are technically rate-correct in steady state, but in any finite test window, they over- or under-grant by exactly 1.

The reason: at the boundary `now == TAT − τ`, both `>=` and `>` are mathematically valid choices, but they imply different conventions about whether the burst includes the "current" slot. Different industrial implementations pick differently:

- **redis-cell** uses `>=` and exposes a `max_burst` parameter that means "additional burst beyond 1" — i.e., `max_burst = burst − 1`. Their docs read: *"max 16 actions per period, with a max burst of 15."*
- **AWS API Gateway** documentation describes burst as "additional capacity beyond the steady rate" — same convention.
- **Stripe's API rate limiter** uses GCRA variant with similar `−1` semantics.

If you're writing your own GCRA, **document which convention you picked** and stick to it. Mixing conventions across services that share a rate limit is how you get "but the dev environment allowed it" tickets.

### Edge Cases: What if `burst = 0`?

`τ = (burst − 1) × T = −T` (negative). The check becomes `now >= TAT + T`. Trace what happens at rate=10/s:

```
init: TAT=0, T=100ms
req at t=0:    0 >= 100?  No. reject.
req at t=100:  100 >= 100? Yes. TAT = 200.
req at t=200:  200 >= 300? No. reject.
req at t=300:  300 >= 300? Yes. TAT = 400.
```

**Effective rate: half the configured rate.** A silent half-throttle, undetectable from "is it working?" smoke tests but visible as p99 latency spikes in production.

This is a class of bug worth a name: **"the (N-1) tax."** Any formula with `(N − 1)` has a malignant edge case at `N = 0`. The fix is always the same — input validation:

```go
func (c Config) Validate() error {
    if c.Burst < 1 {
        return fmt.Errorf("burst must be >= 1, got %d", c.Burst)
    }
    // ... rate, cost similarly
}
```

Plus the same guard in Lua (deep + shallow defense, in case someone bypasses the Go layer):

```lua
if burst < 1 or rate < 1 or cost < 1 then
  return redis.error_reply("invalid config: burst/rate/cost must be >= 1")
end
```

### Performance: GCRA vs Token Bucket

Same hardware, single-node Redis, `con_64`, `per_user`:

| Impl | ns/op | redis_calls/req |
|------|------:|----------------:|
| LuaEvalSha (token bucket) | 21,506 | 1.000 |
| LuaServerTime (token bucket, server time) | 23,950 | 1.000 |
| **GCRA** | **21,523** | **1.000** |

Same speed as Lua-server-time token bucket, **half the state**. In production, the wins compound:
- 50% less Redis memory per active key (1M keys × ~30 bytes saved = ~30MB)
- 1 less `HSET` field per request (already factored into the ns/op above)
- Simpler atomicity reasoning (one field vs two)

Why isn't GCRA dominant if it's strictly better? Because token bucket is **easier to explain** ("a bucket of tokens"). And the state savings only matter at extreme scale where someone has time to argue for them.

## Part 2 — Sliding Window Counter

### The Fixed-Window Problem

The naive fixed-window approach: one counter per key per period. Reset to 0 on period boundary.

```
limit: 100 / minute

11:59:59 → 99 requests accepted (counter just under cap)
12:00:01 → 100 requests accepted (fresh counter)
```

**199 requests in 2 seconds, both windows technically compliant.** Whatever you were trying to protect just took 2× the configured rate.

### The Approximation

SWC keeps **two counters**: current window and previous window. At time `now` within the current window, the estimate is:

```
estimated = current + previous × (1 − elapsed_in_current_window / window_size)
```

**Intuition**: if I'm 30% into the current minute, then the most recent 70% of the previous minute is still inside my "1-minute sliding window." Assuming the previous minute's requests were uniformly distributed, 70% of its count "still counts."

**State**: 3 fields (`window_start, current, previous`). O(1).

**Accuracy**: depends on the uniform-distribution assumption. Cloudflare's [original analysis](https://blog.cloudflare.com/counting-things-a-lot-of-different-things/) measured 0.003% error rate on production traffic.

### What SWC Cannot Express

Token bucket and GCRA both let you say *"average 100/sec, but allow bursts up to 1000."* SWC has no separate burst parameter — it can only say *"≤ N requests per W-second window."* If you need explicit burst tolerance, SWC is the wrong tool.

This is why API rate limits (which often want both an average rate *and* a burst capacity) use GCRA, while UI throttling and login-attempt limits (which just want "≤ N per W seconds") use SWC.

### Benchmark Surprise: 13% Error

Same `TestOverGrant` setup (`limit=10/sec, 3 seconds`):

| Impl | allowed | "expected" |
|------|--------:|-----------:|
| Token Bucket | 40 | 40 |
| GCRA | 39 | 40 |
| **SWC** | **34** | **30** *(SWC's own theoretical max)* |

**Wait — SWC over-granted by ~13%, far more than the 0.003% Cloudflare promised.** Why?

The answer is in the assumption. SWC assumes the previous window's requests were **uniformly distributed**. In production, that's roughly true. In a hot-key benchmark with 64 goroutines hammering one key, requests cluster at boundaries:

- At t=0, all 64 goroutines fire near-simultaneously
- The current window fills its budget instantly
- Then they wait
- At the window boundary, they fire again

This violates the uniform assumption. The math gives the algorithm a "free pass" on the first window (no prior history → `prev = 0` → no penalty), letting the full burst through. Subsequent windows penalize correctly, but the first window's freebies stay in the count.

Concretely:
- Partial first window: 10 accepts (no `prev` penalty)
- Full window 2: 9 accepts (`prev=10` weighting kicks in)
- Full window 3: 9 accepts
- Partial last window: ~6 accepts

`10 + 9 + 9 + 6 = 34`, matching the test result almost exactly.

**The lesson is not "SWC is broken."** The lesson is: **algorithm accuracy claims always have an assumed traffic model.** Production traffic is closer to Poisson; benchmarks with synchronized goroutines are not. When evaluating, run your own workload, not a synthetic worst-case.

## Part 3 — The Decision Map

After implementing all four, here's the flowchart I'd hand a colleague:

```
Need explicit burst parameter? (e.g., "100/sec average, 1000 burst")
├── Yes → 
│   Need to save Redis memory at scale?
│   ├── Yes → GCRA
│   └── No  → Token Bucket (more readable)
│
└── No →
    Need exact precision (compliance, billing)?
    ├── Yes → Sliding Window Log (only if rate × window_size × active_keys is small)
    └── No  → Sliding Window Counter (default)
```

The four bands of typical use:

| Use case | Algorithm | Why |
|----------|-----------|-----|
| Public API rate limit | GCRA | Burst + state efficiency |
| Internal service rate limit | Token Bucket | Readability wins when scale doesn't matter |
| Login throttle ("5 per hour") | SWC | No burst needed, ratio semantics |
| Compliance audit ("exactly 100/day") | SW Log | Precision required, low rate |

## Meta-Lesson (one industrial law worth keeping)

**Any formula with `(N − 1)` has an edge case at `N = 0`.** It will compile, run, and silently misbehave. The fix is always the same: input validation at the boundary, with a clear error message. Found this in GCRA's τ; same shape exists in `cap − 1`, `len(arr) − 1`, `now − timeout`. **Treat `(N-1)` as a flag for a precondition.**

## What's Not Covered

**Sliding Window Log** — implemented mentally, skipped in code. The reason: the use case (low-rate, exact-count compliance) is narrow enough that any team needing it should look at it on its own merits, not as part of a comparison. Putting it in the benchmark would tell a misleading story; its O(N) state would dominate every metric without that being the point.

## What's Next

The final piece of this series is **deployment + observation**. Everything so far has been benchmarks on my laptop. The next post:

1. Multi-node Go service running all 7 implementations behind a real load generator (k6 or similar)
2. Prometheus scraping per-impl metrics (`redis_calls_total`, `allowed_total`, `rejected_total`, `errors_total`)
3. Grafana dashboards: p99, redis CPU, hot-shard detection
4. A failure injection: kill a Redis shard mid-load, see which algorithms degrade gracefully and which fall over

The "1.7M ops/sec" number from the previous post means nothing until I can show it surviving a shard failure. That's the post that earns the series title.

*All benchmarks on Apple M3 Pro, Redis 7, Go 1.25, single-node Redis. Source: [code/rate_limiter](https://github.com/wakenmeng/wakenmeng.github.io/tree/main/code/rate_limiter). This is part 3 of a 4-part series on building a production-grade distributed rate limiter from scratch.*
