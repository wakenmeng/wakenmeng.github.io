---
layout: post
title: "Atlas: Modern Database Schema Management Done Right"
date: 2023-12-02
author: Waken
tags: [database, devops, migrations, schema-management, infrastructure]
comments: true
toc: true
---

Managing database schemas across environments has always been a pain point in software development. Manual migrations, script versioning, and schema drift are common sources of production incidents. Recently, I discovered [Atlas](https://atlasgo.io/), a modern database schema management tool that applies DevOps principles to database changes. Here's why it's worth your attention.

<!-- more -->

## The Problem with Traditional Migrations

If you've worked with databases in production, you've likely experienced these pain points:

### 1. Manual Migration Hell

```sql
-- migration_001.sql
ALTER TABLE users ADD COLUMN email VARCHAR(255);

-- Oops, forgot the NOT NULL constraint!
-- migration_002.sql
ALTER TABLE users MODIFY COLUMN email VARCHAR(255) NOT NULL;

-- Wait, what about existing rows?
-- migration_003_fix.sql
UPDATE users SET email = CONCAT('user', id, '@example.com') WHERE email IS NULL;
ALTER TABLE users MODIFY COLUMN email VARCHAR(255) NOT NULL;
```

This iterative approach leads to:
- Scattered migration files
- Unclear final schema state
- Difficult rollbacks
- Risk of data loss

### 2. Schema Drift

```bash
# Production database
mysql> DESCRIBE users;
+-------+--------------+
| email | varchar(255) |
+-------+--------------+

# Staging database
mysql> DESCRIBE users;
+-------+--------------+
| email | varchar(100) | # Oops! Different size
+-------+--------------+
```

Environments diverge over time, causing subtle bugs and deployment failures.

### 3. Lack of Code Review

SQL migrations often bypass the rigorous code review process:
- No automated checks for destructive changes
- Missing validation for data-dependent modifications
- No visibility into impact before deployment

## Enter Atlas

Atlas solves these problems by treating database schemas as code, with proper versioning, testing, and deployment automation.

### Key Features

1. **Declarative Schema Definition** - Define your desired state, not the steps to get there
2. **Automatic Migration Generation** - Atlas calculates the diff and generates safe SQL
3. **Schema Drift Detection** - Monitor and alert on unexpected changes
4. **Built-in Safety Checks** - Lint migrations for destructive operations
5. **Multi-Database Support** - PostgreSQL, MySQL, SQLite, SQL Server, and more
6. **Language Agnostic** - Define schemas in HCL, SQL, or your programming language

## Getting Started with Atlas

### Installation

```bash
# macOS
brew install ariga/tap/atlas

# Linux
curl -sSf https://atlasgo.sh | sh

# Windows
curl https://release.ariga.io/atlas/atlas-windows-amd64-latest.exe -o atlas.exe

# Docker
docker pull arigaio/atlas

# Verify installation
atlas version
```

### Basic Workflow

#### 1. Define Your Schema

Atlas supports multiple formats. Here's an HCL example:

```hcl
// schema.hcl
schema "public" {
}

table "users" {
  schema = schema.public

  column "id" {
    type = int
    auto_increment = true
  }

  column "email" {
    type = varchar(255)
    null = false
  }

  column "name" {
    type = varchar(100)
    null = false
  }

  column "created_at" {
    type = timestamp
    default = sql("CURRENT_TIMESTAMP")
  }

  primary_key {
    columns = [column.id]
  }

  index "idx_email" {
    columns = [column.email]
    unique = true
  }
}

table "posts" {
  schema = schema.public

  column "id" {
    type = int
    auto_increment = true
  }

  column "user_id" {
    type = int
    null = false
  }

  column "title" {
    type = varchar(255)
    null = false
  }

  column "content" {
    type = text
  }

  column "published_at" {
    type = timestamp
    null = true
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "fk_user" {
    columns = [column.user_id]
    ref_columns = [table.users.column.id]
    on_delete = CASCADE
  }

  index "idx_user_published" {
    columns = [column.user_id, column.published_at]
  }
}
```

Or use plain SQL:

```sql
-- schema.sql
CREATE TABLE users (
  id INT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(100) NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_email (email)
);

CREATE TABLE posts (
  id INT PRIMARY KEY AUTO_INCREMENT,
  user_id INT NOT NULL,
  title VARCHAR(255) NOT NULL,
  content TEXT,
  published_at TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_user_published (user_id, published_at)
);
```

#### 2. Inspect Current Database State

```bash
# Inspect current database schema
atlas schema inspect \
  -u "mysql://user:pass@localhost:3306/mydb" \
  > current.hcl

# Compare desired vs current state
atlas schema diff \
  --from "file://current.hcl" \
  --to "file://schema.hcl"
```

#### 3. Apply Changes

```bash
# Dry run - see what would change
atlas schema apply \
  -u "mysql://user:pass@localhost:3306/mydb" \
  --to "file://schema.hcl" \
  --dry-run

# Apply changes
atlas schema apply \
  -u "mysql://user:pass@localhost:3306/mydb" \
  --to "file://schema.hcl" \
  --auto-approve
```

## Versioned Migrations

For production environments, use versioned migrations instead of declarative apply:

### Initialize Migration Directory

```bash
atlas migrate diff initial \
  --dir "file://migrations" \
  --to "file://schema.hcl" \
  --dev-url "docker://mysql/8/mydb"
```

This generates:

```sql
-- migrations/20231202120000_initial.sql
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `email` varchar(255) NOT NULL,
  `name` varchar(100) NOT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `idx_email` (`email`)
);

CREATE TABLE `posts` (
  `id` int NOT NULL AUTO_INCREMENT,
  `user_id` int NOT NULL,
  `title` varchar(255) NOT NULL,
  `content` text,
  `published_at` timestamp NULL,
  PRIMARY KEY (`id`),
  KEY `idx_user_published` (`user_id`,`published_at`),
  CONSTRAINT `fk_user` FOREIGN KEY (`user_id`)
    REFERENCES `users` (`id`) ON DELETE CASCADE
);
```

### Make Schema Changes

Update `schema.hcl`:

```hcl
table "users" {
  // ... existing columns ...

  column "bio" {
    type = text
    null = true
  }

  column "avatar_url" {
    type = varchar(500)
    null = true
  }
}
```

Generate new migration:

```bash
atlas migrate diff add_user_profile \
  --dir "file://migrations" \
  --to "file://schema.hcl" \
  --dev-url "docker://mysql/8/mydb"
```

### Apply Migrations

```bash
# Apply to target database
atlas migrate apply \
  --dir "file://migrations" \
  --url "mysql://user:pass@localhost:3306/mydb"

# Check migration status
atlas migrate status \
  --dir "file://migrations" \
  --url "mysql://user:pass@localhost:3306/mydb"
```

## Advanced Features

### Schema Linting

Catch potential issues before deployment:

```bash
atlas migrate lint \
  --dir "file://migrations" \
  --dev-url "docker://mysql/8/mydb" \
  --latest 1
```

Atlas checks for:
- **Destructive changes**: Dropping columns or tables
- **Data-dependent changes**: Adding NOT NULL without defaults
- **Backward incompatible changes**: Changing column types
- **Index optimization**: Missing indexes on foreign keys

Example output:

```
Analyzing changes from version 20231202120000 to 20231202130000 (1 migration in total):

  -- analyzing version 20231202130000
    -- destructive changes detected:
      -- L1: Dropping non-virtual column "old_email"
    -- data dependent changes detected:
      -- L5: Adding a non-nullable "varchar" column "phone" will fail
              in case table "users" is not empty

  -------------------------
  -- 2 reports

atlas migrate lint summary:
  -- 1 version with errors
```

### CI/CD Integration

#### GitHub Actions

```yaml
# .github/workflows/atlas.yml
name: Atlas Migration Check

on:
  pull_request:
    paths:
      - 'migrations/**'
      - 'schema.hcl'

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Atlas
        uses: ariga/setup-atlas@v0

      - name: Lint migrations
        run: |
          atlas migrate lint \
            --dir "file://migrations" \
            --dev-url "docker://mysql/8/test" \
            --latest 1
        env:
          ATLAS_TOKEN: ${{ secrets.ATLAS_TOKEN }}

      - name: Apply to staging
        if: github.event_name == 'push'
        run: |
          atlas migrate apply \
            --dir "file://migrations" \
            --url ${{ secrets.STAGING_DATABASE_URL }}
```

#### GitLab CI

```yaml
# .gitlab-ci.yml
atlas-lint:
  image: arigaio/atlas:latest
  stage: test
  script:
    - atlas migrate lint
        --dir "file://migrations"
        --dev-url "docker://postgres/15/test"
        --latest 1
  only:
    changes:
      - migrations/**
      - schema.hcl

atlas-deploy:
  image: arigaio/atlas:latest
  stage: deploy
  script:
    - atlas migrate apply
        --dir "file://migrations"
        --url "$PRODUCTION_DATABASE_URL"
  only:
    - main
  when: manual
```

### Schema from ORM Models

Atlas can generate schemas from popular ORM frameworks:

#### Go (GORM/Ent)

```go
// models/user.go
type User struct {
    gorm.Model
    Email     string `gorm:"uniqueIndex;not null"`
    Name      string `gorm:"not null"`
    Bio       string
    AvatarURL string
    Posts     []Post
}

type Post struct {
    gorm.Model
    UserID      uint
    User        User
    Title       string `gorm:"not null"`
    Content     string `gorm:"type:text"`
    PublishedAt *time.Time
}
```

Generate schema:

```bash
atlas migrate diff \
  --dir "file://migrations" \
  --to "ent://models" \
  --dev-url "docker://mysql/8/dev"
```

#### TypeScript (TypeORM/Prisma)

```typescript
// schema.prisma
model User {
  id        Int      @id @default(autoincrement())
  email     String   @unique
  name      String
  bio       String?
  avatarUrl String?
  posts     Post[]
  createdAt DateTime @default(now())
}

model Post {
  id          Int       @id @default(autoincrement())
  userId      Int
  user        User      @relation(fields: [userId], references: [id])
  title       String
  content     String?   @db.Text
  publishedAt DateTime?

  @@index([userId, publishedAt])
}
```

```bash
atlas migrate diff \
  --dir "file://migrations" \
  --to "prisma://schema.prisma" \
  --dev-url "docker://postgres/15/dev"
```

### Multi-Environment Management

```bash
# Use environment-specific configs
# atlas.hcl
env "local" {
  src = "file://schema.hcl"
  url = "mysql://root:pass@localhost:3306/dev"
  dev = "docker://mysql/8/dev"
}

env "staging" {
  src = "file://schema.hcl"
  url = env("STAGING_DATABASE_URL")
  dev = "docker://mysql/8/dev"

  migration {
    dir = "file://migrations"
  }

  lint {
    destructive {
      error = true
    }
  }
}

env "production" {
  src = "file://schema.hcl"
  url = env("PRODUCTION_DATABASE_URL")
  dev = "docker://mysql/8/dev"

  migration {
    dir = "file://migrations"
  }

  lint {
    destructive {
      error = true
    }
  }

  diff {
    skip {
      drop_schema = true
      drop_table  = true
    }
  }
}
```

Apply to specific environment:

```bash
atlas migrate apply --env production
```

## Real-World Example: Zero-Downtime Deployment

Let's say you need to rename a column without downtime:

### Step 1: Add New Column

```hcl
table "users" {
  column "username" {  // old
    type = varchar(50)
  }

  column "display_name" {  // new
    type = varchar(50)
    null = true
  }
}
```

```bash
atlas migrate diff add_display_name --dir migrations --to file://schema.hcl
atlas migrate apply --env production
```

### Step 2: Dual Write (Application Code)

```go
// Update application to write to both columns
user.Username = username
user.DisplayName = username
db.Save(&user)
```

### Step 3: Backfill Data

```sql
-- migration: backfill_display_name.sql
UPDATE users SET display_name = username WHERE display_name IS NULL;
```

### Step 4: Make Non-Nullable

```hcl
table "users" {
  column "display_name" {
    type = varchar(50)
    null = false  // Now required
  }
}
```

### Step 5: Remove Old Column

```hcl
table "users" {
  column "display_name" {
    type = varchar(50)
    null = false
  }
  // username column removed
}
```

## Comparison with Alternatives

| Feature | Atlas | Flyway | Liquibase | golang-migrate |
|---------|-------|--------|-----------|----------------|
| Declarative Schema | ✅ | ❌ | ✅ | ❌ |
| Auto Migration Gen | ✅ | ❌ | ❌ | ❌ |
| Schema Linting | ✅ | ❌ | Limited | ❌ |
| Drift Detection | ✅ | Limited | Limited | ❌ |
| ORM Integration | ✅ | ❌ | ❌ | ❌ |
| Open Source | ✅ | ✅ | ✅ | ✅ |
| Language Support | All | Java | Java/XML | Go |

## Best Practices

1. **Use Versioned Migrations in Production**
   - Declarative apply is great for dev
   - Versioned migrations provide auditability

2. **Enable Linting in CI/CD**
   - Catch destructive changes early
   - Block PRs with errors

3. **Test Migrations on Production-like Data**
   - Use anonymized production dumps
   - Test migration performance

4. **Use Dev Database for Diff Generation**
   - Ensures consistent SQL generation
   - Avoids drift in migration files

5. **Version Your Schema Definition**
   - Commit `schema.hcl` to git
   - Track changes alongside application code

6. **Monitor Schema Drift**
   - Run periodic schema inspections
   - Alert on unexpected changes

## Conclusion

Atlas brings modern DevOps practices to database schema management:

- **Infrastructure as Code**: Define schemas declaratively
- **Automated Testing**: Lint migrations before deployment
- **CI/CD Integration**: Automate the entire migration pipeline
- **Multi-Environment**: Manage dev, staging, and production consistently

If you're tired of manual migrations and schema drift, give Atlas a try. It's particularly powerful when combined with ORMs and modern CI/CD workflows.

## Resources

- [Atlas Documentation](https://atlasgo.io/docs)
- [Atlas GitHub](https://github.com/ariga/atlas)
- [Interactive Playground](https://gh.atlasgo.cloud/explore)
- [Migration Guides](https://atlasgo.io/guides)

## Getting Help

```bash
# Built-in help
atlas --help
atlas migrate --help

# Community
# GitHub Discussions: https://github.com/ariga/atlas/discussions
# Discord: https://discord.gg/zZ6sWVg6NT
```

Try Atlas on your next project—your future self will thank you! 🚀
