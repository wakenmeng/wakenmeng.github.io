---
layout: post
title: "Tmux: Terminal Sessions That Persist"
date: 2023-05-02
author: Waken
tags: [tmux, terminal, productivity]
comments: true
---

Finally learned tmux properly. Game changer for remote work and managing multiple terminal sessions.

<!-- more -->

## Why Tmux?

**Main benefit:** Detach from sessions without killing processes.

```bash
# SSH into server
ssh server.com

# Start tmux
tmux new -s deploy

# Run long process
./deploy.sh

# Connection drops? No problem!
# Reconnect and reattach:
ssh server.com
tmux attach -t deploy

# Still running! ✅
```

Also: Multiple windows and panes in one terminal.

## Install

```bash
brew install tmux  # macOS
sudo apt install tmux  # Linux
```

## Essential Commands

### Sessions

```bash
# Create
tmux new -s myproject

# Detach (keeps running)
Ctrl-b d

# List sessions
tmux ls

# Reattach
tmux attach -t myproject

# Kill session
tmux kill-session -t myproject
```

### Windows (like tabs)

```bash
Ctrl-b c       # New window
Ctrl-b ,       # Rename window
Ctrl-b n/p     # Next/previous window
Ctrl-b 0-9     # Switch to window 0-9
```

### Panes (split screen)

```bash
Ctrl-b %       # Split vertical
Ctrl-b "       # Split horizontal
Ctrl-b arrows  # Navigate panes
Ctrl-b x       # Close pane
Ctrl-b z       # Zoom/unzoom pane
```

## My Config

`~/.tmux.conf`:

```bash
# Better prefix (Ctrl-a instead of Ctrl-b)
unbind C-b
set -g prefix C-a

# Mouse support
set -g mouse on

# Start windows at 1, not 0
set -g base-index 1

# Vi mode for copy
setw -g mode-keys vi

# Easier splits
bind | split-window -h
bind - split-window -v

# Reload config
bind r source-file ~/.tmux.conf \; display "Reloaded!"
```

## Plugins (TPM)

Install TPM:

```bash
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
```

Add to `~/.tmux.conf`:

```bash
# Plugins
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-sensible'

# Theme
set -g @plugin 'catppuccin/tmux'

# Session save/restore
set -g @plugin 'tmux-plugins/tmux-resurrect'

# Init TPM (keep at bottom)
run '~/.tmux/plugins/tpm/tpm'
```

Install plugins: `Ctrl-b I`

## Session Resurrection

With tmux-resurrect:

```bash
Ctrl-b Ctrl-s  # Save session
Ctrl-b Ctrl-r  # Restore session
```

Survives reboots!

## Copy Mode

```bash
Ctrl-b [       # Enter copy mode
# Use vi keys to navigate
Space          # Start selection
Enter          # Copy
Ctrl-b ]       # Paste
```

## My Workflow

**Project session:**

```bash
tmux new -s project

# Window 0: Editor
nvim

# Window 1: Server
Ctrl-b c
npm run dev

# Window 2: Tests
Ctrl-b c
npm test --watch

# Window 3: Git
Ctrl-b c
git status
```

Detach at end of day, reattach tomorrow. Everything exactly as I left it.

## Quick Tips

**Rename session:**
```bash
tmux rename-session -t old-name new-name
```

**Share session (pair programming):**
```bash
# Both users attach to same session
tmux attach -t pair
```

**Kill all sessions:**
```bash
tmux kill-server
```

**Nested tmux:**
If running tmux inside tmux, press prefix twice:
```bash
Ctrl-b Ctrl-b c  # New window in inner tmux
```

## Resources

- [Cheat sheet](https://tmuxcheatsheet.com/)
- [Catppuccin theme](https://github.com/catppuccin/tmux)
- [Programming fonts](https://www.programmingfonts.org/#jetbrainsmono)

That's it. Start with basic sessions and splits. Add more as you need it.

Most useful: SSH + tmux = never lose work due to connection drops.
