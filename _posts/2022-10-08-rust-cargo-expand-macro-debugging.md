---
layout: post
title: "Debugging Rust Macros with cargo-expand"
date: 2022-10-08
author: Waken
tags: [rust, macros, debugging, cargo, devtools]
comments: true
toc: true
---

Rust macros are powerful but can be notoriously difficult to debug. When your macro-heavy code doesn't compile or behaves unexpectedly, wouldn't it be great to see exactly what code the macros generate? Enter `cargo-expand`, a tool that reveals the expanded form of Rust macros, making debugging significantly easier.

<!-- more -->

## The Problem with Macro Debugging

Rust macros operate at compile time, transforming code before the compiler sees it. When something goes wrong, error messages point to the macro invocation, not the generated code:

```rust
#[derive(Debug, Clone, Serialize, Deserialize)]
struct User {
    id: u32,
    name: String,
}

// Compiler error:
// error[E0277]: the trait bound `User: Serialize` is not satisfied
// But what code did #[derive(Serialize)] actually generate?
```

Common pain points:
- **Cryptic error messages** pointing to macro calls
- **Hidden implementations** from derive macros
- **Unexpected behavior** from procedural macros
- **Difficult debugging** of macro expansion logic

## Solution: cargo-expand

`cargo-expand` is a cargo subcommand that shows you the output of macro expansion and `#[derive]` expansion.

### Installation

```bash
# Requires nightly toolchain (doesn't need to be default)
rustup toolchain install nightly

# Install cargo-expand
cargo install cargo-expand

# Verify installation
cargo expand --version
```

**Important:** You need the nightly toolchain installed, but you don't need to set it as your default.

## Basic Usage

### Expand an Entire Module

```bash
# Expand main.rs or lib.rs
cargo expand

# Expand a specific module
cargo expand module_name

# Expand a submodule
cargo expand module::submodule
```

### Expand a Specific Item

```bash
# Expand a specific function
cargo expand function_name

# Expand a specific struct
cargo expand StructName

# Expand within a module
cargo expand module::StructName
```

## Real-World Examples

### Example 1: Derive Macro Expansion

Let's see what `#[derive(Debug)]` actually generates:

```rust
// src/main.rs
#[derive(Debug)]
struct Point {
    x: i32,
    y: i32,
}

fn main() {
    let p = Point { x: 10, y: 20 };
    println!("{:?}", p);
}
```

Run expansion:

```bash
cargo expand
```

Output (simplified):

```rust
struct Point {
    x: i32,
    y: i32,
}

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

fn main() {
    let p = Point { x: 10, y: 20 };
    {
        ::std::io::_print(
            ::core::fmt::Arguments::new_v1(
                &["", "\n"],
                &[::core::fmt::ArgumentV1::new_debug(&&p)],
            ),
        );
    };
}
```

Now you can see exactly how `Debug` is implemented!

### Example 2: Serde Derive Macros

```rust
use serde::{Deserialize, Serialize};

#[derive(Debug, Clone, Serialize, Deserialize)]
struct User {
    id: u32,
    #[serde(rename = "username")]
    name: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    email: Option<String>,
}
```

Expand to see the generated serialization code:

```bash
cargo expand User
```

This reveals:
- How `Serialize` trait is implemented
- How `#[serde(rename)]` affects field naming
- How `#[serde(skip_serializing_if)]` adds conditional logic

### Example 3: Procedural Macros

```rust
use tokio::main;

#[tokio::main]
async fn main() {
    println!("Hello, async world!");
}
```

See what the `#[tokio::main]` macro actually does:

```bash
cargo expand
```

Output (simplified):

```rust
fn main() {
    tokio::runtime::Builder::new_multi_thread()
        .enable_all()
        .build()
        .unwrap()
        .block_on(async {
            {
                ::std::io::_print(
                    ::core::fmt::Arguments::new_v1(
                        &["Hello, async world!\n"],
                        &[],
                    ),
                );
            };
        })
}
```

Aha! It wraps your async function in a runtime builder.

### Example 4: Custom Declarative Macros

```rust
macro_rules! vec_of_strings {
    ($($x:expr),*) => {
        vec![$($x.to_string()),*]
    };
}

fn main() {
    let v = vec_of_strings!["hello", "world", "foo"];
    println!("{:?}", v);
}
```

```bash
cargo expand main
```

Shows the expanded `vec!` and `to_string()` calls:

```rust
fn main() {
    let v = <[_]>::into_vec(
        #[rustc_box]
        ::alloc::boxed::Box::new([
            "hello".to_string(),
            "world".to_string(),
            "foo".to_string(),
        ]),
    );
    {
        ::std::io::_print(
            ::core::fmt::Arguments::new_v1(
                &["", "\n"],
                &[::core::fmt::ArgumentV1::new_debug(&&v)],
            ),
        );
    };
}
```

## Advanced Usage

### Colorized Output

```bash
# Pretty print with colors
cargo expand --color always | less -R

# Or use bat for syntax highlighting
cargo expand | bat -l rust
```

### Expand with Features

```bash
# Expand with specific features enabled
cargo expand --features async,serde

# Expand for different target
cargo expand --target x86_64-unknown-linux-gnu
```

### Expand Tests

```bash
# Expand test functions
cargo expand --tests test_name

# Expand all tests
cargo expand --tests
```

### Expand Examples

```bash
# Expand an example
cargo expand --example example_name
```

### Specify Nightly Version

```bash
# Use specific nightly version
cargo +nightly-2024-01-01 expand
```

## Debugging Workflow

### Step 1: Identify the Problem

```rust
#[derive(Debug, Clone, Serialize)]
struct Config {
    #[serde(flatten)]
    database: DatabaseConfig,
    server: ServerConfig,
}

// Error: the trait `Serialize` is not satisfied for `DatabaseConfig`
```

### Step 2: Expand to See Generated Code

```bash
cargo expand Config
```

### Step 3: Analyze the Expansion

Look for:
- Missing trait implementations
- Incorrect type transformations
- Unexpected code generation
- Lifetime or generic parameter issues

### Step 4: Fix the Root Cause

```rust
// Fix: Ensure DatabaseConfig also derives Serialize
#[derive(Debug, Clone, Serialize)]
struct DatabaseConfig {
    host: String,
    port: u16,
}
```

## Comparing Macro Versions

```bash
# Expand and save to file
cargo expand > before.rs

# Modify macro or dependencies
# ...

# Expand again
cargo expand > after.rs

# Compare
diff -u before.rs after.rs
```

Or use a better diff tool:

```bash
# Side-by-side comparison
diff -y before.rs after.rs | less

# With syntax highlighting
delta before.rs after.rs
```

## IDE Integration

### Visual Studio Code

Add to tasks.json:

```json
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "cargo expand",
            "type": "shell",
            "command": "cargo",
            "args": ["expand"],
            "group": "build",
            "presentation": {
                "reveal": "always",
                "panel": "new"
            }
        }
    ]
}
```

### Vim/Neovim

```vim
" Add to .vimrc or init.vim
command! CargoExpand :term cargo expand

" Or with floating window (Neovim)
nnoremap <leader>ce :split \| term cargo expand<CR>
```

### Emacs

```elisp
;; Add to .emacs or init.el
(defun rust-cargo-expand ()
  "Run cargo expand and show output in a new buffer."
  (interactive)
  (let ((buffer (get-buffer-create "*cargo-expand*")))
    (with-current-buffer buffer
      (erase-buffer)
      (shell-command "cargo expand" buffer)
      (rust-mode))
    (display-buffer buffer)))

(define-key rust-mode-map (kbd "C-c C-e") 'rust-cargo-expand)
```

## Understanding Hygiene

Macro hygiene prevents name collisions. `cargo-expand` helps visualize this:

```rust
macro_rules! log_value {
    ($val:expr) => {
        let value = $val;
        println!("Value: {}", value);
    };
}

fn main() {
    let value = 10;
    log_value!(value + 5);
}
```

Expand to see how Rust maintains hygiene:

```bash
cargo expand main
```

You'll see Rust generates unique identifiers to prevent the `value` variable in the macro from conflicting with the outer `value`.

## Troubleshooting

### cargo-expand Fails

```bash
# Make sure you have nightly installed
rustup toolchain install nightly

# Update cargo-expand
cargo install cargo-expand --force

# If still fails, try with explicit nightly
cargo +nightly expand
```

### Incomplete Expansion

Sometimes `cargo-expand` doesn't show everything. Try:

```bash
# Show all features
cargo expand --all-features

# Verbose output
cargo expand --verbose

# Check for errors
cargo expand 2>&1 | less
```

### Expansion Too Large

```bash
# Expand only specific items
cargo expand module::function_name

# Pipe to pager
cargo expand | less

# Save to file
cargo expand > expansion.rs
```

## Alternative Tools

### rustfmt on Expanded Code

```bash
# Make expansion more readable
cargo expand | rustfmt
```

### cargo-expand vs rustc -Z

```bash
# Using rustc directly (also requires nightly)
rustc -Z unpretty=expanded src/main.rs

# cargo-expand is more user-friendly
cargo expand
```

## Best Practices

1. **Use Specific Targets**
   ```bash
   cargo expand module::Item  # Not just cargo expand
   ```

2. **Save Expansions for Reference**
   ```bash
   cargo expand > docs/macro_expansions.rs
   ```

3. **Compare Before/After**
   ```bash
   git diff expansion.rs  # After macro changes
   ```

4. **Document Complex Macros**
   ```rust
   /// Expansion example:
   /// ```
   /// // macro_name!(arg)
   /// // expands to:
   /// // actual_code()
   /// ```
   macro_rules! macro_name { ... }
   ```

5. **Test Macro Hygiene**
   ```bash
   # Verify no name collisions
   cargo expand | grep -A5 "let value"
   ```

## Real-World Use Cases

### 1. Understanding async-trait

```rust
use async_trait::async_trait;

#[async_trait]
trait MyTrait {
    async fn my_method(&self) -> Result<(), Error>;
}
```

`cargo expand` reveals how `async-trait` transforms async trait methods into boxed futures.

### 2. Debugging thiserror

```rust
use thiserror::Error;

#[derive(Error, Debug)]
pub enum MyError {
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),

    #[error("Parse error at line {line}")]
    Parse { line: usize },
}
```

See exactly how error Display implementations are generated.

### 3. sqlx Compile-time Verification

```rust
let user = sqlx::query!("SELECT id, name FROM users WHERE id = ?", user_id)
    .fetch_one(&pool)
    .await?;
```

Expand to see the generated struct and verification code.

## Conclusion

`cargo-expand` is an essential tool for:
- ✅ Understanding macro-generated code
- ✅ Debugging derive macro issues
- ✅ Learning how libraries use macros
- ✅ Verifying macro hygiene
- ✅ Developing your own macros

**Quick Reference:**

```bash
# Installation
cargo install cargo-expand

# Basic usage
cargo expand                    # Expand everything
cargo expand module::Item       # Expand specific item
cargo expand | bat -l rust      # Pretty output

# Advanced
cargo expand --tests            # Expand tests
cargo expand --features feat    # With features
cargo expand --color always     # Colored output
```

Next time you're puzzled by a macro error, remember: `cargo expand` is your friend! 🦀

## Resources

- [cargo-expand GitHub](https://github.com/dtolnay/cargo-expand)
- [The Little Book of Rust Macros](https://veykril.github.io/tlborm/)
- [Rust Macro Reference](https://doc.rust-lang.org/reference/macros.html)
- [Procedural Macros Workshop](https://github.com/dtolnay/proc-macro-workshop)
