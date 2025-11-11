---
layout: post
title: "chromedp: Creating Multiple Browser Instances"
date: 2023-06-02
author: Waken
tags: [golang, chromedp, chrome, automation]
comments: true
---

Hit an issue creating multiple Chrome browser instances with chromedp in Go. Same user data directory was the problem.

<!-- more -->

## The Problem

Tried to create multiple browser instances:

```go
ctx1, cancel1 := chromedp.NewContext(context.Background())
ctx2, cancel2 := chromedp.NewContext(context.Background())
```

Second instance wouldn't start properly. It just attached to the existing Chrome session instead of creating a new one.

## The Cause

From [chromedp issue #1147](https://github.com/chromedp/chromedp/issues/1147):

> When you create multiple browsers with the same user_data_dir, Chrome will fail and use the existing Chrome session.

Chrome enforces one process per user data directory. Can't have two instances sharing the same profile folder.

## The Solution

Use different `user-data-dir` for each browser:

```go
import (
    "context"
    "os"
    "github.com/chromedp/chromedp"
)

// Create first browser
opts1 := append(chromedp.DefaultExecAllocatorOptions[:],
    chromedp.UserDataDir("/tmp/chrome-data-1"),
)
allocCtx1, cancel1 := chromedp.NewExecAllocator(context.Background(), opts1...)
ctx1, cancel2 := chromedp.NewContext(allocCtx1)

// Create second browser
opts2 := append(chromedp.DefaultExecAllocatorOptions[:],
    chromedp.UserDataDir("/tmp/chrome-data-2"),
)
allocCtx2, cancel3 := chromedp.NewExecAllocator(context.Background(), opts2...)
ctx2, cancel4 := chromedp.NewContext(allocCtx2)

// Now both browsers run independently
```

Each gets its own profile directory, so both can run simultaneously.

## Cleanup

Remember to clean up temp directories:

```go
defer func() {
    cancel1()
    cancel2()
    cancel3()
    cancel4()
    os.RemoveAll("/tmp/chrome-data-1")
    os.RemoveAll("/tmp/chrome-data-2")
}()
```

## Use Case

Useful for:
- Parallel web scraping
- Testing with different browser profiles
- Isolated automation tasks

Simple fix once you know about it.
