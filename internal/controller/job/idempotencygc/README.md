# Idempotency GC Job Guide (`internal/controller/job/idempotencygc`)

English | [日本語](README.ja.md)

## Role in Onion Architecture

- A **one-shot GC entry point** (Controller layer / CLI driving adapter): another entry point into the Usecase layer, not a new architectural layer.
- It is the **housekeeping half** of the [idempotency](../../../../docs/design/idempotency.md) subsystem. The request path stamps a TTL (`expires_at`) on each `idempotency_keys` row; this job batch-deletes the rows whose TTL has already expired so the table does not grow without bound.
- An external scheduler (k8s CronJob / cron) runs it as a **cron, not a daemon** — a single `cmd job idempotency-gc` invocation sweeps and exits.
- The job owns only **args parsing, span start/end, and result logging**; the `claim → sweep → count` business is fully delegated to `idempotency.GCUsecase`. It never touches the store or transactions directly.

## Public API

- `New(logging logging.Logger, tf observability.TracerFactory, gc idempotency.GCUsecase) job.Job` — the DI constructor. Obtains the Controller-layer tracer via `tf.Controller()`. Registered in `internal/di/module/job.go` under `group:"jobs"`.
- Implements the `job.Job` interface (`internal/usecase/boundary/job`):
  - `Name() string` — returns the job key `"idempotency-gc"`.
  - `Execute(ctx context.Context, args []string) error` — parses args, delegates to the usecase, logs the result.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| `idempotency.GCUsecase` | `SweepExpired(ctx, batchSize) (int64, error)` — deletes expired rows in batches of `batchSize` and returns the total deleted count |
| `logging.Logger` | structured result log |
| `observability.TracerFactory` | Controller-layer tracer via `tf.Controller()` |

## Execution semantics (`Execute`)

1. Start a controller span (`tracer.Start`) and `defer` its end.
2. Parse args into a batch size, then call `gc.SweepExpired(ctx, batchSize)`.
3. On success, log at **Info** with the deleted count under `logging.JobResultKey`.
4. Any usecase error is returned as-is (propagated to the Runner / CLI, which decides the exit code — the job never calls `os.Exit()`).

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

- Idempotent by design: re-running only deletes already-expired rows, so retries are safe.
- Documents only what this job does. The request-path orchestration, the `Store` seam, and its infrastructure impl live in the idempotency usecase / infrastructure layers — see the [design reference](../../../../docs/design/idempotency.md).
