# Orphan Cleanup Job Guide (`internal/controller/job/orphancleanup`)

## Role in Onion Architecture

- A **one-shot reclamation entry point** (Controller layer / CLI driving adapter): another entry point into the Usecase layer, not a new architectural layer.
- It is the recovery half of the [Realtime Delivery](../../../../docs/design/realtime-delivery.md) fan-out. A serve instance creates its own SQS queue and SNS subscription at start and deletes them at stop; an instance that dies without that stop leaves both behind, and its instance lease is the only index that still points at them. This job reclaims what those leases name.
- An external scheduler (k8s CronJob / cron) runs it as a **cron, not a daemon** — a single `cmd job orphan-cleanup` invocation sweeps and exits. Nothing inside the application starts it, and no in-app singleton election guards it ([ADR-0109](../../../../docs/adr/0109-scheduled-job-concurrency-delegated.md)): running two of them concurrently is safe because ownership is taken with a conditional write.
- The job owns only **args rejection, span start/end, and result logging**; which instances may be reclaimed, and in what order, is fully delegated to `realtime.OrphanSweeper`. It never touches a store or the fan-out directly.

## Public API

- `New(logging logging.Logger, tf observability.TracerFactory, newSweeper SweeperFactory) job.Job` — the DI constructor. Obtains the Controller-layer tracer via `tf.Controller()`. Registered under `group:"jobs"` by `internal/di/module/realtimecleanup.go` rather than by `JobModule()`, so the shared job module carries no Realtime dependency (see [`internal/di/module/README.md`](../../../di/module/README.md)).
- `SweeperFactory` — `func(ctx) (realtime.OrphanSweeper, error)`. The job takes a factory rather than the sweeper itself because fx executes every registered job's constructor to assemble the `Runner`: a sweeper built on the graph would make every unrelated job (`outbox-gc` and the rest) fail to start wherever Realtime is not configured. Building it here means a missing `REALTIME_TOPIC` fails this job, when it runs, and nothing else.
- Implements the `job.Job` interface (`internal/usecase/boundary/job`):
  - `Name() string` — returns the job key `"orphan-cleanup"`.
  - `Execute(ctx context.Context, args []string) error` — rejects args, delegates to the usecase, logs the result.

## Dependencies

| Dependency | Purpose |
| --- | --- |
| `realtime.OrphanSweeper` | `Sweep(ctx) (SweepResult, error)` — claims each reclaimable lease, reclaims the receiving end it names, then closes the lease |
| `logging.Logger` | structured result log |
| `observability.TracerFactory` | Controller-layer tracer via `tf.Controller()` |

## Execution semantics (`Execute`)

1. Start a controller span (`tracer.Start`) and `defer` its end.
2. Reject any argument, then call `sweeper.Sweep(ctx)`.
3. On success, log at **Info** with the counts.
4. On partial failure, log the counts at **Warn** before propagating (see [job/README.md § GC / batch jobs](../README.md) for why). The error is then returned as-is — the Runner / CLI decides the exit code, and the job never calls `os.Exit()`.

## Args

None. Any argument is an error and nothing is swept: the sweep has no knob, and silently ignoring an argument would hide a scheduler misconfiguration.

## Result counts

Reported under the job-generic log keys, not under Realtime-specific ones — the counts are a job outcome, and the sibling jobs read the same way:

| Count | Key | Meaning |
| --- | --- | --- |
| detected | `logging.JobScannedKey` | leases whose expiry is older than the cleanup margin |
| reclaimed | `logging.JobResultKey` | receiving ends reclaimed and leases closed |
| skipped | `logging.JobSkippedKey` | another sweeper held the claim, or the instance came back before the lease was closed |

The per-instance failures are not counted in the log because the returned error already carries one wrapped cause each; counting them here would restate the error chain. Turning these counts into metrics belongs to the observability phase, not here.

## Notes

- Idempotent by design: a second run finds nothing left to claim, so retries and overlapping schedules are safe.
- The timing values (expiry, cleanup margin, ownership TTL) belong to `internal/usecase/realtime`; this job knows none of them.
