# Environment Variables List (Mapping Table)

## OS

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OS_TZ|Timezone setting|string|Asia/Tokyo|Time reference for container / application|

## Application

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|APP_MODE|Execution mode|string|development / production|Switch logs and behavior|
|APP_NAME|Application name|string|Boilerplate|Used for log / metrics identification|
|APP_ENV|Environment identifier|string|local / staging / prod|For environment distinction|
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown duration|duration|45s|Wait time on SIGTERM|

## Server

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SERVER_HOST|Bind host|string|localhost|0.0.0.0 recommended in Docker|
|SERVER_PORT|Port number|int|8080||
|SERVER_READ_HEADER_TIMEOUT|Header read timeout|duration|5s|Protection against Slowloris|
|SERVER_READ_TIMEOUT|Request read timeout|duration|10s||
|SERVER_WRITE_TIMEOUT|Response write timeout|duration|10s||
|SERVER_IDLE_TIMEOUT|KeepAlive timeout|duration|60s||

## Metrics

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|METRICS_HOST|metrics bind host|string|0.0.0.0||
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic auth username|string|metrics-user|Secret management recommended (production)|
|METRICS_PASSWORD|Basic auth password|string|metrics-password|Must be changed in production / secret management recommended|

## Observability

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|OBSERVABILITY_ENABLED|Enable trace/log|bool|true||
|OBSERVABILITY_MASKED_DB_QUERY_ARGS|Mask DB parameters|bool|true|Security critical|
|OBSERVABILITY_TARGET_STATUS_CODES|Target status codes for tracing|csv|400,401,403,404,409,422,500,501,503|For error monitoring|

## Database

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

## Database Connection Pool

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|DBCONN_MAX_CONNS|Maximum connections|int|10||
|DBCONN_MIN_CONNS|Minimum connections|int|5||
|DBCONN_MAX_LIFETIME|Connection lifetime|duration|30m||
|DBCONN_MAX_IDLE_TIME|Idle time|duration|10m||

## Security

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

## Cookie

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS only|bool|true|Required in production|
|SECURE_COOKIE_SAME_SITE|SameSite setting|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookie domain|string|example.com||

## Auth

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|AUTH_COOKIE_NAME|Auth cookie name|string|auth_token||
|AUTH_HEADER_NAME|Header name|string|Authorization||
|AUTH_ALLOWED_HEADER_BEARER|Allow Bearer|bool|true||

## IP Rate Limiter

|Variable Name|Description|Type|Example|Notes|
|---|---|---|---|---|
|IP_RATE_LIMITER_ENABLED|Enable|bool|true||
|IP_RATE_LIMITER_REQUESTS|Allowed requests|int|60||
|IP_RATE_LIMITER_PER|Period|duration|1m||
|IP_RATE_LIMITER_BURST|Burst|int|10||
|IP_RATE_LIMITER_TTL|Retention time|duration|10m||
|IP_RATE_LIMITER_CLEANUP_INTERVAL|Cleanup interval|duration|1m||
