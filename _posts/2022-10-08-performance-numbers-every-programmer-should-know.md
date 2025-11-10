---
layout: post
title: "Latency Numbers Worth Knowing"
date: 2022-10-08
author: Waken
tags: [performance, systems]
comments: true
---

Came across [this classic post](https://surana.wordpress.com/2009/01/01/numbers-everyone-should-know/) about computer performance numbers. Really puts things in perspective.

<!-- more -->

## The Numbers

Here are the key ones (approximate, 2024 hardware):

```
L1 cache reference            0.5 ns
Branch mispredict             5 ns
L2 cache reference            7 ns
Mutex lock/unlock             100 ns
Main memory reference         100 ns
Compress 1KB with Snappy      10 µs (10,000 ns)
Send 2KB over 1 Gbps network  20 µs
Read 1MB from memory          250 µs
SSD random read               150 µs
Read 1MB from SSD             1 ms
Disk seek (HDD)               10 ms
Read 1MB from network         10 ms
Send packet CA→EU→CA          150 ms
```

## What This Means

**Cache is Fast**
- L1 cache: 0.5ns
- Memory: 100ns (200x slower)
- SSD: 150µs (300,000x slower)

Moral: Keep hot data in cache. Sequential access > random access.

**Locks Are Expensive**
Mutex lock/unlock costs as much as a memory read (100ns). For high-frequency operations, that adds up.

```go
// Bad: lock on every increment
func (c *Counter) Inc() {
    c.mu.Lock()
    c.count++  // 100ns+ for this lock
    c.mu.Unlock()
}

// Better: use atomic
atomic.AddInt64(&c.count, 1)  // ~5-10ns
```

**Network vs Disk vs Memory**

```
Memory:   100 ns
SSD:      150 µs  (1,500x slower)
Network:  10 ms   (100,000x slower)
```

Even a "slow" memory access is orders of magnitude faster than disk or network.

**Compression Can Help**

Compressing 1KB takes 10µs. Sending 2KB over network takes 20µs. If you can compress 2KB→1KB, total time is 10µs (compress) + 10µs (send) = 20µs. Same as sending uncompressed!

For slow networks or large data, compression is worth it.

## How I Use This

**1. Cache Aggressively**

Even a 50% cache hit rate saves huge amounts of time:

```
100 requests:
- No cache: 100 × 10ms = 1000ms
- 50% cache: 50 × 100ns + 50 × 10ms = 500ms
```

**2. Batch Operations**

```go
// Instead of N network calls
for id := range ids {
    user := api.GetUser(id)  // 150ms each
}

// One batch call
users := api.GetUsers(ids)  // 150ms total
```

**3. Use SSDs**

Random reads: HDD (10ms) vs SSD (150µs) = 67x faster

**4. Sequential > Random**

```
Sequential 1MB from memory: 250µs
Random memory accesses: ~100ns each × 1,000,000 = 100ms
```

400x difference! Structure data for sequential access.

## The 1-Second Scale

If L1 cache access = 1 second, then:
- Memory access = 3 minutes
- SSD read = 3.5 days
- HDD seek = 4 months
- Network round trip = 15 years

Puts it in perspective.

## Takeaway

Don't need to memorize exact numbers, but knowing the rough orders of magnitude helps:
- Cache: nanoseconds
- Memory: ~100 nanoseconds
- SSD: microseconds
- Network/Disk: milliseconds

Each jump is 1000x. Design accordingly.
