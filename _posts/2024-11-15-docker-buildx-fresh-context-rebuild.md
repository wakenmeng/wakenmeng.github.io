---
layout: post
title: "Docker BuildX: Solving Layer Caching Issues with Fresh Context Rebuilds"
date: 2024-11-15
author: Waken
tags: [docker, buildx, devops, troubleshooting, containers]
comments: true
toc: true
---

Have you ever encountered a situation where your Docker image builds successfully but behaves incorrectly because it's reusing stale cached layers? This is a common pitfall with Docker's layer caching mechanism. Here's how to properly rebuild with a fresh context using Docker BuildX.

<!-- more -->

## The Problem: Incorrect Layer Reuse

Docker's layer caching is a powerful feature that speeds up builds by reusing unchanged layers. However, this can backfire when:

1. **Build context changes aren't detected** - Files change outside Docker's awareness
2. **External dependencies update** - Package versions change but Dockerfile remains the same
3. **Cached layers become corrupted** - Rare but happens with interrupted builds
4. **Multi-stage builds** - Intermediate stages cache incorrectly

### Common Symptoms

```bash
# Build succeeds but...
docker build -t myapp:latest .
# [+] Building 2.3s (15/15) FINISHED

# Application doesn't work as expected
docker run myapp:latest
# Old behavior persists despite code changes!
```

You've updated your code, rebuilt the image, but the old version still runs. Sound familiar?

## The Root Cause

Docker BuildX uses multiple caching mechanisms:

```
Build Cache Layers:
├── Build cache (buildx cache)
├── Builder cache (builder instances)
└── Image layers (local registry)
```

A standard `docker build --no-cache` only clears one layer of caching. BuildX maintains additional caches that persist.

## The Solution: Complete Cache Clearing

### Step 1: Prune All BuildX Caches

```bash
# Clear buildx-specific cache
docker buildx prune --all

# Clear general builder cache
docker builder prune --all
```

**What this does:**
- Removes all build cache from all builders
- Clears intermediate layers
- Forces complete rebuild on next build

**⚠️ Warning:** This affects ALL build caches, not just your current project.

### Step 2: Rebuild with Fresh Context

```bash
docker buildx build \
  --platform linux/amd64 \
  -f build/assistant/Dockerfile \
  -t myregistry.com/myapp:latest \
  --load \
  .
```

**Flag breakdown:**
- `--platform linux/amd64` - Target platform (important for M1/M2 Macs)
- `-f build/assistant/Dockerfile` - Specify Dockerfile path
- `-t myregistry.com/myapp:latest` - Tag the image
- `--load` - Load the built image into local Docker (for single platform)
- `.` - Build context (current directory)

## Complete Workflow

### For Single-Platform Builds

```bash
#!/bin/bash
# rebuild-fresh.sh

set -e  # Exit on error

echo "🧹 Cleaning all build caches..."
docker buildx prune --all --force
docker builder prune --all --force

echo "🏗️  Building fresh image..."
docker buildx build \
  --platform linux/amd64 \
  --file Dockerfile \
  --tag myapp:latest \
  --load \
  .

echo "✅ Build complete! Image: myapp:latest"
docker images myapp:latest
```

### For Multi-Platform Builds

```bash
#!/bin/bash
# rebuild-multiplatform.sh

set -e

echo "🧹 Cleaning all build caches..."
docker buildx prune --all --force
docker builder prune --all --force

echo "🏗️  Building multi-platform image..."
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --file Dockerfile \
  --tag myregistry.com/myapp:latest \
  --push \
  .

echo "✅ Multi-platform build complete and pushed!"
```

**Note:** Use `--push` instead of `--load` for multi-platform builds, as `--load` only supports single platforms.

## Understanding BuildX Prune Options

### Prune Levels

```bash
# 1. Light clean (dangling cache only)
docker buildx prune

# 2. Aggressive clean (all unused cache)
docker buildx prune --all

# 3. Nuclear option (force without confirmation)
docker buildx prune --all --force

# 4. Filter by age
docker buildx prune --filter "until=24h"

# 5. Keep recent builds
docker buildx prune --keep-storage 10GB
```

### Check Cache Usage Before Pruning

```bash
# See what will be removed
docker buildx du

# Example output:
# ID            RECLAIMABLE  SIZE     LAST ACCESSED
# abc123...     true         1.2GB    2 hours ago
# def456...     true         856MB    1 day ago
# Total:                     2.1GB
```

## Advanced: Targeted Cache Clearing

Sometimes you don't want to nuke everything. Here's how to be more surgical:

### Clear Cache for Specific Builder

```bash
# List builders
docker buildx ls

# Remove specific builder
docker buildx rm mybuilder

# Recreate it
docker buildx create --name mybuilder --use
```

### Clear Only Old Cache

```bash
# Remove cache older than 7 days
docker buildx prune --filter "until=168h" --force

# Remove cache taking up more than 5GB
docker buildx prune --filter "unused-for=72h" --keep-storage 5GB
```

## Preventing Cache Issues

### 1. Use Cache Mounts (Recommended)

```dockerfile
# Instead of copying package files multiple times
FROM node:18 AS builder

WORKDIR /app

# Use cache mount for npm
RUN --mount=type=cache,target=/root/.npm \
    --mount=type=bind,source=package.json,target=package.json \
    --mount=type=bind,source=package-lock.json,target=package-lock.json \
    npm ci

COPY . .
RUN npm run build
```

### 2. Layer Optimization

```dockerfile
# Bad: Invalidates cache on any file change
COPY . .
RUN npm install && npm run build

# Good: Separate dependency installation
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build
```

### 3. Use .dockerignore

```bash
# .dockerignore
node_modules
.git
.env
*.log
dist
build
.cache
```

### 4. Explicit Cache Busting

```dockerfile
# Add a build arg for cache busting
ARG CACHE_BUST=1

# Use it before steps that should always run
RUN echo "Cache bust: $CACHE_BUST" && \
    apt-get update && \
    apt-get install -y some-package
```

Build with:
```bash
docker buildx build \
  --build-arg CACHE_BUST=$(date +%s) \
  -t myapp:latest \
  .
```

## Multi-Stage Build Cache Issues

Multi-stage builds can have particularly tricky caching problems:

```dockerfile
# Stage 1: Base
FROM node:18 AS base
WORKDIR /app
COPY package*.json ./
RUN npm ci

# Stage 2: Builder (caching issue here!)
FROM base AS builder
COPY . .
RUN npm run build

# Stage 3: Production
FROM node:18-slim
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/node_modules ./node_modules
CMD ["node", "dist/server.js"]
```

**Problem:** Changes to source code don't invalidate the `base` stage cache, but the `builder` stage might use stale intermediate results.

**Solution:** Force rebuild of specific stages:

```bash
docker buildx build \
  --target builder \
  --no-cache \
  -t myapp:builder \
  .

docker buildx build \
  --target production \
  -t myapp:latest \
  .
```

## BuildX vs Traditional Docker Build

| Feature | docker build | docker buildx build |
|---------|-------------|---------------------|
| Multi-platform | ❌ | ✅ |
| Build cache | Local only | Distributed cache support |
| Parallel builds | Limited | Full parallel |
| Cache backends | Local | Local, registry, S3, etc. |
| Build secrets | --secret | Enhanced --secret |

### Migrating from docker build

```bash
# Old way
docker build --no-cache -t myapp:latest .

# New way (equivalent)
docker buildx build --no-cache --load -t myapp:latest .

# New way (better)
docker buildx prune --all
docker buildx build --load -t myapp:latest .
```

## Debugging Build Issues

### Enable BuildKit Debug Mode

```bash
# Set debug mode
export BUILDKIT_PROGRESS=plain

# Run build with verbose output
docker buildx build \
  --progress=plain \
  --no-cache \
  -t myapp:latest \
  .
```

### Inspect Build History

```bash
# See detailed layer information
docker history myapp:latest

# Inspect specific layer
docker inspect myapp:latest | jq '.[0].RootFS.Layers'
```

### Interactive Debugging

```bash
# Build up to a specific stage
docker buildx build \
  --target builder \
  --progress=plain \
  -t myapp:debug \
  .

# Run and inspect
docker run -it myapp:debug /bin/bash
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Docker Build

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2

      - name: Cache Docker layers
        uses: actions/cache@v3
        with:
          path: /tmp/.buildx-cache
          key: ${{ runner.os }}-buildx-${{ github.sha }}
          restore-keys: |
            ${{ runner.os }}-buildx-

      - name: Build and push
        uses: docker/build-push-action@v4
        with:
          context: .
          platforms: linux/amd64,linux/arm64
          push: true
          tags: myregistry.com/myapp:latest
          cache-from: type=local,src=/tmp/.buildx-cache
          cache-to: type=local,dest=/tmp/.buildx-cache-new,mode=max

      # Prevent cache from growing indefinitely
      - name: Move cache
        run: |
          rm -rf /tmp/.buildx-cache
          mv /tmp/.buildx-cache-new /tmp/.buildx-cache
```

### GitLab CI

```yaml
build:
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - docker buildx create --use
  script:
    - docker buildx prune --all --force
    - docker buildx build
        --platform linux/amd64,linux/arm64
        --tag $CI_REGISTRY_IMAGE:$CI_COMMIT_TAG
        --push
        .
  only:
    - tags
```

## Performance Tips

### 1. Use Registry Cache

```bash
# Enable registry cache backend
docker buildx build \
  --cache-from type=registry,ref=myregistry.com/myapp:cache \
  --cache-to type=registry,ref=myregistry.com/myapp:cache,mode=max \
  -t myregistry.com/myapp:latest \
  --push \
  .
```

### 2. Local Cache Directory

```bash
# Use local directory for cache
docker buildx build \
  --cache-from type=local,src=/tmp/buildx-cache \
  --cache-to type=local,dest=/tmp/buildx-cache,mode=max \
  -t myapp:latest \
  --load \
  .
```

### 3. Parallel Builds

```bash
# Build multiple images in parallel
docker buildx bake -f docker-bake.hcl
```

```hcl
// docker-bake.hcl
group "default" {
  targets = ["app", "worker", "scheduler"]
}

target "app" {
  dockerfile = "Dockerfile.app"
  tags = ["myapp:latest"]
}

target "worker" {
  dockerfile = "Dockerfile.worker"
  tags = ["myworker:latest"]
}

target "scheduler" {
  dockerfile = "Dockerfile.scheduler"
  tags = ["myscheduler:latest"]
}
```

## Troubleshooting Common Issues

### Issue 1: "multiple platforms feature is currently not supported"

```bash
# Error when using --load with multiple platforms
# Solution: Use --push instead
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t myregistry.com/myapp:latest \
  --push \
  .
```

### Issue 2: BuildKit version mismatch

```bash
# Update BuildKit
docker buildx install

# Or recreate builder
docker buildx rm mybuilder
docker buildx create --name mybuilder --use
```

### Issue 3: Out of disk space

```bash
# Check disk usage
docker system df
docker buildx du

# Clean up
docker system prune -a --volumes
docker buildx prune --all
```

## Best Practices Summary

1. **Regular Cleanup** - Schedule periodic cache pruning
   ```bash
   # Add to crontab
   0 2 * * * docker buildx prune --filter "until=168h" --force
   ```

2. **Use .dockerignore** - Prevent unnecessary context
3. **Optimize Layer Order** - Put changing layers last
4. **Multi-stage Builds** - Separate build and runtime dependencies
5. **Cache Mounts** - Use for package managers
6. **Explicit Tags** - Don't rely on `:latest`
7. **Scan Images** - Check for vulnerabilities
   ```bash
   docker scan myapp:latest
   ```

## Conclusion

When Docker builds are behaving unexpectedly:

```bash
# The nuclear option
docker buildx prune --all --force
docker builder prune --all --force

# Then rebuild fresh
docker buildx build --load -t myapp:latest .
```

This ensures:
- ✅ No stale cached layers
- ✅ Clean build context
- ✅ Consistent results
- ✅ Proper layer invalidation

Just remember: with great cache clearing comes great build times. Use judiciously in development, but don't hesitate to use it when debugging mysterious issues!

## Resources

- [Docker BuildX Documentation](https://docs.docker.com/buildx/working-with-buildx/)
- [BuildKit Documentation](https://github.com/moby/buildkit)
- [Docker Build Cache](https://docs.docker.com/build/cache/)
- [Multi-platform Builds](https://docs.docker.com/build/building/multi-platform/)

Happy building! 🐳
