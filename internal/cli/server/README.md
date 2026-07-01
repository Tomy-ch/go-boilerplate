# serve

English | [日本語](README.ja.md)

Starts the HTTP server and begins accepting requests.

## Command

```text
serve
```

## Flags

|Flag|Default|Description|
|---|---|---|
|*(none)*|||

## Usage

```bash
./server serve
```

## Notes

- Default listening port is `:8080`. Change it via configuration as needed.
- Ensure database connection settings are correctly configured before starting.
- Enable appropriate middleware (logging, security, etc.) for production deployments.
- In non-production mode only, an auxiliary metrics/pprof HTTP server (`net/http/pprof` on the default mux) is started on the address from `MetricsConfig` (`METRICS_HOST` / `METRICS_PORT`). In production mode it is not started.
- On SIGINT / SIGTERM the server shuts down gracefully. The stop context timeout (`APP_SHUTDOWN_TIMEOUT`) is measured from the moment shutdown begins, so it is not consumed by running time. The auxiliary metrics server (when present) is stopped before the application itself.
