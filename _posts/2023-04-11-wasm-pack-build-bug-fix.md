---
layout: post
title: "wasm-pack Build Bug: Use Version 0.9.1"
date: 2023-04-11
author: Waken
tags: [rust, wasm, wasm-pack]
comments: true
---

Hit a build error with wasm-pack. Solution: downgrade to 0.9.1.

<!-- more -->

## The Problem

Running `wasm-pack build` failed with cryptic errors. Something about missing files or build artifacts.

## The Cause

Known bug in newer wasm-pack versions: [GitHub issue #823](https://github.com/rustwasm/wasm-pack/issues/823)

## The Fix

Use wasm-pack 0.9.1:

```bash
cargo install wasm-pack --version 0.9.1 --force
```

Build works again.

## Checking Your Version

```bash
wasm-pack --version
```

If you have a newer version and builds are failing, downgrade.

## When to Upgrade

Check the GitHub issue. Once it's closed and fixed, upgrade to the latest version.

For now, stick with 0.9.1 for stable builds.

Quick fix for a frustrating bug.
