# worker hook

English | [日本語](README.ja.md)

`internal/di/worker/hook` is a module that registers **lifecycle hooks** to run the CLI-selected worker engine (and its health listener) across application start/stop.

## Role

`RegisterWorkerHooks` registers a Start hook and a Stop hook with `lifecycle.Registrar`:

1. On Start: starts the health listener, then calls `state.Snapshot()` to get the worker name and done channel
2. If `done == nil`: logs "No worker to run" and closes the internal done channel (engine is not started)
3. Otherwise: runs `engine.Run(engineCtx, name)` in a detached goroutine and sends the result to `done`
4. On Stop: cancels `engineCtx`, waits for the engine to drain within `stopCtx`, then stops the health listener

## Usage Flow

Set `State` before application startup from the CLI:

```go
done := make(chan error, 1)
state.Set("user-events", nil, done)
// Application starts → Start hook runs the worker until stop
err := <-done
```

## Notes

- The Start/Stop plumbing (detached goroutine, cancel-on-stop, grace-bounded drain) is delegated to `lifecycle.SupervisedRunner`; the health listener is passed as its `OnStartAux` / `OnStopAux`
- `state.Set(name, args, done)` must be called before application startup
- The engine runs in a detached goroutine; the run context is cancelled only on `OnStop` (not by `startCtx` cancellation after Start completes)
- On Stop, drain is bounded by `stopCtx`; work unfinished past the deadline is not Acked and is redelivered
- The health listener is started on `OnStart` and stopped on `OnStop`
