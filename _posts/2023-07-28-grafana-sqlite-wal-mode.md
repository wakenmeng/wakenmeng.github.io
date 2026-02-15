---
layout: post
title: "Speeding Up Grafana with SQLite WAL Mode"
date: 2023-07-28
author: Waken
tags: [grafana, sqlite, performance]
comments: true
---

Grafana was feeling sluggish. Enabled SQLite WAL mode and noticed immediate improvement.

<!-- more -->

## The Change

In Grafana config, enable WAL (Write-Ahead Logging) for SQLite:

```ini
[database]
wal = true
```

Restart Grafana.

## What is WAL?

Write-Ahead Logging is an SQLite optimization:

- **Normal mode**: Writes lock the entire database
- **WAL mode**: Writes go to a separate log file, readers aren't blocked

Result: Better concurrency, especially for read-heavy workloads.

## Why It Helps Grafana

Grafana does lots of concurrent reads:
- Dashboard queries
- Alerting checks
- User sessions
- Plugin data

Without WAL, writes (like saving dashboards) block all these reads.

With WAL, reads continue uninterrupted.

## The Trade-off

WAL adds a second file (`database.db-wal`) alongside your database. Slightly more disk I/O.

But for most Grafana setups, the performance gain is worth it.

## When to Use

Use WAL if:
- Multiple concurrent users
- Frequent dashboard updates
- Read-heavy workload

Skip it if:
- Single user
- Mostly static dashboards
- Using MySQL/Postgres instead of SQLite

For production Grafana with SQLite backend, WAL should be default.
