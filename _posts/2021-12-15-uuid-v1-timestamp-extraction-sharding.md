---
layout: post
title: "Extracting Timestamps from UUID v1 for Database Sharding"
date: 2021-12-15
author: Waken
tags: [postgresql, uuid, sharding, database]
comments: true
---

UUID v1 embeds timestamps. Here's how to extract them in PostgreSQL for time-based sharding.

<!-- more -->

## The Problem

Sharding databases by time is common. UUID v1 contains timestamp data, but it's encoded in a weird format.

Need to extract the timestamp to partition data by date.

## UUID v1 Structure

UUID v1 format: `time_low-time_mid-time_hi_and_version-clock_seq-node`

Example: `550e8400-e29b-41d4-a716-446655440000`

Timestamp is split across three segments:
- Positions 1-8: `time_low` (lowest 32 bits)
- Positions 10-13: `time_mid` (middle 16 bits)
- Positions 15-18: `time_hi_and_version` (high 12 bits + 4-bit version)

Need to reassemble: `time_hi` + `time_mid` + `time_low`

## The Function

PostgreSQL function to extract timestamp from UUID v1:

```sql
CREATE OR REPLACE FUNCTION uuid_v1_timestamp(_uuid UUID)
RETURNS TIMESTAMP WITH TIME ZONE AS
$$
SELECT to_timestamp(
    (
        ('x' || lpad(h, 16, '0'))::bit(64)::bigint::double precision -
        122192928000000000
    ) / 10000000
)
FROM (
    SELECT substring(u FROM 16 FOR 3) ||
           substring(u FROM 10 FOR 4) ||
           substring(u FROM 1 FOR 8) AS h
    FROM (VALUES (_uuid::text)) s (u)
) s;
$$
LANGUAGE sql;
```

## How It Works

**1. Reassemble timestamp:**
```sql
substring(u FROM 16 FOR 3) ||  -- time_hi
substring(u FROM 10 FOR 4) ||  -- time_mid
substring(u FROM 1 FOR 8)      -- time_low
```

**2. Convert to bigint:**
- Pad to 16 hex digits
- Cast through bit(64) to bigint

**3. Adjust epoch:**
- UUID v1 uses October 15, 1582 as epoch
- Subtract `122192928000000000` (100-ns intervals to Unix epoch)
- Divide by 10000000 (convert to seconds)

**4. Convert to timestamp:**
- `to_timestamp()` gives PostgreSQL timestamp

## Usage

```sql
SELECT uuid_v1_timestamp('550e8400-e29b-41d4-a716-446655440000');

-- Result: 2011-02-03 04:05:06.08-05
```

## Sharding with Timestamp

Create partitions based on extracted timestamp:

```sql
-- Monthly partitions
CREATE TABLE events_2024_01 PARTITION OF events
FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- Route inserts
INSERT INTO events (id, data)
VALUES (
    gen_random_uuid(),  -- or your UUID v1 generator
    'event data'
)
WHERE uuid_v1_timestamp(id) >= '2024-01-01'
  AND uuid_v1_timestamp(id) < '2024-02-01';
```

## Inspiration

Inspired by Instagram's sharding approach: [Sharding & IDs at Instagram](https://instagram-engineering.com/sharding-ids-at-instagram-1cf5a71e5a5c)

Instagram used custom IDs with embedded timestamps. UUID v1 gives similar benefits with standard format.

## Modern Alternative

Consider UUIDv7 (PostgreSQL 18+) which has better timestamp ordering:

```sql
-- UUIDv7 has timestamp-ordered first 48 bits
SELECT gen_uuid_v7();
```

But UUID v1 works great if you're already using it.

Extracting timestamps from IDs unlocks time-based sharding strategies.
