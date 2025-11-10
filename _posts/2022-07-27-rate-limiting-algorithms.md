---
layout: post
title: "Rate Limiting: Leaky Bucket vs Token Bucket"
date: 2022-07-27
author: Waken
tags: [algorithms, rate-limiting, golang]
comments: true
---

Been looking into rate limiting for calling third-party APIs. Two main algorithms keep coming up: Leaky Bucket and Token Bucket. Here's what I learned.

<!-- more -->

## The Problem

Calling external APIs without rate limiting = bad time:
- Hit rate limits → errors
- Waste quota → costs money
- Bursts → overwhelm services

Need to smooth out requests.

## Leaky Bucket

Imagine a bucket with a hole:
- Requests go in at any rate
- Leak out at constant rate
- Bucket full? Drop requests

```go
type LeakyBucket struct {
    capacity  int
    remaining int
    leakRate  time.Duration
    lastLeak  time.Time
    mu        sync.Mutex
}

func (lb *LeakyBucket) Allow() bool {
    lb.mu.Lock()
    defer lb.mu.Unlock()

    // Leak based on time elapsed
    now := time.Now()
    elapsed := now.Sub(lb.lastLeak)
    leaked := int(elapsed / lb.leakRate)

    if leaked > 0 {
        lb.remaining += leaked
        if lb.remaining > lb.capacity {
            lb.remaining = lb.capacity
        }
        lb.lastLeak = now
    }

    if lb.remaining > 0 {
        lb.remaining--
        return true
    }
    return false
}
```

**Good for:** Smooth, constant output rate. Like streaming data or strict API rate limits.

## Token Bucket

Tokens refill at constant rate:
- Bucket starts with tokens
- Request consumes token
- Out of tokens? Wait or reject
- Allows bursts (if tokens available)

```go
type TokenBucket struct {
    capacity   int
    tokens     float64
    refillRate float64  // tokens per second
    lastRefill time.Time
    mu         sync.Mutex
}

func (tb *TokenBucket) Allow() bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()

    // Add tokens
    tb.tokens += elapsed * tb.refillRate
    if tb.tokens > float64(tb.capacity) {
        tb.tokens = float64(tb.capacity)
    }
    tb.lastRefill = now

    if tb.tokens >= 1 {
        tb.tokens -= 1
        return true
    }
    return false
}
```

**Good for:** User-facing APIs. Allows occasional bursts for better UX.

## Key Difference

- **Leaky Bucket**: Constant output, smooths bursts
- **Token Bucket**: Allows bursts, maintains average rate

## What I'm Using

For calling third-party APIs: **Leaky Bucket**. I want predictable, steady rate.

For my own API: **Token Bucket**. Users can burst occasionally, better experience.

Or just use `golang.org/x/time/rate` - it's basically a token bucket:

```go
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 100) // 10 req/s, burst of 100

if limiter.Allow() {
    // make request
}

// Or wait
limiter.Wait(ctx)
```

## Redis Implementation

For distributed systems, need shared state. Found this [Redis rate limiting article](https://zhuanlan.zhihu.com/p/43743489) (in Chinese) that shows Lua script approach.

Basic idea:
```lua
-- Token bucket in Redis (simplified)
local tokens = redis.call('get', key) or capacity
local now = redis.call('time')[1]
local last = redis.call('get', key .. ':last') or now

-- Refill tokens
local elapsed = now - last
tokens = math.min(capacity, tokens + elapsed * rate)

-- Consume token
if tokens >= 1 then
    redis.call('set', key, tokens - 1)
    redis.call('set', key .. ':last', now)
    return 1
else
    return 0
end
```

Atomic operations, works across servers.

That's my understanding so far. Both algorithms work, just pick based on whether you need smoothing (leaky) or bursts (token).
