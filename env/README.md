# Environment Variables List (Mapping Table)

English | [日本語](README.ja.md)

This directory is the canonical reference for every environment variable read by the application. Variables are loaded into typed Go structs under `internal/config/` and grouped here by subsystem (OS / Application / Server / Database / Security / …). Use this README when adding a new variable, auditing what each service reads, or onboarding a new operator.

## Conventions

- Variable names follow `{SUBSYSTEM}_{NAME}` in UPPER_SNAKE_CASE.
- The Type column maps to a Go type loaded by `internal/config/`. The vocabulary below is closed, and `TestEnvReadmeTypeVocabulary` (`internal/architest`) enforces it in both directions: a Type cell outside the vocabulary fails the build — which also catches a row whose Markdown cells have shifted — and the enumeration itself must agree with the vocabulary declared in that test, so adding a type means touching both:
  - `string` → `string`, `int` → `int`, `bool` → `bool`
  - `duration` → `time.Duration` (parsed via `time.ParseDuration`, e.g. `500ms`, `1h30m`)
  - `csv` → a slice split on `,` after whitespace trim — `[]string`, or `[]int` where the elements are numeric (`OBS_TARGET_STATUS_CODES`)
- Variables marked **Secret management required** MUST be loaded from a secret manager in production — never commit them to plain `.env` files.
- Variables marked **Secret management recommended** should be rotated periodically.
- The Example column carries the **effective local value**: the value `env/.env` assigns to the key, or the `envDefault` tag in `internal/config/envspec.go` when the key is absent from `env/.env`. It is a single value, not a set — write the accepted values in the Description column instead (as `APP_MODE` and `OBS_TRACES_EXPORTER` do). A row whose Notes say **Example is a placeholder** is the sole exception, and only where copying the real value here would duplicate a credential the secret scanner already tracks in `env/.env`; read `env/.env` for what the local stack uses. Marking a variable **Secret management required** is *not* an exception on its own — those rows still show their real local value, because the local `.env` files commit them anyway. `TestEnvReadmeExamples` (`internal/architest`) enforces the column over every variable, so changing a value in `env/.env` or in an `envDefault` tag without updating the table fails the build.
- A value that legitimately differs between `env/.env.<env>` files is marked **Per-environment value** in its Notes cell, followed by why it differs. Nothing else records whether a key is a per-environment policy or a value that should match everywhere, so an undocumented difference reads as a propagation miss and should be treated as one. `TestEnvPerEnvironmentValuePolicy` (`internal/architest`) enforces the marker in both directions: an unmarked key must hold the same value in every env file that declares it, and a marked key must actually differ, so a marker left behind after the values were aligned fails too. Values are compared as **effective** values, so a key with a `Code default` counts as holding that default in every file that omits it — overriding it in a single env file is a difference and needs the marker just as much. A key with no `Code default` has no effective value where it is absent, because the value is injected from outside; that absence is not compared here and is covered by **Injected at deploy time** below.
- A key that carries no `envDefault` must appear in every env file; the one legitimate absence is a value the deploy platform injects at run time, and that is declared by marking the Notes cell **Injected at deploy time**. Without such a declaration a missing key is indistinguishable from a propagation miss, and it stays invisible until the environment actually boots and `required` validation rejects it. `TestEnvRequiredKeyPresencePolicy` (`internal/architest`) enforces the marker in both directions: an unmarked key missing from any env file fails, and a marked key must be committed in `local` / `ci` and absent from `dev` / `stg` / `prd`, so a value that reappears in a deploy file — or a marker put on a key that has a `Code default` — fails too.
- The Japanese translation ([README.ja.md](README.ja.md)) duplicates this table in full, values included, and most readers of this repository read that side. Prose is translated, but the key, Type, Example, and `Code default` values are language-independent and MUST match this file — `internal/architest` (`TestEnvReadmeTranslationValues`) fails the build on a divergence, and `TestEnvReadmeTranslationStructure` does the same for the document structure — the subsystem headings that group the table, plus the section and list-item counts, so a paragraph dropped in translation is caught too. A `Code default` that is empty cannot be written in backticks, so it is spelled out as **Code default empty** here and as **Code default は空** in the translation; those two spellings are declared in the test and are the only accepted forms.
- Variables marked **Code default `<value>`** carry an `envDefault:` tag in `internal/config/envspec.go` and are intentionally omitted from the `.env` files. They are framework-level constants that any project derived from this boilerplate keeps unchanged, so the default applies automatically; add an explicit entry to a `.env` file only when a project needs to override it. Every other variable is `required` and must be present in the relevant env file(s).
- A key whose local value comes from the compose stack rather than an env file is the one case this directory does not fully describe. `internal/config` reads the process environment first, so an `environment` entry in a `docker-compose*.yaml` service wins over the embedded `env/.env`, and a key supplied only that way is absent from every env file — which puts it outside what `internal/architest` compares. Reach for it only when an env file genuinely cannot hold the value, keep the row's `Code default` accurate so a reader still knows what the binary does without compose, and expect no test to catch a stale compose entry.

## Adding a New Variable

1. Add the field to the relevant struct in `internal/config/envspec.go` (and the matching getter struct in `model.go` / mapping in `config.go`).
2. Document it in the table below under the matching subsystem (or add a new subsystem section).
3. Decide how the value is supplied:
   - **Project-specific or per-environment value** → mark the field `required` and add it to `env/.env` (the local default) and to every per-environment file (`env/.env.<env>`). Leave it out of `env/.env.dev` / `.stg` / `.prd` only when the deploy platform injects it at run time, and mark the row **Injected at deploy time** when you do.
   - **Universal framework default** → give the field an `envDefault:` tag instead, omit it from the `.env` files, and mark it **Code default `<value>`** in the table.
4. Run `make test` to confirm the config struct still loads.

## Changing the Timezone

The timezone is supplied by **three independent mechanisms that set different layers**, so a project that moves off `Asia/Tokyo` has to change all of them. They are listed here because nothing else in the repository records the full set.

- **Session timezone** — `OS_TZ` becomes the `timezone` parameter of the DSN the application uses for every database connection (`internal/infrastructure/rdb/driver/config.go`). This is the only mechanism that decides what the application reads and writes, and it takes precedence over any database-side setting.
- **Database-side default** — the `TZ` environment variable of the PostgreSQL container. `initdb` writes it into `postgresql.conf`, making it the cluster default that every database created afterwards inherits (`wt<N>_local` / `wt<N>_test` from the worktree slot pool, `gen_schema` from `make dump-schema`). It only affects what a client that does *not* specify a timezone sees, so its purpose is the developer experience of reading timestamps directly through `psql` / pgweb / SchemaSpy.
- **Application container local time** — the `TZ` environment variable of the application image (`docker/server/Dockerfile`, both the `runtime` and `tooling` stages, which install `tzdata` for it). It is what Go reads to resolve `time.Local`, and therefore what `date` inside the container and the local-time rendering of logs show. `OS_TZ` cannot fill this role: Go reads the plain `TZ` name, and a process with neither `TZ` nor `/etc/localtime` falls back to UTC. Correctness does not rest on it — `time.Local` is banned by `forbidigo` in `.golangci-full.yaml`, so application code takes the timezone from the injected `*time.Location` (`config.NewTimeLocation`) instead. This mechanism exists so that a developer reading the container never sees a timezone the configuration did not ask for.

All three are needed: dropping the first would leave the application at the cluster default, dropping the second would show UTC in every direct database session, and dropping the third would show UTC in every shell and log line inside the container. Change all of the following together:

1. `env/.env` and every `env/.env.<env>` — the `OS_TZ` entry (session timezone; `required`, so it exists in all five files).
2. `docker-compose.yaml`, the `database` service — `TZ` and `PGTZ`. `TZ` only takes effect during `initdb`, so an existing volume keeps its old cluster default until the volume is recreated; `docs/maintenance/db-worktree-pool.md` owns that procedure and its worktree caveat. `PGTZ` applies per `psql` session and therefore takes effect immediately, even on an old volume.
3. The PostgreSQL service block of each workflow under `.github/workflows/` that provisions a database — `TZ` and `PGTZ`. GitHub Actions cannot share a service definition between workflows, so the value is repeated in every one of them. Enumerate them by the service image rather than by the variable (`grep -rl 'image: postgres' .github/workflows/`) — grepping for `PGTZ` finds only the workflows that already set it and silently skips the one that still needs it.
4. `docker/server/Dockerfile` — the `ENV TZ` of the `runtime` and `tooling` stages. Both are stated because they are separate images: `runtime` is what a deployment runs, `tooling` is what `make serve` runs. A deployment can override the value at run time without a rebuild, so treat the `ENV` as the default rather than as the only place it can come from. `ENV` is baked at build time, so — like the `initdb` caveat in item 2 — an image built before the change keeps the old value: `make serve` reuses the cached `gobp-wt-<N>-api_server` image and reports success while the container still reports the previous timezone, so run `make serve-build` to pick the change up.
5. Test expectations that pin the value as a literal — `expectedOSTimeZone` in `internal/config/config_testing_mock.go`, plus the assertions in `internal/di/job_test.go`, `internal/di/server/hook/http_server_hook_test.go`, and `internal/infrastructure/rdb/driver/config_test.go`.

`internal/architest` fails on a partially propagated change, so it is caught by `make test` rather than by reading a timestamp in production: `TestTimezoneMechanismValuesMatch` when items 1 through 4 hold different values, and `TestPostgresProvisionersDeclareTimeZone` / `TestDockerfileTzdataStagesDeclareTimeZone` when a declaration is missing outright. Item 5 is not machine-checked — a stale literal there surfaces as a failing assertion in the test that pins it.

## Variables by Subsystem

### OS

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OS_TZ|Timezone setting|string|Asia/Tokyo|Time reference for container / application. The deployment region varies per project, so it is `required` and stated in every env file rather than code-defaulted — the timezone stays where an operator looks for it. Rejected when empty, because an empty value would silently fall back to UTC. Sets the session timezone only; the database-side default and the container's local time each come from a `TZ` of their own, so see [Changing the Timezone](#changing-the-timezone) before moving off `Asia/Tokyo`|

### Application

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|APP_MODE|Execution mode (`development` or `production`)|string|development|Switch logs and behavior. Per-environment value — `production` from `stg` onward so pre-production runs on the same log format and behavior as production; `development` in local / ci / dev|
|APP_LOG_LEVEL|Log output level (`debug` / `info` / `warn` / `error`)|string|debug|Output format follows Mode. Per-environment value — `debug` through `stg` for pre-production diagnosis, `info` in `prd` to hold production log volume down|
|APP_NAME|Application name|string|Boilerplate|Used for log / metrics identification|
|APP_ENV|Environment identifier (`local` / `ci` / `dev` / `stg` / `prd`)|string|local|For environment distinction. Also used as the embedded-env provenance guard (see Notes). Per-environment value — the environment identifier itself, so it differs by definition|
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown duration|duration|65s|Code default `65s`. Wait time on SIGTERM. On the HTTP server it must be `>= SERVER_REQUEST_TIMEOUT` (server startup fails otherwise) so drain never truncates an in-budget request|

### Server

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SERVER_HOST|Bind host|string|localhost|0.0.0.0 recommended in Docker. Per-environment value — the host each environment is reached at|
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
|METRICS_HOST|metrics bind host|string|0.0.0.0|Per-environment value — the host each environment exposes metrics on|
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic auth username|string|metrics-user|Secret management required — kept out of source control; committed only for `local` / `ci`. Injected at deploy time for `dev` / `stg` / `prd`|
|METRICS_PASSWORD|Basic auth password|string|metrics-password|Secret management required — kept out of source control; committed only for `local` / `ci`. Injected at deploy time for `dev` / `stg` / `prd`|

### Observability

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OBS_TRACES_EXPORTER|Trace OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables tracing (lightweight). Per-environment value — only `local` runs the compose observability stack, so every other environment leaves it empty until a collector is wired|
|OBS_METRICS_EXPORTER|Metric OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables metrics (lightweight). Per-environment value — only `local` runs the compose observability stack, so every other environment leaves it empty until a collector is wired|
|OBS_LOGS_EXPORTER|Log OTLP exporter (`otlp` to enable; empty/`none` to disable)|string|otlp|Empty disables log export (zap stdout only). Per-environment value — only `local` runs the compose observability stack, so every other environment leaves it empty until a collector is wired|
|OBS_OTLP_ENDPOINT|OTLP export endpoint URL|string|`http://observability:4318`|Used when an exporter is enabled. Per-environment value — the collector of each environment; empty where no exporter is enabled|
|OBS_OTLP_PROTOCOL|OTLP protocol (`http/protobuf` or `grpc`)|string|http/protobuf|Code default `http/protobuf`|
|OBS_MASKED_DB_QUERY_ARGS|Mask DB parameters|bool|false|Security critical. Per-environment value — `false` only in local / ci, where seeing the raw SQL arguments is the point while debugging a query or a failing test; `true` from `dev` onward so real payloads never reach the trace backend. Never align the upper environments down to the local value|
|OBS_TARGET_STATUS_CODES|Target status codes for tracing|csv|400,401,403,404,405,409,422,429,500,501,503|For error monitoring. Per-environment value — the set narrows monotonically as the environment gets closer to production, so a mismatch between files is the intent rather than a propagation miss. `local` / `ci` monitor the full set for development and test visibility; `dev` / `stg` drop `429`; `prd` additionally drops `403` / `404` / `405`, keeping production monitoring on server-side and contract failures rather than on client-driven noise that dominates at production traffic volume. A lower environment never monitors a code its upper environment ignores. `TestEnvTargetStatusCodesPolicy` (`internal/architest`) enforces the policy, so adding a code to some env files but not others fails the build; excluding a new code from an environment on purpose requires updating the policy declaration in that test as well|

### Database

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|DB_DRIVER|DB driver|string|pgx|Code default `pgx`. Recommended fixed|
|DB_HOST|DB host|string|database|docker service name (change per environment). Per-environment value — the compose service name in `local`, `localhost` in `ci`. Injected at deploy time for `dev` / `stg` / `prd`|
|DB_PORT|DB port|int|5432|Injected at deploy time for `dev` / `stg` / `prd`|
|DB_USER|User|string|postgres|Secret management recommended. Injected at deploy time for `dev` / `stg` / `prd`|
|DB_PASSWORD|Password|string|postgres-password|Secret management required. Injected at deploy time for `dev` / `stg` / `prd`|
|DB_NAME|DB name|string|local|Secret management recommended. Per-environment value — `local` uses the development database and `ci` the test database. Injected at deploy time for `dev` / `stg` / `prd`|
|DB_SSL_MODE|SSL setting|string|disable|require recommended in production. Injected at deploy time for `dev` / `stg` / `prd`|
|DB_PING_TIMEOUT|Connection check timeout|duration|5s|Per-environment value — `5s` in local, where the database sits on the same compose service so a slow ping means a broken startup and failing fast surfaces it; `30s` in `ci`, where many test processes create a pool against one instance at the same moment, so a connect that is merely queued behind the others must not be read as a database that is down — the value is a deliberate margin over that contention, not a budget derived from any other timeout; `10s` from `dev` onward, where a managed database is reached over the network and a transient delay is expected|
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
|DBCONN_MIN_CONNS|Minimum connections|int|5|Code default `5`. Per-environment value — pgxpool opens this many connections at once right after the pool is created, so `ci` overrides it to `0` while everywhere else keeps the default: tests do not need a pre-warmed pool, and with test processes running in parallel that burst is load on the instance and nothing else|
|DBCONN_MAX_LIFETIME|Connection lifetime|duration|30m|Code default `30m`|
|DBCONN_MAX_IDLE_TIME|Idle time|duration|10m|Code default `10m`|

### Security

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SECURITY_ALLOWED_ORIGINS|CORS allow|csv|`http://localhost:3000,http://localhost:8000`|Per-environment value — the frontend origin of each environment|
|SECURITY_CIDR|Allowed IP range|string|127.0.0.0/8||
|SECURITY_CONTENT_TYPE_NOSNIFF|X-Content-Type-Options|string|nosniff||
|SECURITY_X_FRAME_OPTIONS|Clickjacking protection|string|DENY||
|SECURITY_HSTS_MAX_AGE|HSTS duration|duration|0|Per-environment value — `0` disables HSTS in local / ci because they serve plain http, and a browser that once cached the header would refuse to load them; `8760h` (1 year) from `dev` onward, where TLS terminates in front. Never align the upper environments down to the local value — that drops HSTS in production|
|SECURITY_HSTS_EXCLUDE_SUBDOMAINS|Exclude subdomains|bool|false||
|SECURITY_HSTS_PRELOAD_ENABLED|Enable preload|bool|false||
|SECURITY_REFERRER_POLICY|Referrer control|string|no-referrer||

### Cookie

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS only|bool|true|Required in production|
|SECURE_COOKIE_SAME_SITE|SameSite setting|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookie domain|string|localhost|Per-environment value — the cookie domain of each environment|

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
|OUTBOX_PUBLISHER|Publish target kind (`http` / `sqs`)|string|http|Code default `http`. An unknown value fails startup (fail-closed). The publish target is a per-deployment choice, so no env file pins it; a deployment that publishes to a queue supplies the value at run time|
|OUTBOX_ENDPOINT|Destination endpoint URL for relayed messages|string||Code default empty. Required when `OUTBOX_PUBLISHER=http`|
|OUTBOX_POLL_INTERVAL|Wait before the next poll after draining pending rows|duration|1s|Code default `1s`|
|OUTBOX_ERROR_BACKOFF|Wait after a relay batch returns an error|duration|5s|Code default `5s`|
|OUTBOX_BATCH_SIZE|Pending rows claimed per poll|int|100|Code default `100`|
|OUTBOX_QUEUE_ENDPOINT|SQS-compatible endpoint|string|`http://elasticmq:9324`|Code default empty. Empty defers to the SDK's default resolution (real AWS SQS). **Per-environment value**: set only in local, where the broker runs in compose. Other environments leave it empty because the queue is a per-deployment resource|
|OUTBOX_QUEUE_REGION|Region used for SigV4 signing|string|us-east-1|Code default empty. Required when `OUTBOX_PUBLISHER=sqs`. **Per-environment value**: set only in local, where the broker runs in compose. Other environments leave it empty because the queue is a per-deployment resource|
|OUTBOX_QUEUE_URL|Destination queue URL|string|`http://elasticmq:9324/000000000000/gobp-events`|Code default empty. Required when `OUTBOX_PUBLISHER=sqs`. Local queue names come from `docker/elasticmq/elasticmq.conf`. **Per-environment value**: set only in local, where the broker runs in compose. Other environments leave it empty because the queue is a per-deployment resource|
|OUTBOX_QUEUE_ACCESS_KEY_ID|Static credential access key ID|string|local-dummy-access-key|Code default empty. ElasticMQ does not verify signatures locally, so any dummy works. **Per-environment value**: set only in local, where the broker runs in compose. Other environments leave it empty because the queue is a per-deployment resource|
|OUTBOX_QUEUE_SECRET_ACCESS_KEY|Static credential secret access key|string|local-dummy-secret-key|Code default empty. Replace with an IAM role or similar in production. **Per-environment value**: set only in local, where the broker runs in compose. Other environments leave it empty because the queue is a per-deployment resource|

### Auth (JWT)

Access-token (JWT) verification settings. CI / test wire a non-signature stub; `local` / `development` wire the real JWKS-backed JWT authenticator (local development verifies the mock auth server) and fail closed at startup when `AUTH_ISSUER` / `AUTH_AUDIENCE` are missing; the wiring decision lives in DI (`internal/di/module/core/auth.go`). `AUTH_JWKS_URL` is an override — when empty, the `jwks_uri` is derived from `AUTH_ISSUER` via OIDC discovery (issuer strict-match + https + same-origin; `local` allows plain http to the mock provider). The JWKS / discovery fetch goes through the `httpclient` substrate, so its HTTP timeout / retry / circuit breaker / budget come from the `jwks` downstream profile (`NewDownstreamProfile`), not an env var; that profile also blocks private-network SSRF outside `local` / `ci` / `test`.

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|AUTH_ISSUER|Expected `iss` claim value (also the OIDC issuer)|string|`http://localhost:4000`|Code default empty. Per-environment value — `local` / `ci` point at the mock auth server, and every deploy environment keeps the empty default until it wires the JWT authenticator. `db-seed` also expands it into the `user_identities` seed, so an environment that seeds needs it even when it stubs authentication (CI)|
|AUTH_AUDIENCE|Expected `aud` claim value|string|go-boilerplate-api|Code default empty. Required together with the issuer. Per-environment value — only `local` declares the mock audience; everywhere else keeps the empty default until the authenticator is wired|
|AUTH_JWKS_URL|JWKS endpoint URL override; when empty the `jwks_uri` is derived from `AUTH_ISSUER` via OIDC discovery|string|`http://mock_auth_server:4000/.well-known/jwks.json`|Code default empty. Per-environment value — only `local` overrides it with the compose-internal service URL; everywhere else keeps the empty default and derives the `jwks_uri` from `AUTH_ISSUER` via OIDC discovery|
|AUTH_ALLOWED_ALGORITHMS|Allowlist of signing algorithms (comma-separated, asymmetric only)|csv|RS256|Code default `RS256`. `none` / symmetric algorithms are always rejected|
|AUTH_CLOCK_SKEW|Clock-skew tolerance for `exp` / `nbf`|duration|60s|Code default `60s`|
|AUTH_JWKS_CACHE_TTL|Cache lifetime for a fetched JWKS|duration|1h|Code default `1h`|
|AUTH_JWKS_DISCOVERY_TTL|Cache lifetime for the OIDC discovery document (separate axis from the key cache)|duration|24h|Code default `24h`. Only used when the jwks_uri is derived via discovery|
|AUTH_JWKS_UNKNOWN_KID_COOLDOWN|Minimum interval before re-fetching JWKS on an unknown `kid` (DoS throttle)|duration|60s|Code default `60s`|

### Object Storage

S3-compatible object storage for uploaded assets (product images). The usecase depends on the vendor-neutral `objectstorage.Storage` boundary; the infrastructure implementation is an S3 adapter (AWS SDK v2 S3), so `local` connects to a Garage container while deploy environments target AWS S3 by leaving `OBJECT_STORAGE_ENDPOINT` empty. The env names stay vendor-neutral even though the adapter is S3. Values are declared per environment (no code defaults); credentials are injected at deploy time.

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OBJECT_STORAGE_ENDPOINT|S3-compatible endpoint URL; empty means SDK default resolution (AWS S3)|string|`http://garage:3900`|`required` (empty allowed). Per-environment value — `local` points at the Garage compose service; every other environment leaves it empty so the SDK resolves AWS S3|
|OBJECT_STORAGE_REGION|Signing region|string|us-east-1|`required,notEmpty`. Per-environment value — the Garage sample region in local / ci, the AWS region of the environment from `dev` onward|
|OBJECT_STORAGE_BUCKET|Bucket that stores objects|string|gobp-local|`required,notEmpty`. Per-environment value — one bucket per environment|
|OBJECT_STORAGE_ACCESS_KEY_ID|Static-credential access key ID|string|gobp-local-access-key|`required,notEmpty`. Secret management required. Injected at deploy time for `dev` / `stg` / `prd`. Example is a placeholder: `local` uses a fixed Garage credential (`GK` + 24 hex) held in `env/.env`. Per-environment value — each environment has its own credential|
|OBJECT_STORAGE_SECRET_ACCESS_KEY|Static-credential secret access key|string|gobp-local-secret-key|`required,notEmpty`. Secret management required. Injected at deploy time for `dev` / `stg` / `prd`. Example is a placeholder: `local` uses a fixed Garage credential (64 hex) held in `env/.env`, and copying it here would add a second entry to `.gitleaksignore`. Per-environment value — each environment has its own credential|
|OBJECT_STORAGE_USE_PATH_STYLE|Use path-style addressing (Garage / MinIO require true; AWS S3 uses false)|bool|true|`required`. Per-environment value — `true` in local / ci where Garage requires path-style addressing, `false` from `dev` onward for AWS S3|
|OBJECT_STORAGE_MAX_UPLOAD_BYTES|Maximum accepted upload size in bytes|int|5242880|`required,notEmpty`. 5 MiB in the sample. Must stay below the global `SERVER_BODY_LIMIT_MB` (bytes, decimal) minus multipart overhead, otherwise the global body limit rejects first and this check never fires. Enforced at server startup by `config.ValidateUploadBodyLimit`|

Delivery is separate from these variables: the API returns only the object key (`imagePath`) and never a full URL, so the frontend composes `<delivery origin>/<object key>`. There is therefore no delivery-origin variable on this side — the frontend owns it (`http://gobp-local.web.garage.localhost:3902` for `local`, the CDN domain in deploy environments). See [`docker/README.md`](../docker/README.md) for how the local delivery endpoint is opened for anonymous read.

## Notes

- The Example column is a local value and never a deploy-ready one; see Conventions for what the column is defined to hold. Which keys an environment genuinely sets differently is recorded by the **Per-environment value** marker in the table, so read the marker rather than assuming a category of variables differs.
- The `csv` type splits on `,` after trimming whitespace; do not embed commas inside individual values.
- The `duration` type accepts Go `time.ParseDuration` syntax (`500ms`, `1h30m`); plain numbers are invalid.
- When introducing a new subsystem section, keep the table column layout (`Variable Name | Description | Type | Example | Notes`) so the doc stays scannable.
- Env files are embedded into the binary at build time (`embed.go`). `env/.env` is the local default and the single embed target; `env/.env.<env>` hold the per-environment sources. The Docker `builder` stage materializes the target via the `APP_ENV` build arg (`cp env/.env.${APP_ENV} env/.env`) before `go build`. Non-Docker flows (`go run` / `go test`) embed the committed local `env/.env`, so CI that needs another environment re-bakes it the same way (e.g. `cp env/.env.ci env/.env`). Runtime environment variables still win over embedded values.
- Embedded-env provenance guard: because the local `env/.env` (`APP_ENV=local`) is the default embed target, forgetting to materialize it before a production build would silently bake local defaults into the binary. To catch this, config validation captures the embedded `APP_ENV` before the runtime-env merge and, when the effective `APP_MODE` is `production`, rejects a non-production provenance (deny-list: `local` / `ci` / `test` / `dev` / empty). The check is deny-based, so a new environment label is tolerated by default, and `development` mode passes unconditionally (runtime injection is trusted there). The per-environment `APP_ENV` values (`local` / `ci` / `dev` / `stg` / `prd`) are the source of truth; the `Env*` constants in `internal/config/constant.go` mirror them and must not diverge. The guard fires only when the runtime injects `APP_MODE=production`: a production deployment that fails to inject it leaves the effective mode at `development` and the guard silent, so always set `APP_MODE=production` in production runtimes.
