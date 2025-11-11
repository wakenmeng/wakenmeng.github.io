---
layout: post
title: "Fixing Emacs-mac 'file-missing doing vfork' Error"
date: 2022-10-15
author: Waken
tags: [emacs, debugging, doom-emacs]
comments: true
---

Emacs suddenly started throwing `file-missing doing vfork` errors on startup. Turned out to be an emacs-sqlite compilation issue.

<!-- more -->

## The Problem

Emacs GUI wouldn't start properly, showing error:
```
file-missing doing vfork
```

## Debug Process

Normally I'd run `emacs --debug-init`, but that doesn't launch the GUI on macOS. Had to find the actual binary:

```bash
# Navigate to the actual Emacs executable
cd /Applications/Emacs.app/Contents/MacOS
./Emacs --debug-init
```

This showed the real culprit: org-roam database sync was failing when calling `emacsql-sqlite`.

## The Cause

Issue was documented in [Doom #6809](https://github.com/doomemacs/doomemacs/issues/6809#issuecomment-1251148601) - emacs-sqlite module needed to be rebuilt.

## The Fix

**Quick way** (manual rebuild):
```elisp
M-x emacsql-sqlite-compile
```

**Better way** from [Doom #7099](https://github.com/doomemacs/doomemacs/issues/7099#issuecomment-1439132382):
```bash
# Rebuild all native modules
doom sync
```

Restart Emacs. Database sync works again.

## Lesson

When Emacs GUI won't start with cryptic errors:
1. Don't use `emacs --debug-init` from terminal
2. Run the actual binary in `/Applications/.../MacOS/` with `--debug-init`
3. Check for native module compilation issues

Native compiled modules (like emacsql-sqlite) can break after system updates or Emacs upgrades.
