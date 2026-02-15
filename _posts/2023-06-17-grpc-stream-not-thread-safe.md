---
layout: post
title: "gRPC Streams Are Not Thread-Safe"
date: 2023-06-17
author: Waken
tags: [grpc, golang, concurrency]
comments: true
---

Hit a concurrency bug with gRPC streams. Turns out they're not thread-safe for concurrent writes.

<!-- more -->

## The Issue

Was trying to write to a gRPC stream from multiple goroutines. Got strange errors and corrupted messages.

## The Rule

From the [gRPC discussion](https://groups.google.com/g/grpc-io/c/aI6L6M4fzQ0):

> Stream does not allow multiple concurrent writers. The general rule is that you should NOT assume it is safe to be called concurrently unless the comment explicitly states that.

## What This Means

```go
// DON'T do this
func sendMessages(stream pb.Service_StreamClient) {
    go func() {
        stream.Send(&pb.Message{Data: "from goroutine 1"})
    }()

    go func() {
        stream.Send(&pb.Message{Data: "from goroutine 2"})
    }()
}
```

Multiple concurrent `Send()` calls = undefined behavior.

## The Fix

Serialize writes through a single goroutine:

```go
func sendMessages(stream pb.Service_StreamClient) {
    msgChan := make(chan *pb.Message, 10)

    // Single writer goroutine
    go func() {
        for msg := range msgChan {
            stream.Send(msg)
        }
    }()

    // Other goroutines send to channel
    go func() {
        msgChan <- &pb.Message{Data: "from goroutine 1"}
    }()

    go func() {
        msgChan <- &pb.Message{Data: "from goroutine 2"}
    }()
}
```

Channel serializes access. Only one goroutine actually writes to the stream.

## Lesson

gRPC streams work like most Go stdlib types: **not concurrent by default**. Check `container/list`, `bufio.Writer`, etc. - same pattern.

If docs don't explicitly say "thread-safe", assume it's not.
