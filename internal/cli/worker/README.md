# worker

English | [日本語](README.ja.md)

Starts a named pull-ack worker as a resident process and runs it until a termination signal arrives.

## Command

```text
worker <worker-name> [args...]
```

## Flags

This command takes no flags. The worker name is required and any following arguments are passed through to the worker implementation.

## Usage

```bash
# Start a worker by name
./server worker myworker

# Pass extra arguments through to the worker
./server worker myworker --some-arg value
```

## Notes

- Unlike `job`, the worker is a resident process: it keeps running and waits on the engine's completion channel rather than exiting after one execution.
- On SIGINT / SIGTERM the engine is drained and stopped gracefully; the actual engine result (including a late `Fatal`) is always awaited so failures are not silently dropped.
- The engine may also self-stop (e.g. `Fatal` or unknown worker), in which case the process exits with that result.
- Graceful stop is bounded by the shutdown grace (`APP_SHUTDOWN_TIMEOUT`, the single stop-timeout axis), measured from the moment shutdown begins; the stop context drops cancellation but keeps trace/baggage. The same grace is set as `fx.StopTimeout` so fx's 15s default does not cut the drain short, and startup validation enforces `WORKER_DRAIN_TIMEOUT < APP_SHUTDOWN_TIMEOUT`.
- The package can expose a health listener (`/healthz` liveness, `/readyz` readiness) on a dedicated mux, separate from the metrics/pprof server.
