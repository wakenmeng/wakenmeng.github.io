---
layout: post
title: "Profiling Slow Emacs Startup"
date: 2022-09-09
author: Waken
tags: [emacs, performance, debugging]
comments: true
---

Emacs was taking forever to start. Found the built-in profiler to track down the culprit.

<!-- more -->

## The Commands

```elisp
M-x profiler-start
# Do the slow thing (restart Emacs, open files, etc.)
M-x profiler-report
```

That's it. Shows you what's eating CPU time.

## What I Found

In my case:

```
- 89% org-mode
  - 67% org-roam-db-sync
    - 45% emacsql queries
```

org-roam was syncing on every startup. Way too slow.

## The Fix

```elisp
;; Don't auto-sync on startup
(setq org-roam-db-autosync-mode nil)

;; Manually sync when needed
M-x org-roam-db-sync
```

Startup time: 15s → 2s

## How It Works

Profiler samples your Emacs process and shows:
- Which functions are called most
- Where time is spent
- Call stack breakdown

Interactive report lets you expand/collapse call trees.

## Other Uses

Not just for startup:
- Slow file opening
- Laggy commands
- Memory leaks (use `profiler-start` with memory profiling)

## Alternative: esup

For startup-specific profiling:

```bash
# Install esup
M-x package-install RET esup

# Profile init file
M-x esup
```

Shows timing for each line in your init.el.

But built-in profiler works for everything, so I usually just use that.

Quick tip: if Emacs is slow, profile it. Don't guess.
