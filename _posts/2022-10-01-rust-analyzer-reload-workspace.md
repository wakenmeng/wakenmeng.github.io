---
layout: post
title: "rust-analyzer: Manually Reloading Workspace"
date: 2022-10-01
author: Waken
tags: [rust, emacs, lsp, rust-analyzer]
comments: true
---

rust-analyzer wasn't picking up my Cargo.toml changes. Had to reload manually.

<!-- more -->

## The Problem

Modified `Cargo.toml` to add a new dependency. rust-analyzer didn't notice. LSP features (autocomplete, goto definition) didn't work for the new crate.

## Why It Happens

rust-analyzer doesn't actively watch for file changes ([open issue](https://github.com/helix-editor/helix/issues/2479)).

It loads the workspace once at startup. After that, you're on your own.

## The Solution

In Emacs with LSP mode:

```elisp
M-x lsp-rust-analyzer-reload-workspace
```

rust-analyzer re-reads Cargo.toml and rebuilds its index.

## For Other Editors

**VS Code:**
- Command palette → "rust-analyzer: Reload Workspace"

**Vim/Neovim with coc-rust-analyzer:**
```vim
:CocCommand rust-analyzer.reloadWorkspace
```

**Generic LSP:**
- Restart the LSP server

## When to Reload

Reload after:
- Adding/removing dependencies in Cargo.toml
- Changing features
- Adding new workspace members
- Modifying build.rs

Basically, any change that affects project structure.

## Alternative

Just restart your editor. Slower, but guaranteed to work.

But for quick iterations, manual reload is faster.
