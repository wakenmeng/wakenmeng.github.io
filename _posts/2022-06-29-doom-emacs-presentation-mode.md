---
layout: post
title: "Doom Emacs: org-present is Actually org-tree-slide"
date: 2022-06-29
author: Waken
tags: [emacs, doom-emacs, org-mode]
comments: true
---

Confused moment: Doom Emacs `org +present` module doesn't use the `org-present` package. It uses `org-tree-slide-mode` instead.

<!-- more -->

## The Confusion

Saw the `+present` flag in Doom's org module config. Assumed it would use the `org-present` package. Nope!

## What It Actually Uses

Doom's presentation mode is `org-tree-slide-mode`.

**Activate:**
```
SPC t p
# or
M-x org-tree-slide-mode
```

**Navigate:**
```
Ctrl + Left/Right arrows to slide between headings
```

## Difference

**org-present:**
- Minimal, simple
- Full-screen slides
- Basic transitions

**org-tree-slide-mode:**
- More features
- Three different slide styles
- Better integration with Doom

## Why Doom Uses org-tree-slide

Makes sense for Doom's workflow:
- Works better with evil mode
- More customizable
- Plays nice with Doom's keybindings

## Quick Demo

Any org file with headings becomes a presentation:

```org
* First Slide
Content here

* Second Slide
More content

** Sub-heading
This becomes a nested slide
```

Hit `SPC t p` and you're presenting.

That's it. Just a naming gotcha I ran into.
