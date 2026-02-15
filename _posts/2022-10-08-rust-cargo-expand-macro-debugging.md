---
layout: post
title: "cargo-expand: See What Rust Macros Actually Generate"
date: 2022-10-08
author: Waken
tags: [rust, macros, debugging]
comments: true
---

Debugging Rust macros is painful. Compiler errors point to the macro invocation, not what it generates. `cargo-expand` fixes that.

<!-- more -->

## The Problem

```rust
#[derive(Debug, Serialize)]
struct User {
    id: u32,
    name: String,
}

// Error: trait bound `User: Serialize` is not satisfied
// But what code did #[derive(Serialize)] actually generate?
```

No idea what the macro produced.

## Install cargo-expand

```bash
# Need nightly toolchain (doesn't have to be default)
rustup toolchain install nightly

# Install
cargo install cargo-expand

# Use
cargo expand
```

## Usage

```bash
# Expand everything
cargo expand

# Expand specific item
cargo expand MyStruct

# Expand module
cargo expand my_module
```

## Example: What Does #[derive(Debug)] Generate?

```rust
#[derive(Debug)]
struct Point {
    x: i32,
    y: i32,
}
```

Run `cargo expand`:

```rust
// Expanded code:
impl ::core::fmt::Debug for Point {
    fn fmt(&self, f: &mut ::core::fmt::Formatter) -> ::core::fmt::Result {
        ::core::fmt::Formatter::debug_struct_field2_finish(
            f,
            "Point",
            "x",
            &&self.x,
            "y",
            &&self.y,
        )
    }
}
```

Now I can see exactly what `Debug` does!

## Useful for Understanding

**1. Derive Macros**

See how `#[derive(Clone, Serialize, etc)]` are implemented.

**2. Procedural Macros**

```rust
#[tokio::main]
async fn main() {
    // ...
}

// Expands to:
fn main() {
    tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .unwrap()
        .block_on(async {
            // your code
        })
}
```

Aha! That's what `#[tokio::main]` does.

**3. Declarative Macros**

```rust
macro_rules! vec_of_strings {
    ($($x:expr),*) => {
        vec![$($x.to_string()),*]
    };
}

let v = vec_of_strings!["hello", "world"];
```

`cargo expand` shows the final expanded `vec!` and `.to_string()` calls.

## Pretty Output

```bash
# Colorized
cargo expand | bat -l rust

# Save to file for comparison
cargo expand > before.rs
# make changes
cargo expand > after.rs
diff before.rs after.rs
```

## When I Use It

- Debugging macro errors
- Learning how libraries use macros
- Understanding what `async-trait`, `thiserror`, etc. actually do
- Writing my own macros

## One Gotcha

Needs nightly toolchain installed. Not a big deal:

```bash
rustup toolchain install nightly
# Don't need to set as default
cargo expand  # automatically uses nightly
```

That's it. Super useful when macros behave unexpectedly or you want to understand what they're doing under the hood.
