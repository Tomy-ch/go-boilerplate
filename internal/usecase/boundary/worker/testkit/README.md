# worker/testkit

English | [日本語](README.ja.md)

An in-memory test double for the `worker` seams (`Consumer` / `FailureHandler`),
so the engine's receive → process → ack/nack invariants can be tested green
without a real broker (a broker-agnostic 2nd implementation, no SDK dependency).

## `Fake`

`NewFake() *Fake` implements both `worker.Consumer` and `worker.FailureHandler`.
It holds an in-memory queue, in-flight set, and per-ID delivery counter, and
records every Ack / Nack / Extend / Fail call for assertions.

### Seam methods (`Consumer` / `FailureHandler`)

- `Receive(ctx, limit) ([]worker.Message, error)` — returns up to `limit`
  messages; **blocks** until an enqueue / redelivery or `ctx` completion when the
  queue is empty. Each delivery increments the message's `ReceiveCount`.
- `Ack(ctx, m) error` — removes the message from in-flight and records it.
- `Nack(ctx, m) error` — redelivers immediately (back to the queue tail) and
  records it.
- `NackWithBackoff(ctx, m, d) error` — records the requested delay `d`, then
  redelivers (the fake does not simulate real-time delay).
- `Extend(ctx, m, d) error` — records the Extend call count; returns the error
  set by `SetExtendErr` if any.
- `Fail(ctx, m, cause) error` — records the dead-letter (`FailureHandler`) call.

### Test operation / assertion helpers

- `Enqueue(msgs ...worker.Message)` — enqueues messages.
- `FailReceiveOnce(err error)` — queues one error to be returned by an upcoming
  `Receive` (call repeatedly to consume in order).
- `SetExtendErr(err error)` — makes every subsequent `Extend` return `err`.
- `AckedIDs() []string` / `NackedIDs() []string` — IDs in call order.
- `ExtendCount(id string) int` — Extend call count for the given ID.
- `NackBackoffOf(id string) time.Duration` — last requested backoff for the ID
  (0 if none).
- `NackBackoffApplied(id string) bool` — whether `NackWithBackoff` was called for
  the ID (checked by presence, since full jitter can make the delay 0).
- `Failed() []FailedRecord` — the `Fail` records in call order; `FailedRecord`
  carries `Message` and `Cause`.
- `QueueLen() int` — messages waiting to be received.
- `InflightLen() int` — messages received but not yet Ack/Nack'd.

`Fake` is safe for concurrent use (guarded by a mutex).
