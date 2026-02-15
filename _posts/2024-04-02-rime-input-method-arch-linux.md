---
layout: post
title: "Setting Up Rime Input Method on Arch Linux"
date: 2024-04-02
author: Waken
tags: [linux, rime, arch-linux, chinese-input]
comments: true
---

Switched to Rime for Chinese input on Arch Linux. Here's my setup with the rime-ice dictionary.

<!-- more -->

## Install Rime

Using iBus framework:

```bash
sudo pacman -S ibus-rime
```

## Dictionary: rime-ice

Instead of default dictionary, I use [rime-ice](https://github.com/iDvel/rime-ice) - better Chinese input with modern vocabulary.

On Arch:
```bash
# Follow instructions from:
# https://github.com/iDvel/rime-ice?tab=readme-ov-file#arch-linux

# Install via AUR or manual setup
```

## Configuration

Config location: `$HOME/.config/ibus/rime/`

Main config file: `default.custom.yaml`

```yaml
patch:
  schema_list:
    - schema: rime_ice
```

## Package Management (Optional)

Rime has [plum](https://github.com/rime/plum) for package management:

```bash
cd ~/playground/rime/plum
bash rime-install <package-name>
```

Useful for installing additional schemas or dictionaries.

## Deploy

After config changes:

1. Right-click iBus tray icon
2. Select "Deploy"
3. Wait for rebuild

Or restart iBus entirely.

## Why Rime?

- Open source
- Highly customizable
- Works offline
- No privacy concerns
- Active community with good dictionaries

rime-ice specifically has excellent modern Chinese vocabulary that other input methods lack.
