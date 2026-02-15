---
layout: post
title: "PostgreSQL Locks & Spring JPA Locks"
date: 2026-01-14
author: Waken
tags: [postgresql, database, concurrency, jpa, backend]
comments: true
---

I recently spent some focused time reading PostgreSQL’s concurrency and locking documentation, and then mapping that understanding to how locks actually work in Spring JPA / Hibernate.

This is not a reference manual.  
It’s a **mental model** — the kind that helps avoid race conditions, deadlocks, and subtle production bugs.
(with the help with AI)

<!-- more -->

## Why Locks Exist (Even with MVCC)

MVCC (Multi-Version Concurrency Control) allows concurrent reads and writes by maintaining multiple versions of data, giving each transaction a consistent snapshot of the database. Instead of blocking readers, writers create new row versions while readers continue to see older committed versions. This reduces read–write contention and improves concurrency, at the cost of additional storage and background cleanup (e.g., PostgreSQL’s VACUUM).

### How it works
- Versioning: Each data record has a version number or timestamp.
- Reads: A transaction reads the newest version of the data that existed when the transaction started, ensuring a consistent view.
- Writes: Instead of overwriting, a writer creates a new version of the record.
- Commit: Once the new version is committed, it becomes the latest, while older versions remain for other transactions until no longer needed

### PostgreSQL uses MVCC, which already gives:
- readers don’t block writers
- writers don’t block readers

Locks exist to protect:
- table lifetime
- schema stability
- row ownership for business logic

Locks answer a different question:

> Who is allowed to touch this object **right now**, and in what way?

## Table-Level Locks (Structure & Lifetime)

### ACCESS SHARE (normal SELECT)

- Taken by: `SELECT`
- Blocks: only `ACCESS EXCLUSIVE`
- Meaning:  
  > I’m reading. Please don’t drop or rewrite the table.

This is why normal SELECT#1: “ACCESS SHARE is a lock… just not a blocking one”

#### A normal `SELECT` still takes a table lock:

- `SELECT` → **ACCESS SHARE**

It usually *feels* like “no lock” because it conflicts with almost nothing. But it **does** protect you from destructive operations (e.g., `DROP TABLE`, `TRUNCATE`, some `ALTER TABLE`) while the query is running.

> ACCESS SHARE is a *coordination lock* (“don’t destroy the building while I’m inside”), not a *contention lock* (“nobody else may enter”).
s feel “lock-free” — they are coordination locks, not contention locks.

### ROW EXCLUSIVE (INSERT / UPDATE / DELETE)

- Taken by: `INSERT`, `UPDATE`, `DELETE`
- Blocks: schema-changing operations
- Allows: concurrent reads and writes

This is the default write lock in PostgreSQL.

### SHARE (CREATE INDEX)

- Taken by: `CREATE INDEX` (non-concurrent)
- Blocks: all writes
- Allows: reads

#### `CREATE INDEX` (non-concurrent) is a production foot-gun

- `CREATE INDEX` (without `CONCURRENTLY`) takes a **SHARE** table lock.
- SHARE allows **reads**, but blocks **writes** (`INSERT/UPDATE/DELETE`) on that table.
- On a hot/big production table, that can mean:
  - long blocked write queues
  - request timeouts
  - cascading incidents

#### Safer alternative

Use:

```sql
CREATE INDEX CONCURRENTLY idx_orders_status ON orders(status);
```

This uses **SHARE UPDATE EXCLUSIVE** instead of SHARE — it is designed to allow ongoing reads/writes while preventing incompatible schema changes.

Rule of thumb:

> Big hot table in production? Default to `CREATE INDEX CONCURRENTLY`.

### ACCESS EXCLUSIVE (DROP / TRUNCATE / VACUUM FULL)

- Blocks: everything (reads + writes)
- Meaning:  
  > This table is completely frozen.

**Only ACCESS EXCLUSIVE blocks normal SELECT.**

## Row-Level Locks (Business Ownership)

Row locks:
- never block normal `SELECT`
- block writers or other lockers of the same row

### FOR UPDATE (strongest)

```sql
SELECT * FROM orders WHERE id = 123 FOR UPDATE;
```

- Blocks:
  - `UPDATE`
  - `DELETE`
  - any other row-level lock
- Meaning:  
  > This row is mine. Everyone else waits.

Use this for:
- payments
- order processing
- job claiming
- inventory updates

`SELECT ... FOR UPDATE` does **not** show stale rows — it waits

I had this wrong at first:

> “If one session locks a row, can another session still read the old unpaid version?”

Answer:

- normal `SELECT` can read a snapshot
- but `SELECT ... FOR UPDATE` **waits** for the lock (it does not return an “old unpaid” row)

This is exactly why `FOR UPDATE` is a good primitive for “check-then-update” workflows.

#### Example: The safe “unpaid → paid” pattern

For a payment/order state transition, the safe pattern is:

```sql
BEGIN;

SELECT status
FROM orders
WHERE id = 123
FOR UPDATE;

-- decision point is now safe
UPDATE orders SET status = 'paid'
WHERE id = 123 AND status = 'unpaid';

COMMIT;
```

What this buys you:
- prevents double-processing
- ensures only one worker can “own” the decision
- second worker blocks, then sees the committed status

### FOR NO KEY UPDATE (subtle, weaker)

```sql
SELECT * FROM orders WHERE id = 123 FOR NO KEY UPDATE;
```

Meaning:
> I will update this row, but I promise not to change its identity (PK / FK).

Key insight:
- PostgreSQL allows foreign-key readers (`FOR KEY SHARE`) to proceed
- Concurrency improves, but surprises are possible

#### A surprising case

While one transaction holds `FOR NO KEY UPDATE` on an order row:

- another transaction can still insert a payment referencing that order
- database consistency holds
- business invariants may not

Rule:

> **Use `FOR UPDATE` unless weaker semantics are fully understood.**

## Page-Level Locks (Engine Internals)

- protect in-memory pages
- held for microseconds
- invisible to application logic

These exist for **engine safety**, not concurrency control.  
Ignore them for application design.

## Spring JPA / Hibernate Lock Mapping

JPA expresses intent.  
PostgreSQL enforces reality.

| JPA Lock | PostgreSQL |
|--------|------------|
| NONE | no row lock |
| `PESSIMISTIC_READ` | `SELECT … FOR SHARE` |
| `PESSIMISTIC_WRITE` | `SELECT … FOR UPDATE` |

There is **no JPA equivalent** for `FOR NO KEY UPDATE`.

### Important Spring gotcha

This does **not** lock anything:

```java
@Transactional
Order order = repo.findById(id);
```

Unless a pessimistic lock is explicitly requested.

### Correct order-payment pattern in Spring

```java
@Transactional
@Lock(LockModeType.PESSIMISTIC_WRITE)
public void payOrder(Long orderId) {
    Order order = orderRepository.findById(orderId).orElseThrow();
    if (order.isUnpaid()) {
        order.markPaid();
    }
}
```

Maps to:

```sql
SELECT ... FOR UPDATE;
```

Second transaction blocks.  
No double pay.  
No race condition.

## Note #1: Table locks are about *table stability*, row locks are about *business ownership*

I kept mixing these until it clicked:

- **Table locks** → table exists / schema stability / table rewrite safety
- **Row locks** → who owns the business decision for a specific row

And a statement can take both:

- `SELECT ... FOR UPDATE` → table lock (`ROW SHARE`) + row lock (`FOR UPDATE`)

## Note #2: `VACUUM FULL` is a full stop

Another operational reality:

- `VACUUM FULL` takes **ACCESS EXCLUSIVE**.
- That blocks **everything** (reads + writes). Even plain `SELECT` waits.

So:

> Never run `VACUUM FULL` on a hot table unless you have a maintenance window.

Prefer:
- regular `VACUUM`
- `autovacuum` tuning
- targeted bloat investigation first

## Final cheat sheet (what I want to remember)

- `CREATE INDEX` blocks writes → use **CONCURRENTLY** on hot tables
- `VACUUM FULL` blocks reads+writes → maintenance window only
- `FOR UPDATE` is the “business ownership” lock → safe for workflows
- `FOR NO KEY UPDATE` is about key identity → can allow FK-related concurrency
- JPA pessimistic locks are just a wrapper → know the Postgres semantics
- Keep transactions short → fewer deadlocks, fewer long waits

Clarity beats cleverness.

## Takeaway

- MVCC solves *visibility*, not *coordination*
- `FOR UPDATE` is the safest primitive
- `FOR NO KEY UPDATE` is powerful but sharp
- Spring JPA locks are thin wrappers — know the database semantics
- Explicit locks + short transactions win
