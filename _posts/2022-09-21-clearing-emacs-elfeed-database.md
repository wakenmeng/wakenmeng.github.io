---
layout: post
title: "How to Clear Emacs Elfeed Database"
date: 2022-09-21
author: Waken
tags: [emacs, elfeed, rss]
comments: true
---

Elfeed (RSS reader for Emacs) was showing way too many old entries. Here's how to clear it.

<!-- more -->

## The Problem

Thousands of old RSS entries cluttering my elfeed buffer. Search was slow, scrolling was laggy.

## The Solution

Two steps:

```bash
# 1. Quit Emacs

# 2. Delete the database directory
rm -rf ~/.elfeed
# or wherever your elfeed-db-directory points to
```

Restart Emacs. Elfeed rebuilds from scratch, fetching only recent entries.

## Finding Your Database Location

If unsure where elfeed stores data:

```elisp
M-x eval-expression
elfeed-db-directory
```

Default is `~/.elfeed`.

## Alternative: Selective Deletion

Don't want to nuke everything? Inside Emacs:

```elisp
;; Delete old entries (older than 30 days)
(elfeed-db-gc)

;; Or manually from elfeed buffer:
;; Mark entries with 'r' (read)
;; They'll be cleaned up by gc
```

But honestly, deleting the whole database is simpler if you just want a clean slate.

## After Deletion

First `elfeed-update` will take a bit longer as it refetches feeds. After that, back to normal.

My elfeed config for reference:

```elisp
(setq elfeed-db-directory "~/.elfeed")
(setq elfeed-feeds
      '("https://hnrss.org/frontpage"
        "https://news.ycombinator.com/rss"
        ;; ... more feeds
        ))
```

Quick reset when things get cluttered.
