---
layout: post
title: "tmux: Catppuccin Theme Setup"
date: 2024-01-02
author: Waken
tags: [tmux, theme, terminal]
comments: true
---

Found a nice tmux theme and programming font combination.

<!-- more -->

## The Theme

Using Catppuccin theme for tmux: [catppuccin/tmux](https://github.com/catppuccin/tmux)

Clean pastel colors, easy on the eyes.

## Setup

Add to `.tmux.conf`:

```bash
# Using TPM (tmux plugin manager)
set -g @plugin 'catppuccin/tmux'
```

Then install:
```bash
# Press prefix + I (capital i) to install
```

## Font Choice

Switched to JetBrains Mono: [programmingfonts.org/#jetbrainsmono](https://www.programmingfonts.org/#jetbrainsmono)

Features:
- Ligatures for common programming symbols
- Clear distinction between similar characters (0/O, 1/l/I)
- Free and open source

## Install JetBrains Mono

**macOS:**
```bash
brew tap homebrew/cask-fonts
brew install --cask font-jetbrains-mono
```

**Linux:**
```bash
# Download from GitHub releases
# Or use package manager
sudo apt install fonts-jetbrains-mono
```

Then set in your terminal emulator preferences.

## Why This Combo Works

- Catppuccin: soothing colors for long coding sessions
- JetBrains Mono: readable, designed for code

Together they make tmux sessions much more pleasant.

Screenshot your terminal and you'll understand why people obsess over themes and fonts.
