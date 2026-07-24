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
- 備考に **Code default `<値>`** とあるものは `internal/config/envspec.go` の `default:` タグを持ち、`.env` ファイルには意図的に記載しない。boilerplate 派生プロジェクトが基本そのまま使うフレームワークレベルの定数で、既定値が自動適用される。プロジェクト側で上書きしたいときだけ該当 `.env` に明示エントリを追加する。それ以外の変数は `required` で、該当する env ファイルに必ず記載すること

## 新規変数を追加する手順

1. `internal/config/envspec.go` の対応する構造体にフィールドを追加（あわせて `model.go` の getter 構造体・`config.go` のマッピングも）
2. 該当サブシステムのテーブル（または新規サブシステム節）に変数を記載
3. 値の供給方法を決める:
   - **プロジェクト固有・環境ごとに変わる値** → フィールドを `required` にし、`env/.env`（ローカル既定）と各環境ファイル（`env/.env.<env>`）に追加
   - **普遍的なフレームワーク既定値** → 代わりに `default:` タグを付与し、`.env` ファイルには記載せず、テーブルに **Code default `<値>`** と明記
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
|APP_SHUTDOWN_TIMEOUT|Graceful shutdown時間|duration|65s|Code default `65s`。SIGTERM時の待機時間。HTTP サーバーでは `SERVER_REQUEST_TIMEOUT` 以上でなければならない（未満だとサーバー起動失敗）ため、drain が予算内のリクエストを打ち切ることはない|

### Server

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SERVER_HOST|バインドホスト|string|localhost|Dockerでは0.0.0.0推奨|
|SERVER_PORT|ポート番号|int|8080||
|SERVER_READ_HEADER_TIMEOUT|ヘッダ読み取りタイムアウト|duration|5s|Code default `5s`。Slowloris対策|
|SERVER_READ_TIMEOUT|リクエスト読み取りタイムアウト|duration|10s|Code default `10s`|
|SERVER_WRITE_TIMEOUT|レスポンス書き込みタイムアウト|duration|65s|Code default `65s`。SERVER_REQUEST_TIMEOUT 以上であること必須。短いと deadline budget より先に net/http が接続を切断し budget 制御が無効化される|
|SERVER_IDLE_TIMEOUT|KeepAliveタイムアウト|duration|60s|Code default `60s`|
|SERVER_BODY_LIMIT_MB|リクエストボディ上限（MB, 10進・1MB=1,000,000 byte）。超過時 413|int|5|Code default `5`。Pre middleware。OpenAPI 検証がボディを読む前に適用|
|SERVER_REQUEST_TIMEOUT|リクエスト全体の deadline budget（入口で1点設定し ctx で全層伝播）|duration|60s|Code default `60s`。停止/期限の単一軸。statement_timeout 等は backstop|

### Metrics

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|METRICS_HOST|metrics bind host|string|0.0.0.0||
|METRICS_PORT|metrics port|int|6060||
|METRICS_USERNAME|Basic認証ユーザー|string|metrics-user|シークレット管理必須 — ソース管理に入れない。local / ci のみ commit し、dev / stg / prd はデプロイ時に注入|
|METRICS_PASSWORD|Basic認証パスワード|string|metrics-password|シークレット管理必須 — ソース管理に入れない。local / ci のみ commit し、dev / stg / prd はデプロイ時に注入|

### Observability

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OBS_TRACES_EXPORTER|trace の OTLP exporter（`otlp` で有効化／空・`none` で無効）|string|otlp|空でトレース無効（軽量構成）|
|OBS_METRICS_EXPORTER|metric の OTLP exporter（`otlp` で有効化／空・`none` で無効）|string|otlp|空でメトリクス無効（軽量構成）|
|OBS_LOGS_EXPORTER|log の OTLP exporter（`otlp` で有効化／空・`none` で無効）|string|otlp|空でログ送出無効（zap は stdout のみ）|
|OBS_OTLP_ENDPOINT|OTLP 送出先エンドポイント URL|string|<http://observability:4318>|exporter 有効時に使用|
|OBS_OTLP_PROTOCOL|OTLP プロトコル（`http/protobuf` / `grpc`）|string|http/protobuf|Code default `http/protobuf`|
|OBS_MASKED_DB_QUERY_ARGS|DBパラメータマスク|bool|true|セキュリティ重要|
|OBS_TARGET_STATUS_CODES|トレース対象ステータス|csv|400,401,403,404,409,422,500,501,503|エラー監視用|

### Database

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|DB_DRIVER|DBドライバ|string|pgx|Code default `pgx`。固定推奨|
|DB_HOST|DBホスト|string|database|docker service名（環境ごとに変更）|
|DB_PORT|DBポート|int|5432||
|DB_USER|ユーザー|string|postgres|シークレット管理推奨|
|DB_PASSWORD|パスワード|string|postgres-password|シークレット管理必須|
|DB_NAME|DB名|string|local|シークレット管理推奨|
|DB_SSL_MODE|SSL設定|string|disable|本番はrequire推奨|
|DB_PING_TIMEOUT|接続確認タイムアウト|duration|10s||
|DB_SLOW_QUERY_WARN_THRESHOLD|遅延クエリ警告閾値|duration|500ms|Code default `500ms`。observability連携|
|DB_STATEMENT_TIMEOUT|SQL 文ごとの実行時間上限（`statement_timeout`）|duration|30s|Code default `30s`。ctx を無視する query への SQL 層 backstop。0 で無効|
|DB_LOCK_TIMEOUT|ロック獲得待ちの上限（`lock_timeout`）|duration|10s|Code default `10s`。長時間ロック待ちへの backstop。0 で無効|
|DB_TX_MAX_RETRIES|serialization failure / deadlock 時の tx リトライ最大試行回数|int|3|Code default `3`。0 でリトライ無効（単発実行）|
|DB_TX_RETRY_BASE_BACKOFF|tx リトライ backoff の初期値|duration|5ms|Code default `5ms`。指数 backoff の基準値（×2）|
|DB_TX_RETRY_MAX_BACKOFF|tx リトライ backoff の上限値|duration|100ms|Code default `100ms`。1 試行あたりの上限|

### Database Connection Pool

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|DBCONN_MAX_CONNS|最大接続数|int|10|Code default `10`|
|DBCONN_MIN_CONNS|最小接続数|int|5|Code default `5`|
|DBCONN_MAX_LIFETIME|接続寿命|duration|30m|Code default `30m`|
|DBCONN_MAX_IDLE_TIME|アイドル時間|duration|10m|Code default `10m`|

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

### Cookie

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|SECURE_COOKIE_SECURE|HTTPS限定|bool|true|本番必須|
|SECURE_COOKIE_SAME_SITE|SameSite設定|string|Strict||
|SECURE_COOKIE_DOMAIN|Cookieドメイン|string|example.com||

### Worker

worker engine の engine-core 設定（broker 非依存）。

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|WORKER_CONCURRENCY|同時に Handle を実行する最大数|int|4|Code default `4`|
|WORKER_MAX_IN_FLIGHT|受信済み・未確定の最大メッセージ数|int|8|Code default `8`|
|WORKER_BATCH_SIZE|1 回の Receive で取得する最大件数|int|4|Code default `4`|
|WORKER_EXTEND_INTERVAL|Extend を呼ぶ周期（`0` 以下で無効）|duration|0s|Code default `0s`|
|WORKER_DRAIN_TIMEOUT|停止時に in-flight の完了を待つ上限|duration|30s|Code default `30s`|
|WORKER_RECEIVE_COUNT_WARN_THRESHOLD|再配送回数の警告閾値（`0` 以下で無効）|int|5|Code default `5`|
|WORKER_CIRCUIT_FAILURE_THRESHOLD|サーキットを Open にする連続失敗数（`0` 以下で無効）|int|10|Code default `10`|
|WORKER_CIRCUIT_OPEN_BACKOFF_INITIAL|Open の初回 cooldown|duration|1s|Code default `1s`|
|WORKER_CIRCUIT_OPEN_BACKOFF_MAX|Open の cooldown 上限|duration|30s|Code default `30s`|
|WORKER_CIRCUIT_HALF_OPEN_PROBE|Half-open 時に試行する最大件数|int|1|Code default `1`|
|WORKER_HEALTH_LISTEN_ADDR|liveness/readiness を公開する health listener の待ち受けアドレス|string|:8081|Code default `:8081`|
|WORKER_PROGRESS_STALE_AFTER|readiness 判定で「進捗なし」とみなすまでの時間|duration|60s|Code default `60s`|
|WORKER_NACK_BACKOFF_INITIAL|retryable 失敗時の per-message 再配送 backoff の初回待機|duration|1s|Code default `1s`|
|WORKER_NACK_BACKOFF_MAX|per-message 再配送 backoff の上限|duration|30s|Code default `30s`|

### Outbox

transactional outbox relay の設定。

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OUTBOX_ENDPOINT|メッセージの送信先エンドポイント URL|string||Code default は空。relay の送信先が固定のプロジェクトで設定する|
|OUTBOX_POLL_INTERVAL|pending を捌き切った後、次 poll まで待機する時間|duration|1s|Code default `1s`|
|OUTBOX_ERROR_BACKOFF|relay バッチがエラーを返した後に待機する時間|duration|5s|Code default `5s`|
|OUTBOX_BATCH_SIZE|1 回の poll で claim する pending 行数|int|100|Code default `100`|

### Auth (JWT)

access token（JWT）検証の設定。CI / test は署名検証なしのスタブを配線し、`local` / `development` は JWKS backed の実 JWT authenticator を配線する（ローカル開発は mock 認証サーバーを検証）。後者は `AUTH_ISSUER` / `AUTH_AUDIENCE` が欠けていると起動時に fail-closed になる。配線判断は DI（`internal/di/module/core/auth.go`）が担う。`AUTH_JWKS_URL` は override で、空の場合は `AUTH_ISSUER` から OIDC discovery 経由で `jwks_uri` を導出する（issuer 厳密一致 + https + 同一オリジン。`local` は mock provider への平文 http を許容）。JWKS / discovery 取得は `httpclient` substrate を通すため、HTTP タイムアウト / リトライ / circuit breaker / budget は `jwks` downstream プロファイル（`NewDownstreamProfile`）由来で、env 変数では持たない。同プロファイルは `local` / `ci` / `test` 以外では private 網宛て SSRF も遮断する。

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|AUTH_ISSUER|検証する `iss` クレームの期待値（OIDC issuer 兼用）|string||Code default は空。JWT authenticator を配線する環境で設定する|
|AUTH_AUDIENCE|検証する `aud` クレームの期待値|string||Code default は空。issuer と対で必須|
|AUTH_JWKS_URL|JWKS エンドポイント URL の override。空の場合は `AUTH_ISSUER` から OIDC discovery で `jwks_uri` を導出|string||Code default は空。compose では内部サービス URL（例 `http://mock_auth_server:4000/.well-known/jwks.json`）|
|AUTH_ALLOWED_ALGORITHMS|許可する署名アルゴリズムの allowlist（カンマ区切り・非対称のみ）|[]string|RS256|Code default `RS256`。`none` / 対称鍵は常に拒否|
|AUTH_CLOCK_SKEW|`exp` / `nbf` 検証のクロックずれ許容幅|duration|60s|Code default `60s`|
|AUTH_JWKS_CACHE_TTL|取得した JWKS をキャッシュする期間|duration|1h|Code default `1h`|
|AUTH_JWKS_DISCOVERY_TTL|OIDC discovery 文書をキャッシュする期間（鍵キャッシュとは別軸）|duration|24h|Code default `24h`。jwks_uri を discovery で導出する場合のみ使用|
|AUTH_JWKS_UNKNOWN_KID_COOLDOWN|未知 `kid` での JWKS 再取得の最小間隔（DoS 抑止）|duration|60s|Code default `60s`|

### Object Storage

アップロード資産（商品画像）用の S3 互換オブジェクトストレージ。usecase は vendor 中立の `objectstorage.Storage` 境界に依存し、infrastructure 実装は S3 アダプタ（AWS SDK v2 S3）。`local` は Garage コンテナへ接続し、deploy 環境は `OBJECT_STORAGE_ENDPOINT` を空にして AWS S3 を対象にする。アダプタは S3 だが env 名は vendor 中立に保つ。値は環境ごとに宣言（code default を持たない）し、資格情報はデプロイ時に注入する。

|変数名|説明|型|例|備考|
|---|---|---|---|---|
|OBJECT_STORAGE_ENDPOINT|S3 互換エンドポイント URL。空は SDK 既定解決（AWS S3）|string|http://garage:3900|`required`（空を許容）。`local` は Garage の compose サービス、deploy は空|
|OBJECT_STORAGE_REGION|署名リージョン|string|us-east-1|`required,notEmpty`|
|OBJECT_STORAGE_BUCKET|オブジェクト格納先バケット|string|gobp-local|`required,notEmpty`|
|OBJECT_STORAGE_ACCESS_KEY_ID|静的資格情報のアクセスキー ID|string|gobp-local-access-key|`required,notEmpty`。デプロイ時に注入|
|OBJECT_STORAGE_SECRET_ACCESS_KEY|静的資格情報のシークレットアクセスキー|string|gobp-local-secret-key|`required,notEmpty`。デプロイ時に注入|
|OBJECT_STORAGE_USE_PATH_STYLE|path-style アドレッシング（Garage / MinIO は true、AWS S3 は false）|bool|true|`required`|
|OBJECT_STORAGE_MAX_UPLOAD_BYTES|受理する最大アップロードサイズ（バイト）|int|5242880|`required,notEmpty`。サンプルは 5 MiB|

## 補足

- 例欄の値はローカル開発向け。本番では Secret / CIDR / Cookie ドメイン / origin 等は基本的に別の値になります
- `csv` 型は `,` 区切りで空白トリム後に分割。値そのものに `,` を含めないこと
- `duration` 型は Go `time.ParseDuration` 構文（`500ms`, `1h30m`）。素の数値は不可
- 新規サブシステム節を作る際もテーブル列構成（`変数名 | 説明 | 型 | 例 | 備考`）を維持してスキャン性を保つこと
- `APP_LOG_LEVEL` は環境ごとに明示指定する。local / ci / dev と **staging** は `debug`（本番前診断のための詳細 JSON ログ）、production は `info`。出力方式（JSON / console）はレベルとは独立に `APP_MODE` が決定する
- env ファイルはビルド時にバイナリへ埋め込まれる（`embed.go`）。`env/.env` がローカル既定かつ唯一の埋め込み対象で、`env/.env.<env>` は各環境のソース。Docker の `builder` ステージが `APP_ENV` ビルド引数で対象を材料化する（`go build` 前に `cp env/.env.${APP_ENV} env/.env`）。Docker 以外（`go run` / `go test`）はコミット済みの local `env/.env` を埋め込むため、別環境が必要な CI は同様に焼き直す（例: `cp env/.env.ci env/.env`）。実行時の環境変数は埋め込み値より優先される
