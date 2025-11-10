---
layout: post
title: "Rust Traits: What I Wish I Knew Earlier"
date: 2023-04-10
author: Waken
tags: [rust, traits]
comments: true
---

Spent the day figuring out Rust traits. Here's the mental model that finally clicked for me.

<!-- more -->

## The Core Idea

Traits define what a type can do, not what it is.

```rust
trait Printable {
    fn print(&self);
}

struct MyStruct {
    value: i32,
}

impl Printable for MyStruct {
    fn print(&self) {
        println!("Value is: {}", self.value);
    }
}

fn main() {
    let my_struct = MyStruct { value: 42 };
    my_struct.print(); // Value is: 42
}
```

Without `impl Printable for MyStruct`, you can't call `print()` on it. The trait implementation "unlocks" those methods.

## Important: Traits Must Be In Scope

This tripped me up:

```rust
use animals::{Dog, Speak}; // Need BOTH!

let dog = Dog;
dog.speak(); // Works

// Without importing Speak trait:
// ERROR: no method named `speak` found
```

Even if a type implements a trait, you need to import the trait to use its methods.

## Deriving Common Traits

```rust
#[derive(Debug, Clone, PartialEq)]
struct User {
    id: u32,
    name: String,
}

// Now you get:
let u1 = User { id: 1, name: "Alice".into() };
let u2 = u1.clone();
println!("{:?}", u1);  // Debug output
assert_eq!(u1, u2);    // Equality check
```

This auto-implements Debug, Clone, and PartialEq. Super convenient.

## Trait Bounds

```rust
// Long form
fn print_twice<T: Printable>(item: T) {
    item.print();
    item.print();
}

// Shorter form
fn quick_print(item: impl Printable) {
    item.print();
}

// Multiple bounds with where clause
fn process<T>(item: T)
where
    T: Printable + Clone,
{
    item.print();
    let copy = item.clone();
    copy.print();
}
```

## Trait Objects for Heterogeneous Collections

When you need different types in a collection:

```rust
trait Animal {
    fn make_sound(&self) -> String;
}

struct Dog;
struct Cat;

impl Animal for Dog {
    fn make_sound(&self) -> String { "Woof!".into() }
}

impl Animal for Cat {
    fn make_sound(&self) -> String { "Meow!".into() }
}

// Mix different types
let animals: Vec<Box<dyn Animal>> = vec![
    Box::new(Dog),
    Box::new(Cat),
];

for animal in animals {
    println!("{}", animal.make_sound());
}
```

Trade-off: Static dispatch (generics) is faster, dynamic dispatch (trait objects) is more flexible.

## Default Implementations

Traits can provide default methods:

```rust
trait Greeter {
    fn name(&self) -> &str;

    // Default implementation
    fn greet(&self) {
        println!("Hello, {}!", self.name());
    }
}

struct Person { name: String }

impl Greeter for Person {
    fn name(&self) -> &str {
        &self.name
    }
    // greet() comes for free
}
```

## Associated Types

Still wrapping my head around these:

```rust
trait Container {
    type Item; // Associated type

    fn add(&mut self, item: Self::Item);
    fn get(&self, index: usize) -> Option<&Self::Item>;
}

impl Container for Vec<i32> {
    type Item = i32;

    fn add(&mut self, item: i32) {
        self.push(item);
    }

    fn get(&self, index: usize) -> Option<&i32> {
        self.get(index)
    }
}
```

Different from generics. Associated types = one concrete type per implementation.

## Orphan Rule Gotcha

Can't implement external trait on external type:

```rust
// Can't do this:
// impl Display for Vec<i32> {} // ERROR!

// Workaround: newtype pattern
use std::fmt;

struct MyVec(Vec<i32>);

impl fmt::Display for MyVec {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{:?}", self.0)
    }
}
```

## Debugging Tip

When trait errors get cryptic, set `RUST_BACKTRACE=1`:

```bash
RUST_BACKTRACE=1 cargo run
```

Better error traces help a lot.

## What I Use Most

1. **Debug** - `{:?}` for println debugging
2. **Clone** - when I need copies
3. **Default** - `T::default()` for constructors
4. **From/Into** - type conversions
5. **Display** - `{}` for user-facing output

Still getting used to when to use trait bounds vs trait objects. General rule: use generics (static dispatch) unless you need heterogeneous collections or plugin systems.

Traits are powerful once the mental model clicks. They're like interfaces but more flexible.
