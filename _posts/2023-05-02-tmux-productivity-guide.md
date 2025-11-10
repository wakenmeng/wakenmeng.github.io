---
layout: post
title: "Tmux: Terminal Multiplexing for Productivity"
date: 2023-05-02
author: Waken
tags: [tmux, terminal, productivity, devtools, linux]
comments: true
toc: true
---

Tmux (terminal multiplexer) is one of those tools that, once you learn it, fundamentally changes how you work in the terminal. It allows you to create multiple terminal sessions within a single window, detach from sessions without losing your work, and maintain persistent environments on remote servers. Let's explore how to use tmux effectively.

<!-- more -->

## Why Tmux?

Before diving into the how, let's understand the why:

1. **Session Persistence**: Detach from sessions and reattach later—your programs keep running
2. **Window Management**: Multiple windows and panes in a single terminal
3. **Remote Work**: SSH into a server, start tmux, disconnect safely knowing work continues
4. **Pair Programming**: Share a tmux session with colleagues for collaborative debugging
5. **Workflow Organization**: Separate projects into different sessions

### Real-World Scenario

```bash
# SSH into server
ssh user@server.com

# Start tmux session
tmux new -s deployment

# Run long deployment
./deploy.sh

# Oops, connection dropped!
# Later, reconnect:
ssh user@server.com
tmux attach -t deployment

# Your deployment is still running! ✅
```

## Installation

```bash
# macOS
brew install tmux

# Ubuntu/Debian
sudo apt install tmux

# Fedora/RHEL
sudo dnf install tmux

# Arch Linux
sudo pacman -S tmux

# Verify installation
tmux -V
```

## Essential Concepts

### Sessions, Windows, and Panes

```
Session: deployment
├── Window 0: logs
│   ├── Pane 0: tail -f app.log
│   └── Pane 1: tail -f error.log
├── Window 1: code
│   └── Pane 0: vim main.go
└── Window 2: tests
    ├── Pane 0: go test -v
    └── Pane 1: watch -n 1 go test
```

### The Prefix Key

By default, tmux uses `Ctrl-b` as the prefix key. All tmux commands start with this prefix.

**Important**: Many users remap this to `Ctrl-a` (more ergonomic):

```bash
# In ~/.tmux.conf
unbind C-b
set -g prefix C-a
bind C-a send-prefix
```

## Quick Start Cheat Sheet

### Session Management

```bash
# Create new session
tmux new -s mysession

# Create named session
tmux new -s dev

# List sessions
tmux ls

# Attach to session
tmux attach -t mysession
tmux a -t mysession  # shorthand

# Attach to last session
tmux attach

# Kill session
tmux kill-session -t mysession

# Detach from current session
Ctrl-b d

# Switch between sessions (inside tmux)
Ctrl-b s  # Show session list
Ctrl-b (  # Previous session
Ctrl-b )  # Next session
```

### Window Management

```bash
# Inside tmux session

# Create new window
Ctrl-b c

# Rename current window
Ctrl-b ,

# Switch windows
Ctrl-b 0-9     # Switch to window 0-9
Ctrl-b n       # Next window
Ctrl-b p       # Previous window
Ctrl-b l       # Last window
Ctrl-b w       # List windows

# Kill current window
Ctrl-b &
```

### Pane Management

```bash
# Split panes
Ctrl-b %       # Split vertically (left/right)
Ctrl-b "       # Split horizontally (top/bottom)

# Navigate panes
Ctrl-b ←↑→↓    # Move to pane
Ctrl-b o       # Cycle through panes
Ctrl-b ;       # Toggle last pane
Ctrl-b q       # Show pane numbers

# Resize panes
Ctrl-b Ctrl-←↑→↓  # Resize current pane

# Close pane
Ctrl-b x       # Kill current pane
exit           # Exit current shell (closes pane)

# Zoom pane (fullscreen toggle)
Ctrl-b z

# Convert pane to window
Ctrl-b !
```

### Copy Mode (Vim-style)

```bash
# Enter copy mode
Ctrl-b [

# Navigate (vim keys)
h, j, k, l     # Move cursor
Ctrl-u/d       # Page up/down
g, G           # Top/bottom
/              # Search forward
?              # Search backward

# Select text
Space          # Start selection
Enter          # Copy selection

# Paste
Ctrl-b ]

# Exit copy mode
q or Esc
```

## Configuration

Create or edit `~/.tmux.conf`:

### Basic Configuration

```bash
# ~/.tmux.conf

# Remap prefix from 'C-b' to 'C-a'
unbind C-b
set -g prefix C-a
bind C-a send-prefix

# Enable mouse support
set -g mouse on

# Start windows and panes at 1, not 0
set -g base-index 1
setw -g pane-base-index 1

# Renumber windows when one is closed
set -g renumber-windows on

# Enable 256 colors
set -g default-terminal "screen-256color"

# Increase scrollback buffer size
set -g history-limit 10000

# Enable vi mode for copy mode
setw -g mode-keys vi

# Faster command sequences
set -s escape-time 0

# Increase repeat timeout
set -g repeat-time 600

# Set terminal title
set -g set-titles on
set -g set-titles-string '#S:#I.#P #W'

# Activity monitoring
setw -g monitor-activity on
set -g visual-activity off

# Auto rename windows
setw -g automatic-rename on
```

### Better Key Bindings

```bash
# Split panes using | and -
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"
unbind '"'
unbind %

# Switch panes using Alt-arrow without prefix
bind -n M-Left select-pane -L
bind -n M-Right select-pane -R
bind -n M-Up select-pane -U
bind -n M-Down select-pane -D

# Resize panes
bind -r H resize-pane -L 5
bind -r J resize-pane -D 5
bind -r K resize-pane -U 5
bind -r L resize-pane -R 5

# Vim-style copy mode
bind -T copy-mode-vi v send -X begin-selection
bind -T copy-mode-vi y send -X copy-selection-and-cancel
bind -T copy-mode-vi C-v send -X rectangle-toggle

# Reload config
bind r source-file ~/.tmux.conf \; display "Config reloaded!"
```

### Status Bar Customization

```bash
# Status bar
set -g status-position bottom
set -g status-justify left
set -g status-style 'bg=#1e1e2e fg=#cdd6f4'

# Left status
set -g status-left-length 50
set -g status-left '#[fg=#89b4fa,bg=#1e1e2e,bold] #S '

# Right status
set -g status-right-length 100
set -g status-right '#[fg=#cdd6f4,bg=#1e1e2e] %Y-%m-%d #[fg=#89b4fa,bg=#1e1e2e,bold] %H:%M '

# Window status
setw -g window-status-format '#[fg=#6c7086,bg=#1e1e2e] #I:#W '
setw -g window-status-current-format '#[fg=#1e1e2e,bg=#89b4fa,bold] #I:#W '

# Pane border
set -g pane-border-style 'fg=#6c7086'
set -g pane-active-border-style 'fg=#89b4fa'

# Message style
set -g message-style 'fg=#1e1e2e bg=#89b4fa bold'
```

## Plugin Management with TPM

[Tmux Plugin Manager](https://github.com/tmux-plugins/tpm) makes it easy to install and manage plugins.

### Install TPM

```bash
git clone https://github.com/tmux-plugins/tpm ~/.tmux/plugins/tpm
```

### Configure Plugins

Add to `~/.tmux.conf`:

```bash
# List of plugins
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-sensible'

# Theme
set -g @plugin 'catppuccin/tmux'
set -g @catppuccin_flavour 'mocha'

# Session management
set -g @plugin 'tmux-plugins/tmux-resurrect'
set -g @plugin 'tmux-plugins/tmux-continuum'

# Clipboard support
set -g @plugin 'tmux-plugins/tmux-yank'

# Pane navigation
set -g @plugin 'christoomey/vim-tmux-navigator'

# Initialize TPM (keep this at the bottom)
run '~/.tmux/plugins/tpm/tpm'
```

### Plugin Key Bindings

```bash
# Install plugins
Ctrl-b I

# Update plugins
Ctrl-b U

# Uninstall plugins (remove from config first)
Ctrl-b Alt-u
```

## Recommended Plugins

### 1. Tmux Resurrect (Save/Restore Sessions)

```bash
set -g @plugin 'tmux-plugins/tmux-resurrect'

# Save session
Ctrl-b Ctrl-s

# Restore session
Ctrl-b Ctrl-r

# Restore vim sessions
set -g @resurrect-strategy-vim 'session'

# Restore neovim sessions
set -g @resurrect-strategy-nvim 'session'

# Restore pane contents
set -g @resurrect-capture-pane-contents 'on'
```

### 2. Tmux Continuum (Auto-save Sessions)

```bash
set -g @plugin 'tmux-plugins/tmux-continuum'

# Auto-save interval (minutes)
set -g @continuum-save-interval '15'

# Auto-restore on tmux start
set -g @continuum-restore 'on'

# Show continuum status in status bar
set -g status-right 'Continuum: #{continuum_status}'
```

### 3. Catppuccin Theme

```bash
set -g @plugin 'catppuccin/tmux'
set -g @catppuccin_flavour 'mocha'  # latte, frappe, macchiato, mocha

# Window format
set -g @catppuccin_window_default_text "#W"
set -g @catppuccin_window_current_text "#W"

# Status modules
set -g @catppuccin_status_modules_right "session date_time"
```

### 4. Vim-Tmux Navigator

Seamless navigation between vim and tmux panes:

```bash
set -g @plugin 'christoomey/vim-tmux-navigator'

# In vim, also install the vim plugin:
# Plug 'christoomey/vim-tmux-navigator'

# Then use:
# Ctrl-h/j/k/l to navigate between vim and tmux panes
```

## Advanced Workflows

### Project-Based Sessions

Create a script to set up project environments:

```bash
#!/bin/bash
# ~/scripts/dev-session.sh

SESSION="myproject"

# Create session
tmux new-session -d -s $SESSION -n editor

# Window 1: Editor
tmux send-keys -t $SESSION:editor "cd ~/projects/myproject && nvim" C-m

# Window 2: Server
tmux new-window -t $SESSION -n server
tmux send-keys -t $SESSION:server "cd ~/projects/myproject && npm run dev" C-m

# Window 3: Tests
tmux new-window -t $SESSION -n tests
tmux send-keys -t $SESSION:tests "cd ~/projects/myproject" C-m

# Window 4: Logs (split horizontally)
tmux new-window -t $SESSION -n logs
tmux send-keys -t $SESSION:logs "cd ~/projects/myproject && tail -f logs/app.log" C-m
tmux split-window -h -t $SESSION:logs
tmux send-keys -t $SESSION:logs "cd ~/projects/myproject && tail -f logs/error.log" C-m

# Attach to session
tmux attach -t $SESSION:editor
```

Usage:

```bash
chmod +x ~/scripts/dev-session.sh
~/scripts/dev-session.sh
```

### Pair Programming

```bash
# On host machine
tmux new -s pair

# Share session with colleague (same user)
# Colleague connects:
tmux attach -t pair

# For different users, use tmux-pair or wemux
```

### Multi-Monitor Setup

```bash
# Create session with multiple windows
tmux new -s monitors

# In window 0: System monitoring
htop

# Window 1: Logs
Ctrl-b c
journalctl -f

# Window 2: Network
Ctrl-b c
iftop

# Detach and view from different terminal
Ctrl-b d

# In another terminal, link windows
tmux new -s view -t monitors
```

## Tips and Tricks

### Command Mode

```bash
# Enter command mode
Ctrl-b :

# Useful commands:
:setw synchronize-panes on   # Type in all panes at once
:setw synchronize-panes off  # Disable sync
:resize-pane -D 10            # Resize down 10 lines
:swap-window -s 2 -t 1        # Swap windows
```

### Scripting Tmux

```bash
# Send commands to specific pane
tmux send-keys -t mysession:0.0 "ls -la" C-m

# Create window with command
tmux new-window -t mysession -n build "make && ./app"

# List all panes
tmux list-panes -a -F "#{session_name}:#{window_index}.#{pane_index}"
```

### Copy to System Clipboard

```bash
# macOS
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "pbcopy"

# Linux (with xclip)
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "xclip -selection clipboard"

# Linux (with xsel)
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "xsel -b"
```

### Custom Functions

Add to `~/.tmux.conf`:

```bash
# Quick window switching
bind -r Space next-window
bind -r BSpace previous-window

# Quick session switching
bind S choose-session

# Toggle status bar
bind t set status

# Clear history and screen
bind C-k send-keys -R \; clear-history
```

## Troubleshooting

### Colors Look Wrong

```bash
# In ~/.bashrc or ~/.zshrc
export TERM=xterm-256color

# In ~/.tmux.conf
set -g default-terminal "screen-256color"
set -ga terminal-overrides ",xterm-256color:Tc"
```

### Slow Escape Key in Vim

```bash
# In ~/.tmux.conf
set -s escape-time 0
```

### Mouse Scrolling Issues

```bash
# In ~/.tmux.conf
set -g mouse on
set -g terminal-overrides 'xterm*:smcup@:rmcup@'
```

## Recommended Fonts

For the best tmux experience with themes:

- **JetBrains Mono** - https://www.jetbrains.com/lp/mono/
- **Fira Code** - https://github.com/tonsky/FiraCode
- **Hack** - https://sourcefoundry.org/hack/
- **Cascadia Code** - https://github.com/microsoft/cascadia-code

More options: https://www.programmingfonts.org/#jetbrainsmono

## Resources

- **Cheat Sheet**: https://tmuxcheatsheet.com/
- **Best Practices**: https://github.com/datamade/how-to/blob/main/shell/tmux-best-practices.md
- **Tutorial**: https://www.ocf.berkeley.edu/~ckuehl/tmux/
- **Official Docs**: https://github.com/tmux/tmux/wiki
- **Community**: r/tmux on Reddit

## Complete Configuration Example

Here's my full `~/.tmux.conf`:

```bash
# Prefix
unbind C-b
set -g prefix C-a
bind C-a send-prefix

# General
set -g mouse on
set -g base-index 1
setw -g pane-base-index 1
set -g renumber-windows on
set -g default-terminal "screen-256color"
set -ga terminal-overrides ",xterm-256color:Tc"
set -g history-limit 50000
setw -g mode-keys vi
set -s escape-time 0
set -g repeat-time 600
set -g set-titles on
set -g set-titles-string '#S:#I.#P #W'

# Key bindings
bind | split-window -h -c "#{pane_current_path}"
bind - split-window -v -c "#{pane_current_path}"
bind r source-file ~/.tmux.conf \; display "Reloaded!"
bind -T copy-mode-vi v send -X begin-selection
bind -T copy-mode-vi y send -X copy-pipe-and-cancel "pbcopy"

# Plugins
set -g @plugin 'tmux-plugins/tpm'
set -g @plugin 'tmux-plugins/tmux-sensible'
set -g @plugin 'catppuccin/tmux'
set -g @plugin 'tmux-plugins/tmux-resurrect'
set -g @plugin 'tmux-plugins/tmux-continuum'
set -g @plugin 'christoomey/vim-tmux-navigator'

set -g @catppuccin_flavour 'mocha'
set -g @continuum-restore 'on'
set -g @resurrect-capture-pane-contents 'on'

run '~/.tmux/plugins/tpm/tpm'
```

## Conclusion

Tmux is an invaluable tool for:
- Remote server management
- Multi-project workflows
- Pair programming
- Session persistence

Start small with basic splits and windows, then gradually add more advanced features. Your terminal productivity will skyrocket!

Happy multiplexing! 🚀
