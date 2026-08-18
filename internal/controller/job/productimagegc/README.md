# Product Image GC Job Guide (`internal/controller/job/productimagegc`)

English | [日本語](README.ja.md)

## Role in Onion Architecture

- A **one-shot reclamation entry point** (Controller layer / CLI driving adapter): another entry point into the Usecase layer, not a new architectural layer.
- It closes the loop that synchronous deletion cannot. A DB transaction and an object-store delete cannot two-phase commit, so deleting inside the transaction would leave an unrecoverable image if the transaction rolled back. And the "uploaded an image but never created the product" path emits no domain event, so an outbox-driven cleanup cannot see it either. Store-first reconciliation is not a substitute for synchronous deletion — it covers the ground synchronous deletion cannot reach.
- An external scheduler (k8s CronJob / cron) runs it as a **cron, not a daemon** — a single `cmd job product-image-gc` invocation sweeps and exits.
- The job owns only **args parsing, span start/end, and result logging**; the reclamation itself is fully delegated to `product.ImageGCUsecase`. It never touches the store or repositories directly.
- Part of the sample Product API: this package is removed by `make setup-remove-sample-api`.

## Public API

- `New(logging logging.Logger, tf observability.TracerFactory, gc product.ImageGCUsecase) job.Job` — the DI constructor. Obtains the Controller-layer tracer via `tf.Controller()`. Registered in `internal/di/module/job.go` under `group:"jobs"`.
- Implements the `job.Job` interface (`internal/usecase/boundary/job`):
  - `Name() string` — returns the job key `"product-image-gc"`.
  - `Execute(ctx context.Context, args []string) error` — parses args, delegates to the usecase, logs the result.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| `product.ImageGCUsecase` | `SweepOrphans(ctx, grace, batchSize, dryRun) (ImageGCResult, error)` — deletes objects older than `grace` that no product references, one listing page at a time, and reports the deleted / scanned counts |
| `logging.Logger` | structured result log |
| `observability.TracerFactory` | Controller-layer tracer via `tf.Controller()` |

## Execution semantics (`Execute`)

1. Start a controller span (`tracer.Start`) and `defer` its end.
2. Parse args into a grace window, a page size and a dry-run flag, then call `gc.SweepOrphans(...)`.
3. On success, log at **Info** with the deleted count under `logging.JobResultKey` and the reconciled count under `logging.JobScannedKey`. Under `--dry-run` the message states explicitly that nothing was deleted, and the deleted count is the number of objects that *would* have been reclaimed.
4. On failure, log the same two counts at **Warn** before propagating (see [job/README.md § GC / batch jobs](../README.md) for why) The error itself is then returned as-is (propagated to the Runner / CLI, which decides the exit code — the job never calls `os.Exit()`).

## Args

Three independent flags are accepted, in any order:

| Input | Result |
| --- | --- |
| (none) | grace `0` and page size `0` → the usecase applies its own defaults, and the reclamation is real |
| `--older-than=<duration>` | Go duration string (`48h`, `1h30m`). The usecase default is 24 hours; the `d` unit does not exist in Go durations |
| `--batch-size=N` (positive int32) | list `N` objects per page — the reconciliation and the delete work on that same page |
| `--dry-run` | report the counts without deleting anything |
| unknown flag | error (nothing is reclaimed) |
| any flag given more than once | error |
| `--older-than` unparsable / `<= 0` | error |
| `--batch-size` non-numeric / `<= 0` | error |

## Notes

- **The grace window is the heart of the method.** An object uploaded a moment ago is indistinguishable from one whose product form is still being filled in, so without an age predicate the job would delete perfectly healthy uploads.
- **A failed reference lookup never falls through to a delete.** Treating "the lookup failed" as "nothing references it" would delete every image in the bucket; that is the single fatal failure mode, so the error aborts the page instead.
- Only keys under the `products/` prefix are ever considered — the prefix is re-checked after listing, so a store that ignores the prefix filter still cannot cause an unrelated object to be deleted.
- Idempotent by design: the target set is defined by an age predicate and a reference check, so re-running only reclaims what is already eligible and retries are safe. There is no exclusive locking — concurrency is the scheduler's concern ([ADR-0104 (scheduled-job-concurrency-delegated)](../../../../docs/adr/0104-scheduled-job-concurrency-delegated.md)).
- A DB-ledger design (a `product_images` table plus tombstones) would be exact and would not need to walk the bucket, but it requires a migration and a write on the upload path. It is the scaling answer, not the starting one.
