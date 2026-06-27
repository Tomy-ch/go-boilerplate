# job

English | [日本語](README.ja.md)

Runs a registered job by name via the CLI. The job executes synchronously and blocks until completion.

## Command

```text
job <job-name> [args...] [flags]
```

## Flags

|Flag|Default|Description|
|---|---|---|
|`--timeout`|*(none / unlimited)*|Maximum execution duration (e.g. `30s`, `5m`)|

## Usage

```bash
# Run a job with a timeout
./server job user-count --timeout 30s

# Run a job without timeout
./server job cleanup-expired
```

## Notes

- The job name must correspond to a job registered in the DI layer (`di.RunJob()`).
- When `--timeout` is set, the job is cancelled if it exceeds the specified duration.
- Cleanup (`stop`) is always called after the job finishes, whether it succeeds, fails, or times out. It is bounded by the shutdown grace (`APP_SHUTDOWN_TIMEOUT`, the single stop-timeout axis, also set as `fx.StopTimeout`).
- Argument parsing and validation are the responsibility of each job implementation.
