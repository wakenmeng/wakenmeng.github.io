---
layout: post
title: "Found a Good Database Migration Tool: Atlas"
date: 2023-12-02
author: Waken
tags: [database, devops, migrations]
comments: true
---

Discovered [Atlas](https://atlasgo.io/) today. It's a database schema management tool that actually makes sense.

<!-- more -->

## What It Does

Instead of writing manual migration files, you declare your desired schema and Atlas figures out the migrations.

```hcl
// schema.hcl
table "users" {
  column "id" {
    type = int
    auto_increment = true
  }

  column "email" {
    type = varchar(255)
    null = false
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_email" {
    columns = [column.email]
    unique = true
  }
}
```

Then:

```bash
atlas schema apply \
  -u "mysql://user:pass@localhost:3306/mydb" \
  --to "file://schema.hcl"
```

It calculates the diff and applies it.

## Why I Like It

**1. Declarative**

Don't need to remember what migrations you've written. Just define the end state.

**2. Automatic Diff Generation**

```bash
atlas migrate diff add_email_column \
  --dir "file://migrations" \
  --to "file://schema.hcl" \
  --dev-url "docker://mysql/8/dev"
```

Generates the SQL for you.

**3. Built-in Lint**

```bash
atlas migrate lint \
  --dir "file://migrations" \
  --dev-url "docker://mysql/8/dev"
```

Catches stuff like:
- Dropping columns (destructive!)
- Adding NOT NULL without defaults
- Missing indexes on foreign keys

**4. Works with ORMs**

```bash
# Generate schema from GORM models
atlas migrate diff \
  --dir "file://migrations" \
  --to "ent://models" \
  --dev-url "docker://mysql/8/dev"
```

No more manual sync between code and database schema.

## Basic Workflow

```bash
# 1. Install
brew install ariga/tap/atlas

# 2. Define schema (schema.hcl or use existing SQL)

# 3. Generate migration
atlas migrate diff initial \
  --dir "file://migrations" \
  --to "file://schema.hcl" \
  --dev-url "docker://mysql/8/dev"

# 4. Apply to database
atlas migrate apply \
  --dir "file://migrations" \
  --url "mysql://user:pass@localhost:3306/mydb"
```

## Multi-Environment Config

Create `atlas.hcl`:

```hcl
env "local" {
  src = "file://schema.hcl"
  url = "mysql://root:pass@localhost:3306/dev"
  dev = "docker://mysql/8/dev"
}

env "production" {
  src = "file://schema.hcl"
  url = env("DATABASE_URL")
  dev = "docker://mysql/8/dev"

  lint {
    destructive {
      error = true  // Block destructive changes
    }
  }
}
```

Then: `atlas migrate apply --env production`

## CI/CD Integration

GitHub Actions example:

```yaml
- name: Lint migrations
  run: |
    atlas migrate lint \
      --dir "file://migrations" \
      --dev-url "docker://mysql/8/test"

- name: Apply to staging
  run: |
    atlas migrate apply \
      --dir "file://migrations" \
      --url "${{ secrets.STAGING_DB_URL }}"
```

Catches bad migrations before they hit production.

## Comparison

**vs Flyway/Liquibase:**
- More modern
- Better error messages
- Supports declarative schemas

**vs golang-migrate:**
- Auto-generates migrations
- Built-in safety checks
- ORM integration

**vs Alembic (Python):**
- Language-agnostic
- Better multi-database support

## What I'm Using It For

Managing schema changes across dev/staging/prod. No more "did we run this migration?" or "what's the current schema state?"

Atlas keeps everything in sync.

One gotcha: Needs a "dev database" for diff calculation. Just use Docker:

```bash
--dev-url "docker://mysql/8/dev"
```

It spins up a temporary container automatically.

Worth checking out if you're tired of manual database migrations.
