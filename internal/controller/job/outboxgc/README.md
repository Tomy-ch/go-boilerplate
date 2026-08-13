# Outbox GC Job Guide (`internal/controller/job/outboxgc`)

English | [日本語](README.ja.md)

## Role in Onion Architecture

- A **one-shot GC entry point** (Controller layer / CLI driving adapter): another entry point into the Usecase layer, not a new architectural layer.
- It is the **prune half** of the [transactional outbox](../../../../docs/design/outbox.md) subsystem. Once an entry has been delivered it is marked `published`; this job batch-deletes `published` entries older than the retention window so the outbox does not grow without bound.
- Distinct from the two other outbox asynchronous halves: it is **not** the relay poll-loop (that is the resident `Engine` in `internal/controller/outbox`) and **not** dead-entry recovery (that is `ReplayUsecase`). This job only prunes `published` entries.
- An external scheduler (k8s CronJob / cron) runs it as a **cron, not a daemon** — a single `cmd job outbox-gc` invocation sweeps and exits.
- The job owns only **args parsing, span start/end, and result logging**; the sweep business is fully delegated to `outbox.GCUsecase`. It never touches the store or transactions directly.

## Public API

- `New(logging logging.Logger, tf observability.TracerFactory, gc outbox.GCUsecase) job.Job` — the DI constructor. Obtains the Controller-layer tracer via `tf.Controller()`. Registered in `internal/di/module/job.go` under `group:"jobs"`.
- Implements the `job.Job` interface (`internal/usecase/boundary/job`):
  - `Name() string` — returns the job key `"outbox-gc"`.
  - `Execute(ctx context.Context, args []string) error` — parses args, delegates to the usecase, logs the result.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| `outbox.GCUsecase` | `SweepPublished(ctx, batchSize) (int64, error)` — deletes `published` entries older than retention in batches of `batchSize` and returns the total deleted count |
| `logging.Logger` | structured result log |
| `observability.TracerFactory` | Controller-layer tracer via `tf.Controller()` |

## Execution semantics (`Execute`)

1. Start a controller span (`tracer.Start`) and `defer` its end.
2. Parse args into a batch size, then call `gc.SweepPublished(ctx, batchSize)`.
3. On success, log at **Info** with the deleted count under `logging.JobResultKey`.
4. On failure, log the deleted count at **Warn** before propagating (see [job/README.md § GC / batch jobs](../README.md) for why) The error itself is then returned as-is (propagated to the Runner / CLI, which decides the exit code — the job never calls `os.Exit()`).

## Args

Only `--batch-size=N` is accepted:

| Input | Result |
| --- | --- |
| (none) | batch size `0` → the usecase applies its own default |
| `--batch-size=N` (positive int32) | sweep in batches of `N` |
| unknown flag | error (nothing is swept) |
| `--batch-size` given more than once | error |
| `N <= 0` / non-numeric / negative | error |

## Notes

- Idempotent by design: re-running only deletes already-eligible `published` entries, so retries are safe.
- The retention window is the usecase's concern; this job passes only the batch size. For the relay loop, retention, and dead-entry replay, see the [design reference](../../../../docs/design/outbox.md).
