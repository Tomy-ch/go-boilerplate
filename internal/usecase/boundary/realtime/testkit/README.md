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

Some states a real store reaches cannot be produced through `Append`, so each has its own entry
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
- `SeedAppendedThrough(streamID, seq)` — sets the append watermark without appending, which is how a
  test reaches the state retention leaves behind: the events are gone while the log still knows how
  far it was written. `Append` maintains the watermark on its own, so this is only for the states
  `Append` cannot reach.

## Test Strategy

A fake is production code for the tests that depend on it: when it drifts from the port, every test
built on it keeps passing while proving something the real store never does. Walking up from here
reaches `internal/usecase/README.md`, whose Testing Strategy is about interactors — mock the
boundary, never touch infrastructure — and says nothing about how a boundary's own fake is pinned.
These are that missing baseline.

- **Every port method against the interface contract** — `Append` (validation, idempotent re-append,
  conflict on a different `EventID`), `ReadAfter` (ascending order, exclusive `After`, per-stream
  isolation, `HasMore` when truncated), `Latest`, `Find`. The contract is
  `boundary/realtime/eventlog.go`, not this implementation.
- **The default read limit** — `ReadAfter` with no `Limit` truncates at 32 and reports `HasMore`.
  A fake that silently returns everything makes a caller's paging look correct when it is not.
- **Each control port on both sides** — `Seed` writes what `Append` refuses (a gap, an invalid
  envelope); `SeedAppendedThrough` moves the watermark without an append; `SetUnavailable` fails
  every read and write while set and stops when cleared; `Hold`
  blocks a read until released, and its release is idempotent. A control port that half-works is
  worse than none, because the scenario it was added for silently stops being reproduced.
- **Concurrent use** — the README promises the fake is safe for concurrent use, and a fake driven by
  a replay loop and a wakeup at once is exactly how it will be used. Pin it with a test the race
  detector can fail on rather than leaving the promise to inspection.

## `StreamTicketStore`

`NewStreamTicketStore() *StreamTicketStore` is an in-memory `rt.StreamTicketStore`. It exists for the
one contract the generated mock cannot express: revocation is only real when *invalidation and
notification both happen*, so a test has to observe a store that a later `Find` reads back. Verifying
that a revoked ticket cannot reconnect needs the issuer, the verifier and `AccessRevoker` over one
store, not a call-expectation script.

- `Save(ctx, ticket)` — re-saving the same `Hash` overwrites, as the port specifies.
- `Find(ctx, hash, asOf)` — returns `ok=false` for an unknown hash and for one that has reached
  `ExpiresAt`. Expiry is decided here rather than by a sweep, which is the same split the real store
  makes.
- `Invalidate(ctx, subject, destination)` — drops every ticket bound to that pair and succeeds when
  none match.
- `Len()` — how many tickets are held, so a test can assert that invalidation removed them rather
  than merely hid them from `Find`.
