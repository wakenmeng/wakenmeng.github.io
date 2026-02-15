---
layout: post
title: Command Semantics over MQTT
date: 2026-02-08
Author: Waken
tags: [mqtt]
comments: true
toc: true
---
The hardest problem in device communication over MQTT is not delivery. It is semantic consistency under concurrency. Once you scale beyond a single service instance and a single stable device connection, three things become unavoidable: Messages are duplicated; they arrive late; state transitions race.

Most designs fail not because MQTT misbehaves, but because command progression is modeled implicitly instead of explicitly.
The moment you persist commands as rows rather than treating publishes as events, you are forced to confront a more interesting constraint: command progression must be monotonic.

A command cannot move from executed back to acknowledged. It cannot move from failed to sent without a new identity. That sounds trivial, but under concurrent handlers it is easy to violate unless transitions are guarded. In practice, this means:  

1. Every transition is conditional.
2. Every handler must be idempotent.
3. Storage must reject illegal transitions.

For example, an acknowledgement handler should not blindly mark a command as acknowledged. It should perform something equivalent to:

```
UPDATE commands
SET status = 'acknowledged'
WHERE id = ?
  AND status = 'sent';
```

If the update affects zero rows, the event is either duplicated or obsolete. No exception is required. The system simply converges.

This storage-level monotonicity is stronger than any QoS guarantee. QoS 2 prevents duplicate delivery within a session. It does not prevent:

- Backend restarts.
- Device replays.
- Cross-instance race conditions.

Once you scale horizontally, transport guarantees become strictly weaker than database invariants.

Retain semantics reveal a related boundary. Retaining a command transforms an intent into state. That violates monotonic progression because the system can re-enter an earlier lifecycle stage upon reconnect. Intent must therefore be ephemeral, while state topics may be durable.

Another subtle failure mode appears under timeout and retry logic. If retries are triggered based on in-memory timers, they compete with inbound results that may be slightly delayed. Without monotonic state checks, you can transition a command to timed_out and then process a valid executed result afterward. The fix is not more locking. It is to treat timeout as just another conditional transition, applied only if the current state still allows it.

What MQTT exposes is not a protocol problem, but a modeling discipline problem. Distributed systems require that domain progression be:

- Explicit
- Idempotent
- Monotonic
- Enforced at storage

Once those properties hold, duplicates and reordering stop being threats. They become noise. At that point, MQTT is just a pipe.
