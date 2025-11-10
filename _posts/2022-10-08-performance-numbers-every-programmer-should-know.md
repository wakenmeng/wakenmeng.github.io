---
layout: post
title: "Latency Numbers Every Programmer Should Know (2024 Update)"
date: 2022-10-08
author: Waken
tags: [performance, computer-science, optimization, systems]
comments: true
toc: true
---

Understanding computer performance at a fundamental level is crucial for writing efficient code. These latency numbers, originally compiled by Jeff Dean and Peter Norvig, reveal the relative costs of different operations in modern computers. Let's explore what these numbers mean and how they impact your daily programming decisions.

<!-- more -->

## The Classic Numbers (Updated for 2024)

Here are the fundamental latency numbers every programmer should internalize:

| Operation | Latency | Scaled Time* |
|-----------|---------|--------------|
| L1 cache reference | 0.5 ns | 0.5 sec |
| Branch mispredict | 5 ns | 5 sec |
| L2 cache reference | 7 ns | 7 sec |
| Mutex lock/unlock | 100 ns | 100 sec (1.7 min) |
| Main memory reference | 100 ns | 100 sec (1.7 min) |
| Compress 1KB with Snappy | 10 µs | 2.8 hours |
| Send 2KB over 1 Gbps network | 20 µs | 5.6 hours |
| Read 1 MB sequentially from memory | 250 µs | 2.9 days |
| Round trip in same datacenter | 500 µs | 5.8 days |
| SSD random read | 150 µs | 1.7 days |
| Read 1 MB sequentially from SSD | 1 ms | 11.6 days |
| Disk seek (HDD) | 10 ms | 116 days |
| Read 1 MB sequentially from network | 10 ms | 116 days |
| Read 1 MB sequentially from disk (HDD) | 30 ms | 11.6 months |
| Send packet CA → Netherlands → CA | 150 ms | 4.8 years |

\* If L1 cache reference = 1 second, this is how long other operations would take

## Visual Comparison

Let me put this in perspective using time scales we can understand:

```
L1 cache reference:          |
Branch mispredict:           |||||||||
L2 cache reference:          ||||||||||||||
Mutex lock/unlock:           ||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
Main memory:                 ||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||
SSD random read:             [300x longer than memory]
Sequential SSD 1MB:          [2,000x longer than memory]
HDD seek:                    [100,000x longer than memory]
Network round trip (EU-US):  [1,500,000x longer than memory]
```

## Key Insights for Programmers

### 1. Memory Hierarchy Matters

The performance cliff between different levels of the memory hierarchy is dramatic:

```c
// Fast: Data fits in L1 cache (0.5 ns per access)
int sum = 0;
for (int i = 0; i < 1000; i++) {
    sum += arr[i]; // Sequential access, stays in cache
}

// Slower: Data scattered across memory (100 ns per access)
for (int i = 0; i < 1000; i++) {
    sum += arr[random_indices[i]]; // Random access, cache misses
}
```

**Lesson**: Design data structures for cache locality. Sequential access is 200x faster than random access!

### 2. Locks Are Expensive

At 100ns, mutex operations are as slow as main memory access:

```go
// Expensive: Lock on every operation
type Counter struct {
    mu    sync.Mutex
    count int
}

func (c *Counter) Increment() {
    c.mu.Lock()         // 100ns
    c.count++
    c.mu.Unlock()       // 100ns
}

// Better: Batch operations to amortize lock cost
func (c *Counter) IncrementBy(n int) {
    c.mu.Lock()
    c.count += n        // One lock for N increments
    c.mu.Unlock()
}

// Best: Use atomic operations (faster than mutex)
type AtomicCounter struct {
    count atomic.Int64
}

func (c *AtomicCounter) Increment() {
    c.count.Add(1)      // ~5-10ns, no syscall
}
```

**Lesson**: Minimize lock contention, batch operations, use lock-free structures when possible.

### 3. Network vs Disk vs Memory

The hierarchy of data access:

```python
# Fastest: In-memory cache (100 ns)
value = cache.get(key)

# Fast: SSD read (150 µs) - 1,500x slower than memory
value = ssd.read(key)

# Slow: HDD read (10 ms) - 100,000x slower than memory
value = hdd.read(key)

# Very Slow: Network call (0.5-150 ms) - depends on distance
value = api.get(url)
```

**Practical implications**:

- **Cache aggressively**: Even a 50% cache hit rate saves enormous time
- **Use SSDs**: 67x faster than HDDs for random reads
- **Minimize network calls**: One network round trip = 50,000 L1 cache hits

### 4. Compression Can Be Worth It

Compressing 1KB takes 10µs (10,000 ns). Is it worth it?

```python
# Network transfer: 2KB over 1 Gbps = 20 µs
uncompressed_time = 20_000  # ns

# With compression (assuming 50% compression ratio):
compression_time = 10_000    # compress 1KB
transfer_time = 10_000       # transfer 1KB (half size)
decompression_time = 10_000  # decompress
compressed_total = 30_000    # ns

# Without compression is faster for local network!
# But for slow networks (100 Mbps):
slow_transfer = 200_000      # ns for 2KB
compressed_slow = 30_000 + 100_000  # still faster!
```

**Lesson**: Compression makes sense for:
- Slow networks
- Large payloads
- Storage (SSDs/HDDs are much slower than CPU)

### 5. Sequential Access Is King

Reading 1MB sequentially from different sources:

```
Memory:   250 µs    (4 GB/sec)
SSD:      1 ms      (1 GB/sec)
HDD:      30 ms     (33 MB/sec)
Network:  10 ms     (100 MB/sec for 1 Gbps)
```

**Code implications**:

```rust
// Bad: Random access pattern
for i in 0..1_000_000 {
    let index = random_index();
    process(data[index]);  // Cache misses!
}

// Good: Sequential access pattern
for item in data.iter() {
    process(item);         // Cache friendly!
}

// Better: Process in chunks that fit in cache
const CHUNK_SIZE: usize = 64 * 1024; // 64KB chunks
for chunk in data.chunks(CHUNK_SIZE) {
    for item in chunk {
        process(item);
    }
}
```

## Real-World Applications

### Database Query Optimization

```sql
-- Slow: Each row requires disk seek (10 ms each)
-- 1000 rows = 10 seconds!
SELECT * FROM users WHERE email = 'user@example.com';

-- Fast: Index allows sequential read
-- With B-tree index, find in ~3-4 seeks = 30-40 ms
CREATE INDEX idx_email ON users(email);

-- Fastest: Covered index (all data in index)
-- Pure index scan, mostly in memory
CREATE INDEX idx_email_data ON users(email, name, created_at);
```

### API Design

```javascript
// Bad: N+1 query problem
// 100 users = 1 + 100 queries = 50+ seconds
for (const user of users) {
    const posts = await db.query('SELECT * FROM posts WHERE user_id = ?', user.id);
}

// Good: Single query with join
// 1 query = 500 ms
const usersWithPosts = await db.query(`
    SELECT users.*, posts.*
    FROM users
    LEFT JOIN posts ON posts.user_id = users.id
`);

// Best: Add caching layer
// First request: 500 ms
// Subsequent requests: 100 ns (memory)
const cached = await cache.get('users_with_posts');
if (!cached) {
    const data = await db.query(...);
    await cache.set('users_with_posts', data, ttl);
}
```

### Data Structure Choice

```go
// When to use what:
// Array/Slice - Sequential access, cache friendly
data := []int{1, 2, 3, 4, 5}
for _, v := range data {  // ~0.5 ns per element (L1 cache)
    process(v)
}

// Map - Random access, hash lookup
cache := make(map[string]string)
val := cache["key"]  // ~100 ns (main memory + hashing)

// Benchmark: 1 million sequential reads vs 1 million map lookups
// Sequential: 500,000 ns (0.5 ms)
// Map lookup: 100,000,000 ns (100 ms)
// 200x difference!
```

## Modern Hardware Considerations (2024)

### SSD vs HDD

Modern SSDs have dramatically changed the landscape:

```
Operation          HDD         SATA SSD    NVMe SSD
Random Read        10 ms       150 µs      10 µs
Sequential 1MB     30 ms       1 ms        200 µs
IOPS (4K)          100         10,000      500,000
```

**Implications**:
- Random reads are 100-1000x faster on SSDs
- Sequential performance is 30-150x better
- But still 1000x slower than RAM!

### Network Speeds

```
Connection Type    Latency     Bandwidth
Same machine       50 ns       N/A
Same rack          500 µs      10-100 Gbps
Same datacenter    1-2 ms      1-10 Gbps
Cross-country      30-50 ms    100-1000 Mbps
Intercontinental   100-200 ms  10-100 Mbps
```

### CPU Caches (Typical 2024 CPU)

```
Cache    Size       Latency    Bandwidth
L1       32-64 KB   0.5 ns     ~1000 GB/s
L2       256-512 KB 7 ns       ~400 GB/s
L3       8-32 MB    20-40 ns   ~200 GB/s
RAM      16-128 GB  100 ns     ~50 GB/s
```

## Practical Optimization Checklist

Based on these numbers, here's what to optimize first:

1. **Eliminate unnecessary work** (cheapest optimization)
   - Remove redundant computations
   - Cache expensive results

2. **Minimize network calls**
   - Batch requests
   - Use connection pooling
   - Implement client-side caching

3. **Optimize memory access patterns**
   - Use sequential access where possible
   - Design cache-friendly data structures
   - Keep hot data compact

4. **Reduce lock contention**
   - Use fine-grained locking
   - Employ lock-free algorithms
   - Batch updates

5. **Use appropriate data structures**
   - Arrays for sequential access
   - Hash tables for lookups
   - B-trees for range queries

6. **Leverage compression intelligently**
   - Compress data at rest (disk/SSD)
   - Compress over slow networks
   - Skip compression for fast local networks

## Benchmarking Your Own System

Don't trust these numbers blindly—measure your own system:

```go
package main

import (
    "fmt"
    "runtime"
    "sync"
    "testing"
    "time"
)

func BenchmarkMemoryAccess(b *testing.B) {
    data := make([]int, 1000000)
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        _ = data[i%len(data)]
    }
}

func BenchmarkMutex(b *testing.B) {
    var mu sync.Mutex
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        mu.Lock()
        mu.Unlock()
    }
}

func BenchmarkCacheLineFalseSharing(b *testing.B) {
    type PaddedInt struct {
        value int
        _     [7]int64 // Padding to avoid false sharing
    }

    counters := make([]PaddedInt, runtime.NumCPU())
    b.ResetTimer()

    b.RunParallel(func(pb *testing.PB) {
        id := runtime.GoID() % runtime.NumCPU()
        for pb.Next() {
            counters[id].value++
        }
    })
}
```

Run with:
```bash
go test -bench=. -benchmem -count=5
```

## Conclusion

These latency numbers reveal fundamental truths about computer systems:

1. **Memory is 200x slower than cache**—optimize for cache locality
2. **Disk is 100,000x slower than memory**—cache aggressively
3. **Network is 500,000x slower than memory** (for distant calls)—batch and cache
4. **Locks cost as much as memory access**—minimize contention

The next time you write code, ask yourself:
- How many cache misses will this cause?
- Can I batch these operations?
- Am I making unnecessary network calls?
- Are my locks held longer than necessary?

Understanding these numbers turns you from a programmer who writes code that works into one who writes code that **works efficiently**.

## References

- [Original Post by Peter Norvig](https://norvig.com/21-days.html#answers)
- [Latency Numbers Interactive](https://colin-scott.github.io/personal_website/research/interactive_latency.html)
- [Google's Jeff Dean Numbers](https://static.googleusercontent.com/media/research.google.com/en//people/jeff/Stanford-DL-Nov-2010.pdf)
- [Systems Performance by Brendan Gregg](https://www.brendangregg.com/systems-performance-2nd-edition-book.html)
