# Realtime Consumer Engine Guide (`internal/controller/realtime`)

## Role in Onion Architecture

- A **consume-driven driving adapter**, on par with the outbox relay and the worker engine — another
  entry point into the mechanism, not a new layer. It is the receive half of the fan-out
  ([ADR-0073](../../../docs/adr/0073-sns-sqs-instance-fanout.md)): the serve instance's own queue
  (`realtime.InstanceSubscription`) is drained here and every notification is handed to the connection
  side.
- The engine owns only the **loop, the per-batch coalescing and the wait control**. What a wakeup
  *means* — which connections to wake, which to close — belongs to the sinks, which the connection
  registry in `controller/stream` implements (Phase 6 of #1187). This package declares the sinks it
  needs and nothing else about connections.
- Depends only on usecase-layer ports: `realtime.InstanceSubscription`, `ucrealtime.LeaseKeeper`,
  `clock.Sleeper`, `logging.Logger`, and `observability.LayerTracer` (via `TracerFactory.Controller()`).
  It never imports `internal/infrastructure/*` (depguard `maintain_a_sound_controller`) and never names
  `InstanceLeaseStore` (the architecture test's allowlist).

## Public API

- `Engine` — the resident consumer. `NewEngine(sub, wakeups, revocations, sleeper, log, tf, set)`;
  `Run(ctx) error` is the loop body.
- `Settings` — `BatchSize` (default 10, the queue's own cap) and `ErrorBackoff` (default 5 s). Zero
  or negative values fall back to the defaults; there is no config for them because the receive is a long poll and
  neither value changes per deployment.
- `WakeupSink.Wake(ctx, streamID, upTo)` / `RevocationSink.Revoke(ctx, subject, destination)` — the receivers.
  Both are called synchronously on the loop, so an implementation only marks or signals; it never waits
  for a replay. Duplicates are normal and must be idempotent.
- `Heartbeat` — `NewHeartbeat(keeper, id, sleeper, log, tf)`; `Run(ctx)` writes the instance lease at
  once and then every `ucrealtime.LeaseHeartbeatInterval`. A single failure is logged and retried on
  the next tick; only a silence longer than `LeaseExpiry` makes the instance an orphan.

## Loop semantics (`Run`)

| Outcome of `Receive` | Next action | Why |
| --- | --- | --- |
| notifications | coalesce → sinks → `Delete` each → receive again | the receive is a long poll (20 s); there is nothing to wait for on success |
| none | receive again | same |
| error | log, wait `ErrorBackoff` | let a transient fault settle instead of hot-looping against it |
| `ctx` done | return `nil` | checked at loop top, after an error, and by the sleeper |

- **Coalescing** is per batch: wakeups for the same stream collapse to the highest sequence, because
  each one only says "re-read after your cursor". Coalescing *across* batches is the registry's, not the
  engine's.
- A notification whose kind could not be read (`Kind == ""`) is counted, logged at warn, and deleted —
  nobody can act on it and leaving it would redeliver it forever.
- **Delete after hand-off.** A failed delete is logged and the loop continues; the notification comes
  back and the sink's idempotency absorbs it. A delete that fails because the engine is stopping is not
  logged — the cancellation is the cause, not the substrate — and the heartbeat treats a `Beat` that
  fails during stop the same way. Losing a wakeup is the worse failure, and the periodic
  catch-up covers even that.

## Test strategy

As for the other loop-driven controllers ([`../README.md`](../README.md)): the instance subscription and the lease
keeper are generated mocks, the sleeper is mocked so no test sleeps, and the loop is exercised as a
loop — one iteration's effect (batch → sinks → deletes, with the coalescing asserted through a
recording sink), stop semantics (cancel at loop top, inside a receive, inside the backoff), the
per-iteration error path (backoff and continue, delete failure logged), and the settings defaults.
