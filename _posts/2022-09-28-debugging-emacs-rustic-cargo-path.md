---
layout: post
title: "Debugging Emacs rustic-mode: Cannot Find Cargo"
date: 2022-09-28
author: Waken
tags: [emacs, rust, debugging]
comments: true
---

Spent hours debugging why `rustic-format-buffer` wasn't working in Emacs. The error? "Program not found: No such file or directory, cargo"

<!-- more -->

## The Problem

```
rustic-format-buffer
Error: Program not found: No such file or directory, cargo
```

Cargo was definitely installed and working in my terminal (zsh). But Emacs couldn't see it.

## The Debug Journey

Tried the systematic approach:

1. Checked if rustic-mode was actually running (yes)
2. Searched rustic repo issues (nothing relevant)
3. Used `M-x debug-on-error` to trace the error
4. Found it failed in `rustic-buffer-workspace` - literally couldn't find cargo
5. Checked Emacs PATH: `M-x getenv RET PATH`

**Aha moment**: Emacs PATH didn't include `~/.cargo/bin`!

## Why This Happens

Emacs GUI on macOS doesn't inherit shell environment variables. My zsh had cargo in PATH, but Emacs launched from Dock didn't.

## The Fix

Install [exec-path-from-shell](https://github.com/purcell/exec-path-from-shell):

```elisp
(use-package exec-path-from-shell
  :config
  (when (memq window-system '(mac ns x))
    (exec-path-from-shell-initialize)))
```

Restart Emacs. Now `rustic-format-buffer` works perfectly.

## Lesson

Emacs GUI environment ≠ Terminal environment. Always check `M-x getenv` when programs "aren't found".

Found the solution in [this Reddit thread](https://www.reddit.com/r/emacs/comments/w66woa/trying_to_setup_a_rust_environment_rustic_keeps/).
