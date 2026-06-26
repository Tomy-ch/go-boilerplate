# job hook

English | [日本語](README.ja.md)

`internal/di/job/hook` is a module that registers **lifecycle hooks** to automatically execute CLI-specified jobs at application startup.

## Role

`RegisterJobHooks` registers a Start hook with `lifecycle.Registrar`:

1. Calls `state.Snapshot()` to get job name, args, and done channel
2. If `done == nil`: logs and triggers `sd.Shutdown()`
3. Otherwise: executes `runner.Run(startCtx, name, args)` in a goroutine, sends result to `done`, then calls `sd.Shutdown()`

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
