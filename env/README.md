# Environment Variables List (Mapping Table)

English | [日本語](README.ja.md)

This directory is the canonical reference for every environment variable read by the application. Variables are loaded into typed Go structs under `internal/config/` and grouped here by subsystem (OS / Application / Server / Database / Security / …). Use this README when adding a new variable, auditing what each service reads, or onboarding a new operator.

## Conventions

- Variable names follow `{SUBSYSTEM}_{NAME}` in UPPER_SNAKE_CASE.
- The Type column maps to a Go type loaded by `internal/config/`:
  - `string` → `string`, `int` → `int`, `bool` → `bool`
  - `duration` → `time.Duration` (parsed via `time.ParseDuration`, e.g. `500ms`, `1h30m`)
  - `csv` → `[]string` (split on `,` after whitespace trim)
- Variables marked **Secret management required** MUST be loaded from a secret manager in production — never commit them to plain `.env` files.
- Variables marked **Secret management recommended** should be rotated periodically.

## Adding a New Variable

1. Add the field to the relevant struct in `internal/config/`.
2. Document it in the table below under the matching subsystem (or add a new subsystem section).
3. Add the variable to `env/.env` (the local default) and to every per-environment file (`env/.env.<env>`).
4. Run `make test` to confirm the config struct still loads.

## Variables by Subsystem

### OS

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OS_TZ|Timezone setting|string|Asia/Tokyo|Time reference for container / application|

### Application

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|APP_MODE|Execution mode|string|development / production|Switch logs and behavior|
|APP_LOG_LEVEL|Log output level|string|debug / info / warn / error|Output format follows Mode; level is set explicitly per environment|
|APP_NAME|Application name|string|Boilerplate|Used for log / metrics identification|
|APP_ENV|Environment identifier|string|local / staging / prod|For environment distinction|
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown duration|duration|45s|Wait time on SIGTERM|

### Server

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SERVER_HOST|Bind host|string|localhost|0.0.0.0 recommended in Docker|
|SERVER_PORT|Port number|int|8080||
|SERVER_READ_HEADER_TIMEOUT|Header read timeout|duration|5s|Protection against Slowloris|
|SERVER_READ_TIMEOUT|Request read timeout|duration|10s||
|SERVER_WRITE_TIMEOUT|Response write timeout|duration|10s||
|SERVER_IDLE_TIMEOUT|KeepAlive timeout|duration|60s||
|SERVER_REQUEST_TIMEOUT|Per-request deadline budget (set once at the entry, propagated to all layers via ctx)|duration|60s|Single stop-timeout axis; statement_timeout etc. are backstops|

### Metrics

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|METRICS_HOST|metrics bind host|string|0.0.0.0||
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic auth username|string|metrics-user|Secret management recommended (production)|
|METRICS_PASSWORD|Basic auth password|string|metrics-password|Must be changed in production / secret management recommended|

### Observability

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OBS_TRACES_EXPORTER|Trace OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables tracing (lightweight)|
|OBS_METRICS_EXPORTER|Metric OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables metrics (lightweight)|
|OBS_LOGS_EXPORTER|Log OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables log export (zap stdout only)|
|OBS_OTLP_ENDPOINT|OTLP export endpoint URL|string|<http://observability:4318>|Used when an exporter is enabled|
|OBS_OTLP_PROTOCOL|OTLP protocol (`http/protobuf` or `grpc`)|string|http/protobuf|Default `http/protobuf`|
|OBS_MASKED_DB_QUERY_ARGS|Mask DB parameters|bool|true|Security critical|
|OBS_TARGET_STATUS_CODES|Target status codes for tracing|csv|400,401,403,404,409,422,500,501,503|For error monitoring|

### Database

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|DB_DRIVER|DB driver|string|pgx|Recommended fixed|
|DB_HOST|DB host|string|database|docker service name (change per environment)|
|DB_PORT|DB port|int|5432||
|DB_USER|User|string|postgres|Secret management recommended|
|DB_PASSWORD|Password|string|postgres-password|Secret management required|
|DB_NAME|DB name|string|local|Secret management recommended|
|DB_SSL_MODE|SSL setting|string|disable|require recommended in production|
|DB_PING_TIMEOUT|Connection check timeout|duration|10s||
|DB_SLOW_QUERY_WARN_THRESHOLD|Slow query warning threshold|duration|500ms|Integrated with observability|

### Database Connection Pool

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|DBCONN_MAX_CONNS|Maximum connections|int|10||
|DBCONN_MIN_CONNS|Minimum connections|int|5||
|DBCONN_MAX_LIFETIME|Connection lifetime|duration|30m||
|DBCONN_MAX_IDLE_TIME|Idle time|duration|10m||

### Security

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SECURITY_ALLOWED_ORIGINS|CORS allow|csv|<http://localhost:3000,http://localhost:8000>||
|SECURITY_CIDR|Allowed IP range|string|127.0.0.0/8||
|SECURITY_CONTENT_TYPE_NOSNIFF|X-Content-Type-Options|string|nosniff||
|SECURITY_X_FRAME_OPTIONS|Clickjacking protection|string|DENY||
|SECURITY_HSTS_MAX_AGE|HSTS duration|duration|8760h||
|SECURITY_HSTS_EXCLUDE_SUBDOMAINS|Exclude subdomains|bool|false||
|SECURITY_HSTS_PRELOAD_ENABLED|Enable preload|bool|false||
|SECURITY_REFERRER_POLICY|Referrer control|string|no-referrer||
|SECURITY_BCRYPT_COST|bcrypt cost|int|12||

### Cookie

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS only|bool|true|Required in production|
|SECURE_COOKIE_SAME_SITE|SameSite setting|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookie domain|string|example.com||

### Auth

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|AUTH_COOKIE_NAME|Auth cookie name|string|auth_token||
|AUTH_HEADER_NAME|Header name|string|Authorization||
|AUTH_ALLOWED_HEADER_BEARER|Allow Bearer|bool|true||

## Notes

- The Example column shows values appropriate for local development. Production values typically differ for any Secret / CIDR / Cookie-domain / origin entries.
- The `csv` type splits on `,` after trimming whitespace; do not embed commas inside individual values.
- The `duration` type accepts Go `time.ParseDuration` syntax (`500ms`, `1h30m`); plain numbers are invalid.
- When introducing a new subsystem section, keep the table column layout (`Variable Name | Description | Type | Example | Notes`) so the doc stays scannable.
- `APP_LOG_LEVEL` is set explicitly per environment: `debug` for local / ci / dev and **staging** (verbose JSON for pre-production diagnosis), `info` for production. The output format (JSON / console) is chosen by `APP_MODE`, independently of the level.
- Env files are embedded into the binary at build time (`embed.go`). `env/.env` is the local default and the single embed target; `env/.env.<env>` hold the per-environment sources. The Docker `builder` stage materializes the target via the `APP_ENV` build arg (`cp env/.env.${APP_ENV} env/.env`) before `go build`. Non-Docker flows (`go run` / `go test`) embed the committed local `env/.env`, so CI that needs another environment re-bakes it the same way (e.g. `cp env/.env.ci env/.env`). Runtime environment variables still win over embedded values.
