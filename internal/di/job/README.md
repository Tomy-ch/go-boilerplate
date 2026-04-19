# job DI Module

English | [日本語](README.ja.md)

This directory provides DI (dependency injection) components related to job execution. Specifically, it provides a provider that assembles a `Runner` from registered jobs, a `State` for holding execution targets and completion channels, and a mechanism to register lifecycle hooks that execute registered jobs at application startup.

## Structure

```text
internal/di/job/
├── runner.go   # Runner DI provider
└── hook/       # Lifecycle hook (job execution at startup)
```

## Public API

### runner.go

|Type / Function|Description|
|---|---|
|`RunnerIn`|`fx.In` struct holding `[]job.Job` injected via `group:"jobs"`|
|`ProvideRunner(in RunnerIn)`|Create `job.Runner` from `RunnerIn` and provide to DI container|

### hook/

Lifecycle hook for startup job execution. See [hook/README.md](hook/README.md) for details.

## DI Registration Example

```go
fx.Provide(
    job.ProvideRunner,
    job.NewState,
)
fx.Invoke(hook.RegisterJobHooks)
```

## Job Execution Flow

1. CLI sets job info via `state.Set(name, args, done)`
2. Application starts
3. Start hook references `state.Snapshot()`
4. If `done` exists, `runner.Run()` executes the job asynchronously
5. Result is sent to `done` channel, then application shuts down

## Notes

- `state.Set` must be called before application startup
- If `done` is `nil`, shutdown is triggered immediately
- Job execution runs in a separate goroutine
- To add jobs, add them to `provideJobs(...)` in `internal/di/module/job.go`
