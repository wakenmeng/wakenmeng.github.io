---
layout: post
title: "dumb-init: Simple Init System for Docker"
date: 2023-06-12
author: Waken
tags: [docker, containers, init, process-management]
comments: true
---

Discovered dumb-init for handling process signals properly in Docker containers.

<!-- more -->

## The Problem

Docker containers often run a single process as PID 1. But PID 1 has special responsibilities in Unix:
- Reaping zombie processes
- Forwarding signals properly

Most applications aren't designed to be PID 1. They don't handle these tasks correctly.

## What dumb-init Does

[dumb-init](https://github.com/Yelp/dumb-init) is a minimal init system that:
- Runs as PID 1
- Forwards signals to your app
- Reaps zombie processes
- Stays out of the way

By Yelp, battle-tested in production.

## Usage

**In Dockerfile:**

```dockerfile
FROM ubuntu:latest

# Install dumb-init
RUN apt-get update && apt-get install -y dumb-init

# Use dumb-init as entrypoint
ENTRYPOINT ["dumb-init", "--"]

# Your application
CMD ["python", "app.py"]
```

Or with Alpine:

```dockerfile
FROM alpine:latest

RUN apk add --no-cache dumb-init

ENTRYPOINT ["dumb-init", "--"]
CMD ["./myapp"]
```

## What It Fixes

**Without dumb-init:**
- `docker stop` takes 10 seconds (SIGTERM timeout → SIGKILL)
- Zombie processes accumulate
- Signal handling is broken

**With dumb-init:**
- Clean shutdown
- Proper signal forwarding
- Zombie reaping works

## Alternative: `--init` Flag

Docker has built-in init:

```bash
docker run --init myimage
```

Uses `tini` under the hood. Similar to dumb-init.

But if you want explicit control in your Dockerfile, use dumb-init directly.

## When You Need It

Use an init system when:
- Running shell scripts as containers
- Application spawns child processes
- Need proper signal handling
- `docker stop` is slow

Small tool, solves real problems.
