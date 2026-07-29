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
- Variables marked **Code default `<value>`** carry a `default:` tag in `internal/config/envspec.go` and are intentionally omitted from the `.env` files. They are framework-level constants that any project derived from this boilerplate keeps unchanged, so the default applies automatically; add an explicit entry to a `.env` file only when a project needs to override it. Every other variable is `required` and must be present in the relevant env file(s).

## Adding a New Variable

1. Add the field to the relevant struct in `internal/config/envspec.go` (and the matching getter struct in `model.go` / mapping in `config.go`).
2. Document it in the table below under the matching subsystem (or add a new subsystem section).
3. Decide how the value is supplied:
   - **Project-specific or per-environment value** → mark the field `required` and add it to `env/.env` (the local default) and to every per-environment file (`env/.env.<env>`).
   - **Universal framework default** → give the field a `default:` tag instead, omit it from the `.env` files, and mark it **Code default `<value>`** in the table.
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
|APP_ENV|Environment identifier|string|local / ci / dev / stg / prd|For environment distinction. Also used as the embedded-env provenance guard (see Notes)|
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown duration|duration|65s|Code default `65s`. Wait time on SIGTERM. On the HTTP server it must be `>= SERVER_REQUEST_TIMEOUT` (server startup fails otherwise) so drain never truncates an in-budget request|

### Server

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SERVER_HOST|Bind host|string|localhost|0.0.0.0 recommended in Docker|
|SERVER_PORT|Port number|int|8080||
|SERVER_READ_HEADER_TIMEOUT|Header read timeout|duration|5s|Code default `5s`. Protection against Slowloris|
|SERVER_READ_TIMEOUT|Request read timeout|duration|10s|Code default `10s`|
|SERVER_WRITE_TIMEOUT|Response write timeout|duration|65s|Code default `65s`. Must be >= SERVER_REQUEST_TIMEOUT; net/http cuts the connection before the deadline budget fires if this is shorter|
|SERVER_IDLE_TIMEOUT|KeepAlive timeout|duration|60s|Code default `60s`|
|SERVER_BODY_LIMIT_MB|Request body size limit in MB (decimal, 1MB=1,000,000 bytes); 413 on exceed|int|6|Code default `6`. Pre middleware, applied before OpenAPI validation reads the body. Must stay above `OBJECT_STORAGE_MAX_UPLOAD_BYTES` (plus multipart overhead) or the per-endpoint upload limit is unreachable. Enforced at server startup by `config.ValidateUploadBodyLimit`|
|SERVER_REQUEST_TIMEOUT|Per-request deadline budget (set once at the entry, propagated to all layers via ctx)|duration|60s|Code default `60s`. Single stop-timeout axis; statement_timeout etc. are backstops|

### Metrics

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|METRICS_HOST|metrics bind host|string|0.0.0.0||
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic auth username|string|metrics-user|Secret management required — kept out of source control; committed only for local / ci, injected at deploy time for dev / stg / prd|
|METRICS_PASSWORD|Basic auth password|string|metrics-password|Secret management required — kept out of source control; committed only for local / ci, injected at deploy time for dev / stg / prd|

### Observability

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OBS_TRACES_EXPORTER|Trace OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables tracing (lightweight)|
|OBS_METRICS_EXPORTER|Metric OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables metrics (lightweight)|
|OBS_LOGS_EXPORTER|Log OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables log export (zap stdout only)|
|OBS_OTLP_ENDPOINT|OTLP export endpoint URL|string|<http://observability:4318>|Used when an exporter is enabled|
|OBS_OTLP_PROTOCOL|OTLP protocol (`http/protobuf` or `grpc`)|string|http/protobuf|Code default `http/protobuf`|
|OBS_MASKED_DB_QUERY_ARGS|Mask DB parameters|bool|true|Security critical|
|OBS_TARGET_STATUS_CODES|Target status codes for tracing|csv|400,401,403,404,409,422,500,501,503|For error monitoring|

### Database

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|DB_DRIVER|DB driver|string|pgx|Code default `pgx`. Recommended fixed|
|DB_HOST|DB host|string|database|docker service name (change per environment)|
|DB_PORT|DB port|int|5432||
|DB_USER|User|string|postgres|Secret management recommended|
|DB_PASSWORD|Password|string|postgres-password|Secret management required|
|DB_NAME|DB name|string|local|Secret management recommended|
|DB_SSL_MODE|SSL setting|string|disable|require recommended in production|
|DB_PING_TIMEOUT|Connection check timeout|duration|10s||
|DB_SLOW_QUERY_WARN_THRESHOLD|Slow query warning threshold|duration|500ms|Code default `500ms`. Integrated with observability|
|DB_STATEMENT_TIMEOUT|Per-statement execution timeout (`statement_timeout`)|duration|30s|Code default `30s`. SQL-level backstop for queries that ignore ctx; 0 disables|
|DB_LOCK_TIMEOUT|Lock acquisition wait timeout (`lock_timeout`)|duration|10s|Code default `10s`. Backstop against long lock waits; 0 disables|
|DB_TX_MAX_RETRIES|Max tx retry attempts on serialization failure / deadlock|int|3|Code default `3`. 0 disables retry (single attempt)|
|DB_TX_RETRY_BASE_BACKOFF|Initial backoff for tx retry|duration|5ms|Code default `5ms`. Exponential base (×2)|
|DB_TX_RETRY_MAX_BACKOFF|Max backoff for tx retry|duration|100ms|Code default `100ms`. Upper bound per attempt|

### Database Connection Pool

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|DBCONN_MAX_CONNS|Maximum connections|int|10|Code default `10`|
|DBCONN_MIN_CONNS|Minimum connections|int|5|Code default `5`|
|DBCONN_MAX_LIFETIME|Connection lifetime|duration|30m|Code default `30m`|
|DBCONN_MAX_IDLE_TIME|Idle time|duration|10m|Code default `10m`|

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

### Cookie

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS only|bool|true|Required in production|
|SECURE_COOKIE_SAME_SITE|SameSite setting|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookie domain|string|example.com||

### Worker

Engine-core settings for the worker engine (broker-agnostic).

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|WORKER_CONCURRENCY|Max number of Handle executions running concurrently|int|4|Code default `4`|
|WORKER_MAX_IN_FLIGHT|Max received-but-unsettled messages|int|8|Code default `8`|
|WORKER_BATCH_SIZE|Max messages fetched per Receive|int|4|Code default `4`|
|WORKER_EXTEND_INTERVAL|Interval for calling Extend (`<= 0` disables)|duration|0s|Code default `0s`|
|WORKER_DRAIN_TIMEOUT|Upper bound for waiting on in-flight completion at shutdown|duration|30s|Code default `30s`|
|WORKER_RECEIVE_COUNT_WARN_THRESHOLD|Redelivery-count warning threshold (`<= 0` disables)|int|5|Code default `5`|
|WORKER_CIRCUIT_FAILURE_THRESHOLD|Consecutive failures that open the circuit (`<= 0` disables)|int|10|Code default `10`|
|WORKER_CIRCUIT_OPEN_BACKOFF_INITIAL|Initial cooldown while the circuit is Open|duration|1s|Code default `1s`|
|WORKER_CIRCUIT_OPEN_BACKOFF_MAX|Max cooldown while the circuit is Open|duration|30s|Code default `30s`|
|WORKER_CIRCUIT_HALF_OPEN_PROBE|Max probes attempted in Half-open|int|1|Code default `1`|
|WORKER_HEALTH_LISTEN_ADDR|Listen address for the liveness/readiness health listener|string|:8081|Code default `:8081`|
|WORKER_PROGRESS_STALE_AFTER|Time after which readiness treats progress as stale|duration|60s|Code default `60s`|
|WORKER_NACK_BACKOFF_INITIAL|Initial per-message redelivery backoff on retryable failure|duration|1s|Code default `1s`|
|WORKER_NACK_BACKOFF_MAX|Upper bound for per-message redelivery backoff|duration|30s|Code default `30s`|

### Outbox

Settings for the transactional outbox relay.

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OUTBOX_ENDPOINT|Destination endpoint URL for relayed messages|string||Code default empty. Set per project when the relay has a fixed target|
|OUTBOX_POLL_INTERVAL|Wait before the next poll after draining pending rows|duration|1s|Code default `1s`|
|OUTBOX_ERROR_BACKOFF|Wait after a relay batch returns an error|duration|5s|Code default `5s`|
|OUTBOX_BATCH_SIZE|Pending rows claimed per poll|int|100|Code default `100`|

### Auth (JWT)

Access-token (JWT) verification settings. CI / test wire a non-signature stub; `local` / `development` wire the real JWKS-backed JWT authenticator (local development verifies the mock auth server) and fail closed at startup when `AUTH_ISSUER` / `AUTH_AUDIENCE` are missing; the wiring decision lives in DI (`internal/di/module/core/auth.go`). `AUTH_JWKS_URL` is an override — when empty, the `jwks_uri` is derived from `AUTH_ISSUER` via OIDC discovery (issuer strict-match + https + same-origin; `local` allows plain http to the mock provider). The JWKS / discovery fetch goes through the `httpclient` substrate, so its HTTP timeout / retry / circuit breaker / budget come from the `jwks` downstream profile (`NewDownstreamProfile`), not an env var; that profile also blocks private-network SSRF outside `local` / `ci` / `test`.

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|AUTH_ISSUER|Expected `iss` claim value (also the OIDC issuer)|string||Code default empty. Set per environment that wires the JWT authenticator. `db-seed` also expands it into the `user_identities` seed, so an environment that seeds needs it even when it stubs authentication (CI)|
|AUTH_AUDIENCE|Expected `aud` claim value|string||Code default empty. Required together with the issuer|
|AUTH_JWKS_URL|JWKS endpoint URL override; when empty the `jwks_uri` is derived from `AUTH_ISSUER` via OIDC discovery|string||Code default empty. Internal service URL in compose (e.g. `http://mock_auth_server:4000/.well-known/jwks.json`)|
|AUTH_ALLOWED_ALGORITHMS|Allowlist of signing algorithms (comma-separated, asymmetric only)|[]string|RS256|Code default `RS256`. `none` / symmetric algorithms are always rejected|
|AUTH_CLOCK_SKEW|Clock-skew tolerance for `exp` / `nbf`|duration|60s|Code default `60s`|
|AUTH_JWKS_CACHE_TTL|Cache lifetime for a fetched JWKS|duration|1h|Code default `1h`|
|AUTH_JWKS_DISCOVERY_TTL|Cache lifetime for the OIDC discovery document (separate axis from the key cache)|duration|24h|Code default `24h`. Only used when the jwks_uri is derived via discovery|
|AUTH_JWKS_UNKNOWN_KID_COOLDOWN|Minimum interval before re-fetching JWKS on an unknown `kid` (DoS throttle)|duration|60s|Code default `60s`|

### Object Storage

S3-compatible object storage for uploaded assets (product images). The usecase depends on the vendor-neutral `objectstorage.Storage` boundary; the infrastructure implementation is an S3 adapter (AWS SDK v2 S3), so `local` connects to a Garage container while deploy environments target AWS S3 by leaving `OBJECT_STORAGE_ENDPOINT` empty. The env names stay vendor-neutral even though the adapter is S3. Values are declared per environment (no code defaults); credentials are injected at deploy time.

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OBJECT_STORAGE_ENDPOINT|S3-compatible endpoint URL; empty means SDK default resolution (AWS S3)|string|`http://garage:3900`|`required` (empty allowed). `local` points at the Garage compose service; deploy leaves it empty|
|OBJECT_STORAGE_REGION|Signing region|string|us-east-1|`required,notEmpty`|
|OBJECT_STORAGE_BUCKET|Bucket that stores objects|string|gobp-local|`required,notEmpty`|
|OBJECT_STORAGE_ACCESS_KEY_ID|Static-credential access key ID|string|gobp-local-access-key|`required,notEmpty`. Injected at deploy time|
|OBJECT_STORAGE_SECRET_ACCESS_KEY|Static-credential secret access key|string|gobp-local-secret-key|`required,notEmpty`. Injected at deploy time|
|OBJECT_STORAGE_USE_PATH_STYLE|Use path-style addressing (Garage / MinIO require true; AWS S3 uses false)|bool|true|`required`|
|OBJECT_STORAGE_MAX_UPLOAD_BYTES|Maximum accepted upload size in bytes|int|5242880|`required,notEmpty`. 5 MiB in the sample. Must stay below the global `SERVER_BODY_LIMIT_MB` (bytes, decimal) minus multipart overhead, otherwise the global body limit rejects first and this check never fires. Enforced at server startup by `config.ValidateUploadBodyLimit`|

Delivery is separate from these variables: the API returns only the object key (`imagePath`) and never a full URL, so the frontend composes `<delivery origin>/<object key>`. There is therefore no delivery-origin variable on this side — the frontend owns it (`http://gobp-local.web.garage.localhost:3902` for `local`, the CDN domain in deploy environments). See [`docker/README.md`](../docker/README.md) for how the local delivery endpoint is opened for anonymous read.

## Notes

- The Example column shows values appropriate for local development. Production values typically differ for any Secret / CIDR / Cookie-domain / origin entries.
- The `csv` type splits on `,` after trimming whitespace; do not embed commas inside individual values.
- The `duration` type accepts Go `time.ParseDuration` syntax (`500ms`, `1h30m`); plain numbers are invalid.
- When introducing a new subsystem section, keep the table column layout (`Variable Name | Description | Type | Example | Notes`) so the doc stays scannable.
- `APP_LOG_LEVEL` is set explicitly per environment: `debug` for local / ci / dev and **staging** (verbose JSON for pre-production diagnosis), `info` for production. The output format (JSON / console) is chosen by `APP_MODE`, independently of the level.
- Env files are embedded into the binary at build time (`embed.go`). `env/.env` is the local default and the single embed target; `env/.env.<env>` hold the per-environment sources. The Docker `builder` stage materializes the target via the `APP_ENV` build arg (`cp env/.env.${APP_ENV} env/.env`) before `go build`. Non-Docker flows (`go run` / `go test`) embed the committed local `env/.env`, so CI that needs another environment re-bakes it the same way (e.g. `cp env/.env.ci env/.env`). Runtime environment variables still win over embedded values.
- Embedded-env provenance guard: because the local `env/.env` (`APP_ENV=local`) is the default embed target, forgetting to materialize it before a production build would silently bake local defaults into the binary. To catch this, config validation captures the embedded `APP_ENV` before the runtime-env merge and, when the effective `APP_MODE` is `production`, rejects a non-production provenance (deny-list: `local` / `ci` / `test` / `dev` / empty). The check is deny-based, so a new environment label is tolerated by default, and `development` mode passes unconditionally (runtime injection is trusted there). The per-environment `APP_ENV` values (`local` / `ci` / `dev` / `stg` / `prd`) are the source of truth; the `Env*` constants in `internal/config/constant.go` mirror them and must not diverge. The guard fires only when the runtime injects `APP_MODE=production`: a production deployment that fails to inject it leaves the effective mode at `development` and the guard silent, so always set `APP_MODE=production` in production runtimes.
