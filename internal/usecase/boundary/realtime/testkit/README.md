# realtime/testkit

Test doubles for the `boundary/realtime` ports, so replay and cursor logic can be driven from a
real store's semantics instead of a per-call expectation script.

## `EventLog`

`NewEventLog() *EventLog` is an in-memory `rt.EventLogStore`. Reads always observe the latest write,
which is the strong-consistency behaviour the port specifies. It is safe for concurrent use.

The generated gomock (`boundary/realtime/mock`) stays the right tool for asserting *that a call was
made with given arguments*. This fake is for the tests that instead need a store which behaves like
one across many calls — a replay loop paging forward, a wakeup arriving while a read is in flight,
a client reconnecting and resuming.

- `Append(ctx, event)` — validates the envelope, then writes it once per `(StreamID, Sequence)`.
  Re-appending the same `EventID` succeeds; a different `EventID` at the same position returns
  `rt.ErrSequenceConflict`.
- `ReadAfter(ctx, q)` / `Latest(ctx, streamID)` / `Find(ctx, streamID, seq)` — as the port defines
  them. `ReadAfter` falls back to a limit of 32 when the query does not set one.

Two states a real store reaches cannot be produced through `Append`, so each has its own entry
point:

- `Seed(events…)` — writes without validation or the idempotency check, which is how a test builds a
  **gap** (a sequence the retention has already removed). A cursor pointing below such a gap is what
  makes `CursorValidator` refuse and what makes an established connection resync.
- `SetUnavailable(bool)` — every read and write returns `apperror.ErrUnavailable` while set, which is
  how a test drives the degraded paths: refusing a new connection with `503`, and keeping an
  established one open until the dependency returns.
- `Hold() func()` — every read blocks until the returned function is called. This is the only way to
  hold a read *in flight*, which is what a test needs to occupy a replay slot and watch the next
  connection be refused admission. `SetUnavailable` cannot stand in for it: a read that fails
  releases the slot immediately.
