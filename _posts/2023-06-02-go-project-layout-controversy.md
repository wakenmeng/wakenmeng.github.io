---
layout: post
title: "Go Project Layout: The Controversy"
date: 2023-06-02
author: Waken
tags: [golang, project-structure, best-practices]
comments: true
---

Was researching Go project layouts. Found an interesting debate about golang-standards/project-layout.

<!-- more -->

## The Repository

[golang-standards/project-layout](https://github.com/golang-standards/project-layout/issues/117) is a popular repo showing a "standard" Go project structure with `/cmd`, `/pkg`, `/internal`, etc.

**Problem:** It's not actually a standard. Just one person's opinion.

## The Debate

Issue #117 discusses concerns:
- Misleading name (not endorsed by Go team)
- Overly complex for most projects
- Cargo-cult adoption without understanding

Many developers copy this structure blindly, thinking it's official.

## What Go Actually Recommends

From the Go team: Keep it simple.

Eli Bendersky's [simple Go project layout](https://eli.thegreenplace.net/2019/simple-go-project-layout-with-modules/):

```
myproject/
  main.go           # or cmd/myapp/main.go if multiple binaries
  handler.go
  handler_test.go
  go.mod
```

That's it. Add directories as you need them.

## When to Add Structure

**Small project:**
```
myproject/
  main.go
  user.go
  post.go
```

**Multiple binaries:**
```
myproject/
  cmd/
    server/main.go
    worker/main.go
  internal/
    user.go
    post.go
```

**Large project:**
Add `/internal`, `/pkg`, `/api` as needed. But start simple.

## Lesson

Don't cargo-cult project structures. Start minimal, add complexity only when necessary.

The "standard" layout works for large teams and complex projects. For most projects, it's overkill.
