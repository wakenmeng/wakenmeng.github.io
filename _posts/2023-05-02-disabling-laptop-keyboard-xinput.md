---
layout: post
title: "Disabling Laptop Built-in Keyboard with xinput"
date: 2023-05-02
author: Waken
tags: [linux, xinput, keyboard]
comments: true
---

Needed to disable my laptop's built-in keyboard when using an external one. Found a simple xinput solution.

<!-- more -->

## The Problem

Using external keyboard but laptop keys still register. Accidental keypresses are annoying.

## The Solution

Use `xinput` to disable the built-in keyboard:

```bash
# 1. List all input devices
xinput list

# Find your keyboard, looks like:
# ⎜   ↳ AT Translated Set 2 keyboard    id=12 [slave keyboard (3)]
```

```bash
# 2. Disable it
xinput set-prop "AT Translated Set 2 keyboard" "Device Enabled" 0

# 3. Re-enable when needed
xinput set-prop "AT Translated Set 2 keyboard" "Device Enabled" 1
```

## Make It Easier with Aliases

Add to `.bashrc` or `.zshrc`:

```bash
alias kb-off='xinput set-prop "AT Translated Set 2 keyboard" "Device Enabled" 0'
alias kb-on='xinput set-prop "AT Translated Set 2 keyboard" "Device Enabled" 1'
```

Source the file:
```bash
source ~/.zshrc  # or ~/.bashrc
```

Now just type `kb-off` or `kb-on`.

## Important Note

**These changes are temporary.** System reboot resets everything.

For permanent solution, create a script and set it to run at startup.

**Safety tip**: Always have a way to re-enable the keyboard! If your external keyboard disconnects, you'll need the built-in one.
