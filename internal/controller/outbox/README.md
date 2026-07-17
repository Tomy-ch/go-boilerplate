# Outbox Relay Engine Guide (`internal/controller/outbox`)

English | [日本語](README.ja.md)

## Role in Onion Architecture

- A **poll-driven driving adapter**, on par with the HTTP handler and the worker engine — it is **another entry point into the Usecase layer**, not a new architectural layer.
- It is the **relay half** of the transactional outbox: a long-running engine that periodically polls the outbox store and drives delivery of not-yet-published rows. The counterpart **emit half** runs synchronously inside the caller's business transaction and lives in the usecase layer.
- The engine owns only the **loop and wait control**; the `claim → publish → mark` business is fully delegated to `outboxuc.RelayUsecase`. It never touches the store, the broker, or transactions directly.
- Depends only on usecase-layer ports: `outboxuc.RelayUsecase`, `clock.Sleeper`, `logging.Logger`, and `observability.LayerTracer` (obtained via `TracerFactory.Controller()`). It never imports `internal/infrastructure/*` (enforced by depguard `maintain_a_sound_controller`).

> The relay is a controller because its responsibility is **cadence orchestration** (poll interval, backoff, drain on `ctx` done, span), not domain logic. The `claim/publish/mark` transaction, the persistence port, and the HTTP send port all live behind the usecase boundary — same separation the worker engine keeps.

## Public API

- `Engine` — the resident poll engine. `NewEngine(uc, sleeper, log, tf, set) *Engine` wires it; `Run(ctx) error` is the loop body.
- `Settings` — engine tuning values, populated from `OutboxConfig` by the DI layer:
  - `BatchSize int32` — rows to claim per poll.
  - `PollInterval time.Duration` — wait after a batch that did not fully drain (empty / partial / stalled).
  - `ErrorBackoff time.Duration` — wait after `RelayBatch` returns an error.
  - **Clamping (safe defaults, not silent):** `provideRelaySettings` (`internal/di/module/outboxrelay.go`) **clamps** `BatchSize` / `PollInterval` / `ErrorBackoff` to their defaults when set to `0` / a negative value, since a non-positive poll/backoff would spin (hot loop). This is a deliberate resilience choice, not a failure. The `OUTBOX_*` env vars carry non-zero `envDefault`s, so a clamp only triggers on an explicit `0` override; it is documented here and enumerated in the setup review ([`docs/get-started/setup-repository.md`](../../../docs/get-started/setup-repository.md)) so it stays reviewable rather than silent.

## Loop semantics (`Run`)

`Run` starts a controller span, then loops until `ctx` is done (returning `nil` on completion). Each iteration calls `uc.RelayBatch(ctx, BatchSize)`, which claims up to `BatchSize` pending rows and returns a `RelayResult` (`Claimed` / `Published`). The wait decision after each iteration:

| Outcome | Next action | Why |
| --- | --- | --- |
| **Full batch with progress** (`Claimed >= BatchSize` and `Published > 0`) | **no wait** — poll again immediately | more pending rows likely remain; keep draining at full speed |
| Empty / partial / **full-but-zero-progress** (all publish failed) | wait `PollInterval` | nothing left to drain, or downstream is failing |
| `RelayBatch` error | log, then wait `ErrorBackoff` | let a transient DB/broker fault settle |

- The **full-but-zero-progress → must wait** rule is load-bearing: re-claiming a full, all-failed batch with zero wait would hot-loop while the downstream is down and burn through attempts, driving rows to `dead` instantly. A stalled full batch is always demoted to a wait.
- Waiting goes through `clock.Sleeper.Sleep(ctx, d)`, so `ctx` cancellation breaks out of the wait immediately.
- `ctx` completion is re-checked at loop top, after a `RelayBatch` error, and before lag recording, so shutdown never emits a spurious error log or an extra RPC.

## Observability

- `observeLag` records the outbox lag SLI (age of the oldest pending row) via `uc.RecordLag(ctx)` on a **best-effort** basis: it runs **only after a successful batch** (an error batch skips it to avoid double-logging the same root cause, e.g. a DB outage), skips when `ctx` is done, and on failure logs without stopping the loop.
- The span is started once per `Run` via the controller `LayerTracer`; the engine never touches the OpenTelemetry SDK directly.
- Error logs use the `logging.JobErrorKey` structured field under the `outbox-relay` logger name.

## Wiring & lifecycle

- Provided by `OutboxRelayModule` (`internal/di/module/outboxrelay.go`) and used **only in the dedicated relay process** (`cmd outbox-relay`) — the module also confines the non-standard publisher HTTP-client profile (e.g. `MaxAttempts=1`) so it cannot leak into other processes.
- `Settings` is derived from `OutboxConfig` by `provideRelaySettings`, which clamps a non-positive `BatchSize` to `outboxuc.DefaultBatchSize` to avoid a spin loop.
- `RegisterRelayHooks` (`internal/di/outboxrelay/hook`) binds the loop to the fx lifecycle via `SupervisedRunner`: `OnStart` launches `Run` in a detached goroutine (non-blocking), `OnStop` cancels the engine context and waits for the loop to unwind within the stop deadline.

## Files

- `relay.go` — `Engine` (`Run` / `waitDone` / `observeLag`), `NewEngine`, `Settings`. That is the entire adapter; there is no per-broker code here (the send port lives behind the usecase boundary, implemented in `internal/infrastructure/publisher`).

## Related

- Store boundary (persistence port the relay drives through the usecase): [`internal/usecase/boundary/outbox/README.md`](../../usecase/boundary/outbox/README.md)
- Design deep-dive (role theory / state transitions / implementation map / glossary): [docs/design/outbox.md](../../../docs/design/outbox.md)
