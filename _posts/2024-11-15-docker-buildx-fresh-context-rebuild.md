---
layout: post
title: "Docker BuildX: When Cache Goes Wrong"
date: 2024-11-15
author: Waken
tags: [docker, buildx, devops]
comments: true
---

Had a weird Docker issue today. Image built successfully but ran old code. Turned out to be stale cache layers.

<!-- more -->

## The Problem

```bash
# Build succeeds
docker buildx build -t myapp:latest .
# [+] Building 2.3s FINISHED

# But running it shows old behavior!
docker run myapp:latest
# Old code is running??
```

Changed code, rebuilt image, but the old version still runs.

## The Fix

Nuclear option - clear ALL build caches:

```bash
# Clear buildx cache
docker buildx prune --all --force

# Clear builder cache
docker builder prune --all --force

# Then rebuild fresh
docker buildx build \
  --platform linux/amd64 \
  -t myapp:latest \
  --load \
  .
```

This forces a complete rebuild without any cached layers.

## Why This Happens

Docker BuildX has multiple cache layers:
1. Build cache (buildx-specific)
2. Builder instance cache
3. Image layer cache

`--no-cache` only clears #1. BuildX can still reuse #2 and #3, causing stale layers.

## My Rebuild Script

```bash
#!/bin/bash
# rebuild-fresh.sh

echo "Cleaning caches..."
docker buildx prune --all --force
docker builder prune --all --force

echo "Building..."
docker buildx build \
  --platform linux/amd64 \
  --file Dockerfile \
  --tag myapp:latest \
  --load \
  .

echo "Done!"
docker images myapp:latest
```

## Multi-Platform Builds

For multiple architectures, use `--push` instead of `--load`:

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --tag registry.com/myapp:latest \
  --push \
  .
```

Can't use `--load` with multi-platform (limitation of BuildX).

## When to Use This

- Mysterious "old code" in new images
- Build succeeds but behavior is wrong
- After interrupted builds
- When external dependencies update but Dockerfile doesn't change

## Less Nuclear Options

If you don't want to nuke everything:

```bash
# Remove specific builder
docker buildx rm mybuilder
docker buildx create --name mybuilder --use

# Clear old cache only
docker buildx prune --filter "until=168h"  # older than 7 days

# Check what will be deleted
docker buildx du
```

## CI/CD

For GitHub Actions:

```yaml
- name: Clean build cache
  run: |
    docker buildx prune --all --force

- name: Build
  run: |
    docker buildx build \
      --platform linux/amd64,linux/arm64 \
      --tag ${{ env.IMAGE }} \
      --push \
      .
```

## Preventing Cache Issues

**1. Use .dockerignore**

```
node_modules
.git
*.log
dist
```

**2. Order Dockerfile properly**

```dockerfile
# Dependencies first (changes less)
COPY package*.json ./
RUN npm ci

# Code last (changes more)
COPY . .
RUN npm run build
```

**3. Explicit cache busting when needed**

```dockerfile
ARG CACHE_BUST=1
RUN echo "Bust: $CACHE_BUST" && apt-get update
```

Build with: `--build-arg CACHE_BUST=$(date +%s)`

## Takeaway

When Docker builds behave mysteriously:
1. Prune all caches
2. Rebuild fresh
3. Check .dockerignore

Most "weird Docker issues" are cache-related.
