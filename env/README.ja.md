# 環境変数一覧（対応表）

[English](README.md) | 日本語

このディレクトリは、アプリケーションが読み込むすべての環境変数の正規リファレンスです。各変数は `internal/config/` 配下の型付き Go 構造体にロードされ、本 README ではサブシステム別（OS / Application / Server / Database / Security / …）にグルーピングしています。新規変数の追加、各サービスが読む変数の棚卸し、オンボーディング資料として活用してください。

## 命名・型の規約

- 変数名は `{SUBSYSTEM}_{NAME}` の UPPER_SNAKE_CASE
- 型欄は `internal/config/` で読み込まれる Go 型に対応:
  - `string` → `string`、`int` → `int`、`bool` → `bool`
  - `duration` → `time.Duration`（`time.ParseDuration` でパース、例: `500ms`, `1h30m`）
  - `csv` → `[]string`（`,` 区切り、空白トリム後分割）
- 備考に **Secret management required** とあるものは本番で **必ずシークレットマネージャーから取得**。`.env` に平文で含めない
- **Secret management recommended** は定期ローテーションを推奨

## 新規変数を追加する手順

1. `internal/config/` の対応する構造体にフィールドを追加
2. 該当サブシステムのテーブル（または新規サブシステム節）に変数を記載
3. `env/.env`（ローカル既定）と各環境ファイル（`env/.env.<env>`）に変数を追加
4. `make test` を実行して config 構造体がロードできることを確認

## 変数一覧（サブシステム別）

### OS

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OS_TZ|タイムゾーン設定|string|Asia/Tokyo|コンテナ / アプリの時刻基準|

### Application

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|APP_MODE|実行モード|string|development / production|ログや挙動切り替え|
|APP_LOG_LEVEL|ログ出力レベル|string|debug / info / warn / error|出力方式は Mode が決定、レベルは環境ごとに明示指定|
|APP_NAME|アプリケーション名|string|Boilerplate|ログ・メトリクス識別|
|APP_ENV|環境識別子|string|local / staging / prod|環境区別用|
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown時間|duration|45s|SIGTERM時の待機時間|

### Server

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SERVER_HOST|バインドホスト|string|localhost|Dockerでは0.0.0.0推奨|
|SERVER_PORT|ポート番号|int|8080||
|SERVER_READ_HEADER_TIMEOUT|ヘッダ読み取りタイムアウト|duration|5s|Slowloris対策|
|SERVER_READ_TIMEOUT|リクエスト読み取りタイムアウト|duration|10s||
|SERVER_WRITE_TIMEOUT|レスポンス書き込みタイムアウト|duration|10s||
|SERVER_IDLE_TIMEOUT|KeepAliveタイムアウト|duration|60s||

### Metrics

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|METRICS_HOST|metrics bind host|string|0.0.0.0||
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic認証ユーザー|string|metrics-user|シークレット管理推奨（本番）|
|METRICS_PASSWORD|Basic認証パスワード|string|metrics-password|本番は必ず変更 / シークレット管理推奨|

### Observability

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OBSERVABILITY_ENABLED|トレース/ログ有効化|bool|true||
|OBSERVABILITY_MASKED_DB_QUERY_ARGS|DBパラメータマスク|bool|true|セキュリティ重要|
|OBSERVABILITY_TARGET_STATUS_CODES|トレース対象ステータス|csv|400,401,403,404,409,422,500,501,503|エラー監視用|

### Database

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

### Database Connection Pool

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|DBCONN_MAX_CONNS|最大接続数|int|10||
|DBCONN_MIN_CONNS|最小接続数|int|5||
|DBCONN_MAX_LIFETIME|接続寿命|duration|30m||
|DBCONN_MAX_IDLE_TIME|アイドル時間|duration|10m||

### Security

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

### Cookie

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS限定|bool|true|本番必須|
|SECURE_COOKIE_SAME_SITE|SameSite設定|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookieドメイン|string|example.com||

### Auth

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|AUTH_COOKIE_NAME|認証Cookie名|string|auth_token||
|AUTH_HEADER_NAME|ヘッダ名|string|Authorization||
|AUTH_ALLOWED_HEADER_BEARER|Bearer許可|bool|true||

## 補足

- 例欄の値はローカル開発向け。本番では Secret / CIDR / Cookie ドメイン / origin 等は基本的に別の値になります
- `csv` 型は `,` 区切りで空白トリム後に分割。値そのものに `,` を含めないこと
- `duration` 型は Go `time.ParseDuration` 構文（`500ms`, `1h30m`）。素の数値は不可
- 新規サブシステム節を作る際もテーブル列構成（`変数名 | 説明 | 型 | 例 | 備考`）を維持してスキャン性を保つこと
- `APP_LOG_LEVEL` は環境ごとに明示指定する。local / ci / dev と **staging** は `debug`（本番前診断のための詳細 JSON ログ）、production は `info`。出力方式（JSON / console）はレベルとは独立に `APP_MODE` が決定する
- env ファイルはビルド時にバイナリへ埋め込まれる（`embed.go`）。`env/.env` がローカル既定かつ唯一の埋め込み対象で、`env/.env.<env>` は各環境のソース。Docker の `builder` ステージが `APP_ENV` ビルド引数で対象を材料化する（`go build` 前に `cp env/.env.${APP_ENV} env/.env`）。Docker 以外（`go run` / `go test`）はコミット済みの local `env/.env` を埋め込むため、別環境が必要な CI は同様に焼き直す（例: `cp env/.env.ci env/.env`）。実行時の環境変数は埋め込み値より優先される
