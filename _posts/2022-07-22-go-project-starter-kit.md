---
layout: post
title: "My Go Project Starter Kit"
date: 2022-07-22
author: Waken
tags: [golang, tools, project-setup]
comments: true
---

Been thinking about my default Go project setup. Here's what I'm settling on.

<!-- more -->

## The Stack

**Database: sqlc**
- Generates type-safe Go code from SQL
- No ORM magic, just SQL
- Compile-time safety

**Tests: gotests**
- Auto-generates table-driven tests
- Saves so much boilerplate

**Router: TBD**
- Still deciding between `chi`, `gin`, or just `net/http`

## Why sqlc

```sql
-- queries.sql
-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC;
```

Generate Go code:

```bash
sqlc generate
```

Get type-safe functions:

```go
user, err := queries.GetUser(ctx, userID)
users, err := queries.ListUsers(ctx)
```

No reflection, no ORM complexity. Just plain SQL and generated Go.

## Why gotests

```bash
# Generate tests for a function
gotests -w -all myfile.go
```

Creates table-driven test templates:

```go
func TestMyFunc(t *testing.T) {
    tests := []struct {
        name string
        args args
        want want
    }{
        // TODO: Add test cases
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test code
        })
    }
}
```

Saves me from typing the same structure every time.

## What About Go Wire?

Looked into [Wire](https://github.com/google/wire) for dependency injection. Conclusion: too complicated for most projects.

**Good for:**
- Platform-level projects
- Multiple third-party service integrations
- Google Cloud Platform scale

**Overkill for:**
- Normal web apps
- Small services
- Most of what I build

I'll stick with simple constructors:

```go
func NewServer(db *sql.DB, logger *log.Logger) *Server {
    return &Server{
        db:     db,
        logger: logger,
    }
}
```

Manual DI is fine for 90% of projects.

## The Template Plan

Thinking about creating a `go-starter` repo with:
- Project structure
- Makefile for common tasks
- Docker compose for local DB
- sqlc config
- CI/CD setup

Not done yet, but that's the idea.

For now, these tools (sqlc + gotests) are my go-to for new projects.
