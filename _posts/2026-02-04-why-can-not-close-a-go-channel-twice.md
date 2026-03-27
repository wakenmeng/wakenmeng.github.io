---
title: Why Cannot Close a Go Channel Twice
date: 2026-02-04
Author: Waken
tags: [go]
toc: true
---

Every time someone asks why closing a channel twice panics, the real
question is usually about ownership.

<!-- more -->

A channel has exactly two states: open and closed. `close(ch)` flips a
bit in the runtime and wakes up any blocked receivers. That's it.
There's no "half-closed" state, no reference counting, no reopening.

After it's closed:

-   receives keep working (draining buffer, then zero value)
-   sends panic
-   closing again panics

The important part is that closing mutates shared state. It's not a
local operation. If two goroutines both think they're responsible for
closing the same channel, one of them is wrong.

Go chooses to panic instead of silently ignoring the second call. That's
intentional. If double-close were ignored, ownership bugs would quietly
survive until something worse happened.

------------------------------------------------------------------------

## Why There's No isClosed

You'll sometimes see people wishing for:

``` go
if !isClosed(ch) {
    close(ch)
}
```

Even if such a function existed, it wouldn't solve anything. Between the
check and the close, another goroutine could close it. The race is still
there.

The only real solution is serialization.

------------------------------------------------------------------------

## How to Correctly Check if a Channel isClosed

Closing a channel will sent a `_, false` message to it, and you get this message every time you read a closed channel. And we can use this to check if a channel is closed. Pay attention on the channel type, it's receive-only, means we shouldn't check and close a channel like mentioned above. 

```go
func isClosed(ch <- chan struct{}) bool {
	select {
	case <- ch:
		return true
	default:
		return false
	}
}
```

------------------------------------------------------------------------

## Single Producer: Easy Case

If one goroutine produces values, it owns the close. That's the clean
model:

``` go
func producer(ch chan<- int) {
    defer close(ch)
    for i := 0; i < 10; i++ {
        ch <- i
    }
}
```

Receivers just range:

``` go
for v := range ch {
    fmt.Println(v)
}
```

Nothing controversial here.

------------------------------------------------------------------------

## Multiple Producers: Where It Breaks

This is wrong:

``` go
go func() {
    ch <- 1
    close(ch)
}()

go func() {
    ch <- 2
    close(ch) // panic, eventually
}()
```

Neither goroutine has exclusive ownership of the lifecycle.

The usual pattern is to separate production from closure:

``` go
var wg sync.WaitGroup
ch := make(chan int)

producer := func(id int) {
    defer wg.Done()
    ch <- id
}

wg.Add(2)
go producer(1)
go producer(2)

go func() {
    wg.Wait()
    close(ch)
}()
```

The close happens exactly once, after all producers finish. Ownership is
explicit.

------------------------------------------------------------------------

## "Just Ignore the Second Close"

You can't.

Not safely.

If you need "at most once" semantics, you serialize the mutation:

``` go
var once sync.Once

func safeClose(ch chan struct{}) {
    once.Do(func() {
        close(ch)
    })
}
```

Now the close has a clear synchronization boundary.

------------------------------------------------------------------------

## Data Completion vs Cancellation

Closing a data channel signals "no more values." Cancellation is
different.

For cancellation, use context:

``` go
select {
case ch <- value:
case <-ctx.Done():
    return
}
```

`ctx.Done()` is a channel that the runtime guarantees will be closed
once. It avoids the ownership ambiguity entirely.

------------------------------------------------------------------------

The panic on double close isn't about being strict. It's about forcing
you to define who owns the channel's lifecycle.

If more than one goroutine might close it, the design isn't finished.

