---
layout: post
title: "Rate Limiting Algorithms: Leaky Bucket vs Token Bucket"
date: 2022-07-27
author: Waken
tags: [algorithms, system-design, rate-limiting, golang]
comments: true
toc: true
---

Rate limiting is a critical technique for protecting services from being overwhelmed by excessive requests. Whether you're building an API, integrating with third-party services, or managing system resources, understanding rate limiting algorithms is essential. In this post, we'll explore two fundamental approaches: the Leaky Bucket and Token Bucket algorithms.

<!-- more -->

## Why Rate Limiting Matters

Before diving into algorithms, let's understand why rate limiting is crucial:

1. **API Protection**: Prevent abuse and ensure fair usage among clients
2. **Third-party Integration**: Respect rate limits imposed by external services
3. **Resource Management**: Protect backend services from overload
4. **Cost Control**: Manage costs for metered services (e.g., cloud APIs)
5. **Quality of Service**: Ensure consistent performance for all users

## The Two Main Algorithms

### 1. Leaky Bucket Algorithm

The Leaky Bucket algorithm enforces a constant output rate, regardless of input bursts. Think of it as a bucket with a small hole at the bottom—water (requests) flows in at varying rates, but leaks out at a constant rate.

#### How It Works

```
┌─────────────────┐
│   Requests In   │ (variable rate)
└────────┬────────┘
         │
    ┌────▼────┐
    │         │
    │  Bucket │ ← Fixed capacity
    │         │
    └────┬────┘
         │ (constant leak rate)
    ┌────▼────┐
    │ Process │
    └─────────┘
```

**Key Characteristics:**
- Requests arrive at variable rates
- Bucket has a fixed capacity
- Requests leak out (process) at a constant rate
- If bucket is full, new requests are discarded or queued

#### Go Implementation

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
)

// LeakyBucket implements the leaky bucket algorithm
type LeakyBucket struct {
    capacity     int           // Maximum bucket size
    remaining    int           // Current water level
    leakRate     time.Duration // Rate at which requests leak out
    mu           sync.Mutex
    lastLeakTime time.Time
}

// NewLeakyBucket creates a new leaky bucket rate limiter
func NewLeakyBucket(capacity int, leakRate time.Duration) *LeakyBucket {
    return &LeakyBucket{
        capacity:     capacity,
        remaining:    capacity,
        leakRate:     leakRate,
        lastLeakTime: time.Now(),
    }
}

// leak processes the leaked requests based on time elapsed
func (lb *LeakyBucket) leak() {
    now := time.Now()
    elapsed := now.Sub(lb.lastLeakTime)

    // Calculate how many requests should have leaked
    leaked := int(elapsed / lb.leakRate)

    if leaked > 0 {
        lb.remaining += leaked
        if lb.remaining > lb.capacity {
            lb.remaining = lb.capacity
        }
        lb.lastLeakTime = now
    }
}

// Allow checks if a request can be processed
func (lb *LeakyBucket) Allow() bool {
    lb.mu.Lock()
    defer lb.mu.Unlock()

    lb.leak()

    if lb.remaining > 0 {
        lb.remaining--
        return true
    }

    return false
}

// Wait blocks until the request can be processed
func (lb *LeakyBucket) Wait(ctx context.Context) error {
    for {
        if lb.Allow() {
            return nil
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(lb.leakRate):
            // Try again after leak interval
        }
    }
}
```

**Usage Example:**

```go
func main() {
    // Create a bucket that processes 10 requests per second
    bucket := NewLeakyBucket(10, 100*time.Millisecond)

    // Simulate API requests
    for i := 0; i < 20; i++ {
        if bucket.Allow() {
            fmt.Printf("Request %d: Allowed\n", i)
        } else {
            fmt.Printf("Request %d: Rate limited\n", i)
        }
    }

    // Or use Wait for blocking behavior
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := bucket.Wait(ctx); err == nil {
        fmt.Println("Request processed")
    }
}
```

### 2. Token Bucket Algorithm

The Token Bucket algorithm allows bursts while maintaining an average rate. Tokens are added to a bucket at a fixed rate, and each request consumes a token. If tokens are available, the request is processed immediately.

#### How It Works

```
         Add tokens at fixed rate
                  │
             ┌────▼────┐
Request ───► │  Bucket │ (has N tokens)
             └────┬────┘
                  │
                  ▼
    Token available? ──Yes──► Process Request
         │
         No
         │
         ▼
    Reject/Wait
```

**Key Characteristics:**
- Tokens are added at a constant rate
- Bucket has a maximum capacity
- Requests consume tokens instantly
- Allows bursts up to bucket capacity
- More flexible than leaky bucket

#### Go Implementation

```go
package main

import (
    "context"
    "sync"
    "time"
)

// TokenBucket implements the token bucket algorithm
type TokenBucket struct {
    capacity      int           // Maximum tokens
    tokens        float64       // Current tokens
    refillRate    float64       // Tokens per second
    lastRefill    time.Time
    mu            sync.Mutex
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
    return &TokenBucket{
        capacity:   capacity,
        tokens:     float64(capacity), // Start full
        refillRate: refillRate,
        lastRefill: time.Now(),
    }
}

// refill adds tokens based on time elapsed
func (tb *TokenBucket) refill() {
    now := time.Now()
    elapsed := now.Sub(tb.lastRefill).Seconds()

    // Add tokens based on elapsed time
    tb.tokens += elapsed * tb.refillRate

    // Cap at maximum capacity
    if tb.tokens > float64(tb.capacity) {
        tb.tokens = float64(tb.capacity)
    }

    tb.lastRefill = now
}

// Allow checks if a request can be processed
func (tb *TokenBucket) Allow() bool {
    return tb.AllowN(1)
}

// AllowN checks if N requests can be processed
func (tb *TokenBucket) AllowN(n int) bool {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    tb.refill()

    if tb.tokens >= float64(n) {
        tb.tokens -= float64(n)
        return true
    }

    return false
}

// Wait blocks until a token is available
func (tb *TokenBucket) Wait(ctx context.Context) error {
    return tb.WaitN(ctx, 1)
}

// WaitN blocks until N tokens are available
func (tb *TokenBucket) WaitN(ctx context.Context, n int) error {
    for {
        if tb.AllowN(n) {
            return nil
        }

        // Calculate wait time for next token
        tb.mu.Lock()
        needed := float64(n) - tb.tokens
        waitTime := time.Duration(needed/tb.refillRate*1000) * time.Millisecond
        tb.mu.Unlock()

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(waitTime):
            // Try again
        }
    }
}

// Tokens returns the current number of tokens
func (tb *TokenBucket) Tokens() float64 {
    tb.mu.Lock()
    defer tb.mu.Unlock()

    tb.refill()
    return tb.tokens
}
```

**Usage Example:**

```go
func main() {
    // Create a bucket with capacity of 100, refilling at 10 tokens/second
    bucket := NewTokenBucket(100, 10)

    // Allow bursts
    if bucket.AllowN(50) {
        fmt.Println("Burst of 50 requests: Allowed")
    }

    // Gradual consumption
    for i := 0; i < 15; i++ {
        if bucket.Allow() {
            fmt.Printf("Request %d: Allowed (%.2f tokens remaining)\n",
                i, bucket.Tokens())
        } else {
            fmt.Printf("Request %d: Rate limited\n", i)
        }
        time.Sleep(50 * time.Millisecond)
    }
}
```

## Comparison: Leaky Bucket vs Token Bucket

| Feature | Leaky Bucket | Token Bucket |
|---------|--------------|--------------|
| **Output Rate** | Constant | Variable (up to burst limit) |
| **Bursts** | Smoothed out | Allowed (up to capacity) |
| **Implementation** | Simpler | Slightly more complex |
| **Use Case** | Strict rate enforcement | Flexible with burst tolerance |
| **Memory** | Tracks queue size | Tracks token count |
| **Fairness** | Very fair | Allows occasional bursts |

### When to Use Leaky Bucket

```go
// Good for: Strict rate limiting to external APIs
type APIClient struct {
    limiter *LeakyBucket
}

func (c *APIClient) Call(endpoint string) error {
    // Wait for rate limit
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := c.limiter.Wait(ctx); err != nil {
        return fmt.Errorf("rate limit timeout: %w", err)
    }

    // Make API call
    return c.makeRequest(endpoint)
}
```

**Best for:**
- Protecting downstream services from bursts
- Ensuring smooth, predictable traffic
- Network traffic shaping
- Streaming data at constant rates

### When to Use Token Bucket

```go
// Good for: User-facing APIs with burst allowance
type Server struct {
    limiter *TokenBucket
}

func (s *Server) HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Allow occasional bursts for better UX
    if !s.limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    // Process request
    s.processRequest(w, r)
}
```

**Best for:**
- User-facing APIs (better UX with bursts)
- Variable workload patterns
- Systems that can handle temporary bursts
- Flexible rate limiting

## Real-World Implementation with Redis

For distributed systems, implement rate limiting with Redis:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
)

// RedisTokenBucket implements distributed token bucket using Redis
type RedisTokenBucket struct {
    client     *redis.Client
    key        string
    capacity   int
    refillRate float64
}

func NewRedisTokenBucket(client *redis.Client, key string, capacity int, refillRate float64) *RedisTokenBucket {
    return &RedisTokenBucket{
        client:     client,
        key:        key,
        capacity:   capacity,
        refillRate: refillRate,
    }
}

// Allow checks and consumes a token using Lua script for atomicity
func (rtb *RedisTokenBucket) Allow(ctx context.Context) (bool, error) {
    script := `
local key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1]) or capacity
local last_refill = tonumber(bucket[2]) or now

-- Refill tokens
local elapsed = now - last_refill
local new_tokens = tokens + (elapsed * refill_rate)
if new_tokens > capacity then
    new_tokens = capacity
end

-- Try to consume a token
if new_tokens >= 1 then
    new_tokens = new_tokens - 1
    redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
    redis.call('EXPIRE', key, 3600)
    return 1
else
    redis.call('HMSET', key, 'tokens', new_tokens, 'last_refill', now)
    redis.call('EXPIRE', key, 3600)
    return 0
end
`

    now := time.Now().Unix()
    result, err := rtb.client.Eval(ctx, script, []string{rtb.key},
        rtb.capacity, rtb.refillRate, now).Int()

    if err != nil {
        return false, err
    }

    return result == 1, nil
}
```

**Usage:**

```go
func main() {
    client := redis.NewClient(&redis.Options{
        Addr: "localhost:6379",
    })

    limiter := NewRedisTokenBucket(
        client,
        "api:ratelimit:user:123",
        100,  // capacity
        10.0, // 10 tokens per second
    )

    ctx := context.Background()
    allowed, err := limiter.Allow(ctx)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    if allowed {
        fmt.Println("Request allowed")
    } else {
        fmt.Println("Rate limited")
    }
}
```

## Popular Libraries

Instead of rolling your own, consider these battle-tested libraries:

### Go Libraries

```go
// 1. golang.org/x/time/rate - Official Go rate limiter
import "golang.org/x/time/rate"

limiter := rate.NewLimiter(10, 100) // 10 req/s, burst of 100

if limiter.Allow() {
    // Process request
}

// Or wait
ctx := context.Background()
limiter.Wait(ctx)

// 2. github.com/uber-go/ratelimit - Uber's rate limiter
import "go.uber.org/ratelimit"

rl := ratelimit.New(100) // 100 requests per second

for i := 0; i < 1000; i++ {
    rl.Take()
    // Process request
}
```

## Best Practices

1. **Choose the Right Algorithm**
   - Use leaky bucket for strict, constant rate enforcement
   - Use token bucket for user-facing APIs that benefit from bursts

2. **Set Appropriate Limits**
   ```go
   // Consider: burst size, refill rate, and timeout
   limiter := NewTokenBucket(
       100,              // Allow bursts up to 100 requests
       10,               // Refill at 10 req/second
   )
   ```

3. **Handle Rate Limit Errors Gracefully**
   ```go
   if !limiter.Allow() {
       // Return proper HTTP status code
       http.Error(w, "Rate limit exceeded. Try again in 10s",
           http.StatusTooManyRequests)
       w.Header().Set("Retry-After", "10")
       return
   }
   ```

4. **Monitor and Alert**
   ```go
   // Track rate limit hits
   metrics.Increment("ratelimit.exceeded")

   // Log for analysis
   log.Warn("Rate limit exceeded",
       "user_id", userID,
       "endpoint", endpoint)
   ```

5. **Use Distributed Rate Limiting**
   - For multi-server deployments, use Redis or similar
   - Consider eventual consistency trade-offs

## Conclusion

Both Leaky Bucket and Token Bucket algorithms serve important roles in rate limiting:

- **Leaky Bucket**: Best for scenarios requiring strict, constant rate enforcement
- **Token Bucket**: Better for user-facing systems that benefit from burst allowance

Choose based on your specific requirements, and consider using established libraries like `golang.org/x/time/rate` for production systems. Always monitor your rate limits and adjust them based on actual usage patterns and system capacity.

## References

- [Token Bucket Algorithm - Wikipedia](https://en.wikipedia.org/wiki/Token_bucket)
- [Leaky Bucket Algorithm - Wikipedia](https://en.wikipedia.org/wiki/Leaky_bucket)
- [Go time/rate package](https://pkg.go.dev/golang.org/x/time/rate)
- [Stripe API Rate Limits](https://stripe.com/docs/rate-limits)
