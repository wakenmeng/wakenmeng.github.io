---
layout: post
title: "Go Performance: When to Use Slices vs Maps for Lookups"
date: 2022-10-10
author: Waken
tags: [golang, performance, data-structures]
comments: true
toc: true
---

When writing Go code, one of the most common decisions developers face is choosing between slices and maps for data storage and lookup. While the choice might seem obvious at first—"maps are for lookups, right?"—the reality is more nuanced. Understanding the performance characteristics of each can help you write more efficient Go code.

<!-- more -->

## The Question

Should you use a `map[K]V` or iterate through a `[]V` when searching for elements? The answer, as with many performance questions, is: **it depends on the size of your dataset**.

## Performance Characteristics

### Theoretical Complexity

Let's start with the theoretical time complexity:

- **Map lookup**: O(1) average case, O(n) worst case
- **Slice iteration**: O(n) always

From a Big-O perspective, maps should always win for larger datasets. However, Big-O notation doesn't tell the whole story for small collections.

### Real-World Benchmarks

Here are benchmark results comparing maps and slices for different data sizes:

#### String Key-Value Pairs

```go
// Map approach
m := map[string]string{
    "key1": "value1",
    "key2": "value2",
    // ...
}
val := m["key1"]

// Slice approach
type pair struct {
    key   string
    value string
}
s := []pair{
    {"key1", "value1"},
    {"key2", "value2"},
    // ...
}
// Linear search
for _, p := range s {
    if p.key == "key1" {
        val = p.value
        break
    }
}
```

**Benchmark Results:**

| Elements | Map Lookup | Slice Scan | Winner |
|----------|-----------|-----------|---------|
| 1        | 8ns       | 6ns       | Slice ✓ |
| 5        | 100ns     | 100ns     | Tie     |
| 10       | 100ns     | 230ns     | Map ✓   |
| 100      | 100ns     | 2200ns    | Map ✓   |

#### Integer Sets

```go
// Map approach (set)
m := map[int]struct{}{
    1: {},
    2: {},
    // ...
}
_, exists := m[1]

// Slice approach
s := []int{1, 2, 3, 4, 5}
exists := false
for _, v := range s {
    if v == 1 {
        exists = true
        break
    }
}
```

**Benchmark Results:**

| Elements | Map Lookup | Slice Scan | Winner |
|----------|-----------|-----------|---------|
| 1        | 22ns      | 6ns       | Slice ✓ |
| 5        | 35ns      | 30ns      | Slice ✓ |
| 10       | 55ns      | 45ns      | Slice ✓ |
| 100      | 75ns      | 500ns     | Map ✓   |

## The Break-Even Points

Based on these benchmarks, we can identify critical thresholds:

1. **String key-value pairs**: Maps become faster at **~5 elements**
2. **Integer sets**: Maps become faster at **~10 elements**

### Why the Difference?

The discrepancy between string and integer performance comes from several factors:

1. **Hashing overhead**: Computing hash values for strings is more expensive than for integers
2. **Memory access patterns**: Slices offer better cache locality for sequential access
3. **Comparison costs**: String comparison is slower than integer comparison

## Why Maps Can Be Faster

Despite the O(1) vs O(n) complexity difference, several factors make maps performant even for small datasets:

### 1. Optimized Hash Functions

Go's map implementation uses AES-based hashing, which is hardware-accelerated on modern CPUs:

```go
// Go's runtime uses CPU AES instructions
// for fast, secure hashing
hash := runtime.memhash(key, seed)
```

### 2. Efficient Bucket Design

Go maps use 8 items per bucket, reducing collision overhead:

```go
// Simplified Go map bucket structure
type bmap struct {
    tophash [8]uint8  // Hash prefixes for quick rejection
    keys    [8]keytype
    values  [8]valuetype
    overflow *bmap     // Overflow bucket if needed
}
```

### 3. Memory Pre-allocation

Maps pre-allocate buckets, avoiding repeated allocations during growth.

## Practical Recommendations

### Use Slices When:

```go
// 1. Very small, known-size collections (< 5-10 items)
weekdays := []string{"Mon", "Tue", "Wed", "Thu", "Fri"}

// 2. Ordered iteration is important
for _, day := range weekdays {
    fmt.Println(day)  // Guaranteed order
}

// 3. Memory is extremely constrained
// Slices have lower overhead than maps for tiny datasets
```

### Use Maps When:

```go
// 1. Frequent lookups with dynamic data
cache := make(map[string]*User)

// 2. Set operations (membership testing)
visited := make(map[int]struct{})
if _, ok := visited[nodeID]; !ok {
    visited[nodeID] = struct{}{}
}

// 3. Unknown or growing dataset size
config := make(map[string]string)

// 4. Key-value association is the primary use case
headers := map[string]string{
    "Content-Type": "application/json",
    "Accept":       "application/json",
}
```

## Optimization Tips

### Pre-sizing Maps

When you know the approximate size, pre-allocate:

```go
// Bad: causes multiple reallocations
m := make(map[string]int)
for i := 0; i < 1000; i++ {
    m[fmt.Sprintf("key%d", i)] = i
}

// Good: single allocation
m := make(map[string]int, 1000)
for i := 0; i < 1000; i++ {
    m[fmt.Sprintf("key%d", i)] = i
}
```

### Using sync.Map for Concurrent Access

For concurrent scenarios, consider `sync.Map`:

```go
var cache sync.Map

// Store
cache.Store("key", value)

// Load
if val, ok := cache.Load("key"); ok {
    // use val
}
```

## Benchmark Your Specific Use Case

While these guidelines are helpful, **always profile your actual workload**:

```go
package main

import "testing"

func BenchmarkMapLookup(b *testing.B) {
    m := make(map[int]struct{}, 100)
    for i := 0; i < 100; i++ {
        m[i] = struct{}{}
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = m[50]
    }
}

func BenchmarkSliceLookup(b *testing.B) {
    s := make([]int, 100)
    for i := 0; i < 100; i++ {
        s[i] = i
    }

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        for _, v := range s {
            if v == 50 {
                break
            }
        }
    }
}
```

Run with:
```bash
go test -bench=. -benchmem
```

## Conclusion

The old adage **"use the obvious data structure"** holds true in Go. For most real-world scenarios:

- **Maps** are the natural choice for key-value lookups and sets
- **Slices** are better for small, ordered collections and sequential processing

The performance benefits of choosing the "correct" structure are usually marginal for small datasets, but the code clarity and maintainability benefits are significant. When in doubt, choose the data structure that best represents your domain—chances are it will also be the most performant option.

## References

- [Go Maps in Action](https://go.dev/blog/maps)
- [Go Slice Internals](https://go.dev/blog/slices-intro)
- [Original Benchmark Article](https://darkcoding.net/software/go-slice-search-vs-map-lookup/)
