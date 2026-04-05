# 環境変数一覧（対応表）

## OS

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OS_TZ|タイムゾーン設定|string|Asia/Tokyo|コンテナ / アプリの時刻基準|

## Application

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|APP_MODE|実行モード|string|development / production|ログや挙動切り替え|
|APP_NAME|アプリケーション名|string|Boilerplate|ログ・メトリクス識別|
|APP_ENV|環境識別子|string|local / staging / prod|環境区別用|
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown時間|duration|45s|SIGTERM時の待機時間|

## Server

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SERVER_HOST|バインドホスト|string|localhost|Dockerでは0.0.0.0推奨|
|SERVER_PORT|ポート番号|int|8080||
|SERVER_READ_HEADER_TIMEOUT|ヘッダ読み取りタイムアウト|duration|5s|Slowloris対策|
|SERVER_READ_TIMEOUT|リクエスト読み取りタイムアウト|duration|10s||
|SERVER_WRITE_TIMEOUT|レスポンス書き込みタイムアウト|duration|10s||
|SERVER_IDLE_TIMEOUT|KeepAliveタイムアウト|duration|60s||

## Metrics

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|METRICS_HOST|metrics bind host|string|0.0.0.0||
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic認証ユーザー|string|metrics-user|シークレット管理推奨（本番）|
|METRICS_PASSWORD|Basic認証パスワード|string|metrics-password|本番は必ず変更 / シークレット管理推奨|

## Observability

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OBSERVABILITY_ENABLED|トレース/ログ有効化|bool|true||
|OBSERVABILITY_MASKED_DB_QUERY_ARGS|DBパラメータマスク|bool|true|セキュリティ重要|
|OBSERVABILITY_TARGET_STATUS_CODES|トレース対象ステータス|csv|400,401,403,404,409,422,500,501,503|エラー監視用|

## Database

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|DB_DRIVER|DBドライバ|string|pgx|固定推奨|
|DB_HOST|DBホスト|string|database|docker service名（環境ごとに変更）|
|DB_PORT|DBポート|int|5432||
|DB_USER|ユーザー|string|postgres|シークレット管理推奨|
|DB_PASSWORD|パスワード|string|postgres-password|シークレット管理必須|
|DB_NAME|DB名|string|local|シークレット管理推奨|
|DB_SSL_MODE|SSL設定|string|disable|本番はrequire推奨|
|DB_PING_TIMEOUT|接続確認タイムアウト|duration|10s||
|DB_SLOW_QUERY_WARN_THRESHOLD|遅延クエリ警告閾値|duration|500ms|observability連携|

## Database Connection Pool

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|DBCONN_MAX_CONNS|最大接続数|int|10||
|DBCONN_MIN_CONNS|最小接続数|int|5||
|DBCONN_MAX_LIFETIME|接続寿命|duration|30m||
|DBCONN_MAX_IDLE_TIME|アイドル時間|duration|10m||

## Security

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SECURITY_ALLOWED_ORIGINS|CORS許可|csv|<http://localhost:3000,http://localhost:8000>||
|SECURITY_CIDR|許可IPレンジ|string|127.0.0.0/8||
|SECURITY_CONTENT_TYPE_NOSNIFF|X-Content-Type-Options|string|nosniff||
|SECURITY_X_FRAME_OPTIONS|clickjacking対策|string|DENY||
|SECURITY_HSTS_MAX_AGE|HSTS期間|duration|8760h||
|SECURITY_HSTS_EXCLUDE_SUBDOMAINS|サブドメイン除外|bool|false||
|SECURITY_HSTS_PRELOAD_ENABLED|preload有効|bool|false||
|SECURITY_REFERRER_POLICY|referrer制御|string|no-referrer||
|SECURITY_BCRYPT_COST|bcryptコスト|int|12||

## Cookie

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS限定|bool|true|本番必須|
|SECURE_COOKIE_SAME_SITE|SameSite設定|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookieドメイン|string|example.com||

## Auth

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|AUTH_COOKIE_NAME|認証Cookie名|string|auth_token||
|AUTH_HEADER_NAME|ヘッダ名|string|Authorization||
|AUTH_ALLOWED_HEADER_BEARER|Bearer許可|bool|true||

## IP Rate Limiter

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|IP_RATE_LIMITER_ENABLED|有効化|bool|true||
|IP_RATE_LIMITER_REQUESTS|許可リクエスト数|int|60||
|IP_RATE_LIMITER_PER|期間|duration|1m||
|IP_RATE_LIMITER_BURST|バースト|int|10||
|IP_RATE_LIMITER_TTL|保持時間|duration|10m||
|IP_RATE_LIMITER_CLEANUP_INTERVAL|クリーン間隔|duration|1m||
