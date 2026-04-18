# job

English | [日本語](README.ja.md)

Provides interfaces for job definition, execution, and state management.

## Interfaces

|Interface|Description|
|---|---|
|`Job`|`Name()` + `Execute(ctx, args)` — defines a single job|
|`Runner`|`Run(ctx, jobName, args)` + `Names()` — executes and lists registered jobs|
|`State`|`Set(name, args, done)` + `Snapshot()` — holds job execution state for lifecycle hooks|

## Design Intent

- Abstract execution units to eliminate implementation dependencies
- Separate job definition (`Job`) from dispatch (`Runner`) and lifecycle (`State`)
- Enable testable batch infrastructure via mock substitution

## Implementation

- `Job`: Implemented per job in `internal/controller/job/<name>/`
- `Runner`: Assembled in `internal/controller/job/` from registered jobs
- `State`: Implemented in `internal/controller/job/` for lifecycle hook coordination

## Notes

- `Runner.Run` returns an error with available job names when a job is not found
- Each job must respect `context.Context` for cancellation and timeout
- Job names must be unique — duplicate names cause an error at Runner creation
