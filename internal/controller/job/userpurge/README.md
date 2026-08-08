# User Purge Job Guide (`internal/controller/job/userpurge`)

English | [日本語](README.ja.md)

## Role in Onion Architecture

- A **one-shot purge entry point** (Controller layer / CLI driving adapter): another entry point into the Usecase layer, not a new architectural layer.
- It is the **retention half of account deletion**. Withdrawal itself only marks a user as deleted (a logical delete); this job is what finally erases the row and its dependents once the retention window has passed.
- An external scheduler (k8s CronJob / cron) runs it as a **cron, not a daemon** — a single `cmd job user-purge` invocation sweeps and exits.
- The job owns only **args parsing, span start/end, and result logging**; the purge business is fully delegated to `user.PurgeUsecase`. It never touches repositories or transactions directly.
- Part of the sample User API: this package is removed by `make setup-remove-sample-api`.

## Public API

- `New(logging logging.Logger, tf observability.TracerFactory, purge user.PurgeUsecase) job.Job` — the DI constructor. Obtains the Controller-layer tracer via `tf.Controller()`. Registered in `internal/di/module/job.go` under `group:"jobs"`.
- Implements the `job.Job` interface (`internal/usecase/boundary/job`):
  - `Name() string` — returns the job key `"user-purge"`.
  - `Execute(ctx context.Context, args []string) error` — parses args, delegates to the usecase, logs the result.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| `user.PurgeUsecase` | `PurgeDeleted(ctx, retention, batchSize, dryRun) (PurgeResult, error)` — erases users whose withdrawal is older than `retention`, in batches of `batchSize`, and reports the purged / skipped counts |
| `logging.Logger` | structured result log |
| `observability.TracerFactory` | Controller-layer tracer via `tf.Controller()` |

## Execution semantics (`Execute`)

1. Start a controller span (`tracer.Start`) and `defer` its end.
2. Parse args into a retention window, a batch size and a dry-run flag, then call `purge.PurgeDeleted(...)`.
3. On success, log at **Info** with the purged count under `logging.JobResultKey` and the skipped count under `logging.JobSkippedKey`. Under `--dry-run` the message states explicitly that nothing was deleted, and the purged count is the number of users that *would* have been erased.
4. On failure, log the same two counts at **Warn** before propagating. The usecase reports what it committed before it stopped, and a committed physical delete cannot be undone, so dropping the counts would hide users that are already gone. The error itself is then returned as-is (propagated to the Runner / CLI, which decides the exit code — the job never calls `os.Exit()`).

## Args

Three independent flags are accepted, in any order:

| Input | Result |
| --- | --- |
| (none) | retention `0` and batch size `0` → the usecase applies its own defaults, and the purge is real |
| `--older-than=<duration>` | Go duration string (`720h`, `1h30m`). 30 days is the usecase default, so a month is `--older-than=720h` — the `d` unit does not exist in Go durations |
| `--batch-size=N` (positive int32) | purge in batches of `N` |
| `--dry-run` | report the counts without deleting anything |
| unknown flag | error (nothing is purged) |
| any flag given more than once | error |
| `--older-than` unparsable / `<= 0` | error |
| `--batch-size` non-numeric / `<= 0` | error |

## Notes

- Idempotent by design: the target set is defined by an age predicate, so re-running only erases users that are already eligible and retries are safe. There is no exclusive locking — concurrency is the scheduler's concern ([ADR-0101](../../../../docs/adr/0101-scheduled-job-concurrency-delegated.md)).
- A user who still holds purchases is **not** erased; it is counted under `logging.JobSkippedKey` instead. Deciding that is the usecase's concern, not this job's.
- The retention window, batching, and the skip rule are the usecase's concern; this job only converts CLI syntax into typed values.
