---
layout: post
title: "Go: Slices Can Be Faster Than Maps for Small Collections"
date: 2022-10-10
author: Waken
tags: [golang, performance]
comments: true
---

Found an interesting article about [Go slice search vs map lookup performance](https://darkcoding.net/software/go-slice-search-vs-map-lookup/). The TLDR surprised me.

<!-- more -->

## The Benchmarks

For small collections, linear search through slices is actually faster than map lookups:

**String key-value pairs:**
- **< 5 items**: Slice wins
- **≥ 5 items**: Map wins

**Integer sets:**
- **< 10 items**: Slice wins
- **≥ 10 items**: Map wins

## Why?

Maps have overhead:
- Hash computation (even with hardware-accelerated AES)
- Bucket lookups
- Pointer chasing

For tiny collections, this overhead costs more than just iterating through a few items in cache-friendly memory.

## What I'm Doing Differently

Before, I'd automatically reach for maps. Now:

```go
// Old habit: always use maps
cache := make(map[string]int)

// New approach: for small known sets
weekdays := []string{"Mon", "Tue", "Wed", "Thu", "Fri"}
for _, day := range weekdays {
    if day == target { // Fast enough for 5 items
        // ...
    }
}
```

The performance difference isn't huge, but it's a good reminder: **use the obvious data structure**. For small, known-size collections, slices are simpler and often faster.

## Pre-sizing Maps

One thing I'm now doing more:

```go
// Instead of
m := make(map[string]int)

// Do this when you know the size
m := make(map[string]int, 1000)
```

Avoids multiple reallocations.

That's it. Not going to micro-optimize everything, but nice to know the break-even points.
