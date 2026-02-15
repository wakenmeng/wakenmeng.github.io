---
layout: post
title: "Bluetooth on macOS: AAC vs SBC Codec Stability"
date: 2025-02-13
author: Waken
tags: [bluetooth, macos, audio, codecs]
comments: true
---

Had Bluetooth audio dropouts with my Sony speaker. Switching from AAC to SBC fixed it.

<!-- more -->

## The Problem

Sony Bravia Theater U speaker kept stuttering and dropping connection on macOS. Audio would cut out every few minutes.

## AAC vs SBC

**AAC (Advanced Audio Codec):**
- Optimized for Apple devices
- Lower latency (better for video/calls)
- Better sound quality at same bitrate (256-320 kbps)
- **Usually more stable on macOS**

**SBC (Subband Coding):**
- Universal Bluetooth codec
- Not optimized for macOS
- Higher latency
- **But sometimes more reliable with problematic devices**

## When AAC Fails

Despite being "better," AAC can have issues:
- Wi-Fi interference (2.4 GHz overlap)
- USB 3.0 devices nearby (yes, really)
- Some speaker firmware doesn't implement AAC well

In these cases, SBC can be more stable.

## How to Switch

For Sony devices, use the **Sony Connect** app:

1. Download Sony Connect (App Store or Sony website)
2. Connect your speaker
3. Go to device settings
4. Change codec to **SBC**

## Results

After switching to SBC:
- No more dropouts
- Slightly lower quality, but acceptable
- Stable connection

## General Rule

**On macOS:**
- AAC: Better quality, usually more stable
- SBC: Fall back if AAC has problems

**On Windows/Android:**
- SBC often more reliable (AAC implementations vary)

Sometimes "worse" codec wins on stability.
