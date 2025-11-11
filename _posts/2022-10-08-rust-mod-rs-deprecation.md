---
layout: post
title: "Rust: Stop Using mod.rs"
date: 2022-10-08
author: Waken
tags: [rust, conventions]
comments: true
---

Learned that `mod.rs` files are the old way. Rust 1.30+ has a better convention.

<!-- more -->

## The Old Way (Don't Do This)

```
src/
  util/
    mod.rs        # Old style
    helper.rs
```

## The New Way

```
src/
  util.rs         # Module definition
  util/
    helper.rs     # Submodule
```

In `util.rs`:
```rust
mod helper;  // Load util/helper.rs
```

## Why Change?

From the Rust docs:

> Note: Prior to rustc 1.30, using mod.rs files was the way to load a module with nested children. It is encouraged to use the new naming convention as it is more consistent, and avoids having many files named mod.rs within a project.

Having 20 files all named `mod.rs` was confusing. The new way is clearer.

## Important

**You cannot have both** `util.rs` and `util/mod.rs`. Rust will error. Pick one style and stick to it.

For new projects: use the new style. Cleaner and more consistent.
