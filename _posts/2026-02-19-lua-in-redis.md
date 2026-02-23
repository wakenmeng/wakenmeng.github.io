---
layout: post
title: "Lua in Redis: RateLimiter Example"
date: 2026-02-19
author: Waken
tags: [redis, lua, backend, ratelimit]
---

Use Lua in Redis, how and why, with an example of **token bucket rate limiter**.
<!-- more -->

raise NotImplmentedError

## How to run Lua script in Redis?

### Cache Script

We can use `SCRIPT LOAD "xxx"` to store the script in redis. Redis use SHA1 to generate the hash of the content. As long as the content doesn't change, the hash key of the script won't change. We can use this to check and load scripts without worrying that we may generate multiple script cache every time we load it for the hash key is the same.


## Why do we need Lua in Redis?

### Comparison

### Performance

## Practicle Example: Token Bucket Rate Limiter

### Benchmark
