# job hook

English | [日本語](README.ja.md)

`internal/di/job/hook` is a module that registers **lifecycle hooks** to automatically execute CLI-specified jobs at application startup.

## Role

`RegisterJobHooks` wires the job into a `lifecycle.SupervisedRunner` (the shared primitive also used by the worker / outbox-relay hooks), which registers both a Start and a Stop hook with `lifecycle.Registrar`:

1. Calls `state.Snapshot()` to get job name, args, and done channel
2. If `done == nil`: logs and triggers `sd.Shutdown()`
3. Otherwise: executes `runner.Run(jobCtx, name, args)` in a goroutine, sends result to `done`, then calls `sd.Shutdown()`

`jobCtx` is the run context supplied by `SupervisedRunner`: derived from `context.Background()` (so it is not affected by the start context being cancelled after `OnStart`) and **cancelled on `OnStop`**. This is what makes `--timeout` work: when the CLI exceeds the timeout and calls `app.Stop`, `OnStop` cancels `jobCtx`, interrupting the in-flight job (e.g. a long DB query). See `lifecycle/README.md` (SupervisedRunner).

## Usage Flow

Set `State` before application startup from the CLI:

```go
done := make(chan error, 1)
state.Set("user-count", []string{"--active-only"}, done)
// Application starts → Start hook executes the job
err := <-done
```

## Notes

- `state.Set(name, args, done)` must be called before application startup
- Job execution starts asynchronously in a separate goroutine
- The `done` channel is closed by the hook side (callers should not close it)
- `shutdowner.Shutdown()` triggers application stop after job completion
- On `OnStop` the run context is cancelled, so a job still running (e.g. on `--timeout`) is interrupted rather than detached
