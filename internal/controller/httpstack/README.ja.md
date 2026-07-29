# httpstack

[English](README.md) | 日本語

Echo サーバ起動時に登録する **HTTP 周りの共通ミドルウェア群**をまとめたディレクトリです。

各サブパッケージは小さな役割に分割されており、アプリケーションの起動処理で組み合わせて利用します。

## 役割

`httpstack` はアプリケーション全体で使われる Echo ミドルウェアおよびサーバー設定ヘルパーのカタログです。各サブパッケージは 1 つの関心事（リクエスト ID / ロギング / リカバリ / CORS / セキュリティヘッダ 等）を担い、薄い `Middleware(...)` または `New(...)` コンストラクタを公開します。ミドルウェアの **登録** は意図的に別の場所（`internal/di/server/extension`）で行い、本ディレクトリは fx や Echo インスタンス依存を含めない設計にしています。これにより各ユニットは独立して単体テスト可能・再利用可能になります。

## 設計方針

- 各機能は小さい単位で実装し、必要なものを選んで組み合わせる
- ミドルウェアは Echo の `e.Use(...)` で登録可能な形にラップ
- このディレクトリはミドルウェアの **実装のみ** を提供し、登録は `internal/di/server/extension` で行う

## サブパッケージ一覧

### ミドルウェア

|パッケージ|関数|説明|
|---|---|---|
|`requestid`|`Middleware`|X-Request-ID ヘッダの自動生成|
|`logging`|`Middleware`|HTTP リクエスト / レスポンスの構造化ログ|
|`recovery`|`Middleware`|パニックのキャッチとログ出力|
|`cors`|`Middleware`|CORS 設定|
|`security`|`Middleware`|セキュリティヘッダ（HSTS, X-Frame-Options 等）|
|`cookie`|`Middleware`|Set-Cookie ヘッダのセキュリティ属性強制|
|`forcejson`|`Middleware`|レスポンスの Content-Type を JSON に強制|
|`uri`|`Middleware`|末尾スラッシュの除去|
|`bodylimit`|`Middleware`|リクエストボディのサイズ上限（MB）、超過時 413|
|`timeout`|`Middleware`|per-request deadline budget（deadline 伝播の入口）|
|`observability`|`Middleware`|OpenTelemetry トレーシング統合|
|`redmetrics`|`Middleware`|HTTP RED メトリクス（request count / duration / status）。label は method / route / status_code / status_class のみ|
|`idempotency`|`Middleware` / `StrictMiddleware`|`Idempotency-Key` によるリクエスト冪等化の入口（oapi-codegen StrictMiddleware スロット。`e.Use` 登録ではない）|

### エラーハンドリング

|パッケージ|関数|説明|
|---|---|---|
|`errorhandler`|`New`|Echo / OpenAPI / apperror の統一エラーハンドラ|

### OpenAPI 統合

|パッケージ|関数|説明|
|---|---|---|
|`oapi`|`Middleware`|OpenAPI リクエストバリデーション|
|`oapi/auth`|`NewAuthenticator`|Cookie / Header からのトークン認証|
|`oapi/skipper`|`New`|ops エンドポイントのバリデーションスキップ|
|`oapi/validator`|`GetValidator`|OpenAPI スキーマ（spec）の読み込みと提供。バリデーション自体は `oapi` が担当|

### インフラ / ユーティリティ

|パッケージ|関数|説明|
|---|---|---|
|`basicauth`|`NewBasicAuthValidator`|メトリクスエンドポイント用 Basic 認証|
|`ipextractor`|`New`|環境に応じたクライアント IP 抽出|
|`ops`|`IsOpsPath`|運用系パス（/health, /metrics 等）の判定|

## ミドルウェア登録

ミドルウェアの登録は `internal/di/server/extension` で行います。

```go
// internal/di/server/extension での呼び出し例（概念）
func ConfigureHTTP(e *echo.Echo, cfg *config.ApplicationConfig, logger logging.Logger, lf logging.LogFieldBuilder) {
    e.Use(requestid.Middleware())
    e.Use(logging.Middleware(logger, lf))
    e.Use(recovery.Middleware(logger, lf, cfg))
    e.Use(cors.Middleware(cfg.SecurityConfig))
    e.Use(observability.Middleware(cfg))
}
```

`httpstack` 内で直接 Echo インスタンスへの登録を行わないでください。依存関係や初期化順序の問題が発生します。

## 環境による振る舞いの違い

|機能|Development|Production|
|---|---|---|
|IP 抽出|直接抽出|X-Forwarded-For + CIDR|
|リカバリスタック|10KB（フル）|4KB（制限）|

## テスト戦略

各サブパッケージは **単体のミドルウェアとして独立に**テストする。登録順序と組み上がったチェーンは `internal/di/server/extension` の担当であり、合成後のスタックは `internal/integration` の HTTP 境界テストで検証する — どちらもここで再テストしないこと。

### 実体を使う対象とモックにする対象

|依存|方法|
|---|---|
|`*echo.Echo` / ルータ / `*echo.Context`|実体（`echo.New()` + `httptest`）|
|後続ハンドラ（`next`）|受け取った内容を記録する / 固定エラーを返すテスト用クロージャ|
|`logging.Logger`|`logging.NewTestLogger` / `NewObservedTestLogger` — observed エントリ（メッセージ・フィールド）で検証し、整形済みログ文字列では検証しない|
|`config.*Config`|`config.MockConfigForTest` + `Set*(t, …)` セッタ|
|パッケージが宣言する協調オブジェクトのインターフェース（例: `redmetrics.Recorder`）|`*/mock/` の生成モック|

戻り値のエラーを検証するときはミドルウェアを直接駆動し（`Middleware(...)(next)(c)`）、実際に書き出されたレスポンス（status / ヘッダ / ボディ）を検証するときは `e.ServeHTTP` 経由で駆動する — レスポンスが commit されるのは実 Echo 経路のみ。oapi-codegen の `StrictMiddleware` スロットに入るミドルウェア（`idempotency`）は `e.Use` ではなくそのシグネチャ経由で駆動する。

### 全ミドルウェア共通で押さえる観点

- **素通し** — ミドルウェアが介入しないリクエストは変更されずに `next` へ届き、`next` の戻り値がそのまま伝播すること。
- **運用系パスの除外** — `ops.IsOpsPath` を参照するミドルウェア（`logging` / `redmetrics`、および `oapi/skipper` の skipper）は両側を検証する。運用系パス（`/health`・`/metrics` 等）ではログ／メトリクスが出ず、アプリケーションパスでは出ること。
- **`server.ResponseOf` の縮退** — Echo のレスポンスを取り出すミドルウェアは、ライタを unwrap できない場合に単なる素通しへ縮退する。`c.SetResponse(httptest.NewRecorder())` で再現し、失敗もせず何も記録しないことを検証する。この分岐は本番スタック経由では到達しないため、パッケージ単体テストだけが担保となる。
- **環境依存の分岐** — 設定によって振る舞いが切り替わる場合（`recovery` のスタックサイズ、`ipextractor` の抽出方式）は、config セッタで各モードを網羅する。不明モードのフォールバックも含めること。

### `Before` / `After` フックの観点

`server.ResponseOf(c).Before(...)` / `.After(...)` は遅延実行を登録するため、検証はミドルウェア呼び出しの戻り後ではなく、レスポンス書き出し後に置く。

- **発火条件** — `Before` は `WriteHeader` の直前に走り、ヘッダを補正できる最後の地点である（`forcejson`）。`After` は status 確定後の `Write` で走るため、エラーハンドラ／リカバリが最終的に決めた status を観測できる（`logging` / `redmetrics`）。
- **複数回の発火** — `After` は `Write` ごとに発火するため、チャンク／ストリーミング応答では複数回呼ばれる。1 リクエスト 1 回であるべき効果は、1 回に留まることを検証する（`redmetrics` は `sync.Once` で担保）。
- **発火しない経路** — ボディ無し応答（204 / 304）は `Write` が呼ばれないため `After` が走らない。それが観測欠落を招く場合、その欠落はフックの仕様上の限界として文書化し、暗黙にせずテストで固定する。

## 補足

- 新規ミドルウェアは独立したサブパッケージとして追加。1 パッケージに複数の関心事を詰め込まないこと
- 各ミドルウェアは順序に依存しない設計が原則だが、`internal/di/server/extension` での登録順序は `recovery`（他を包む位置で最外殻）と `requestid`（最先で実行し以降の log に ID が乗るように）だけは守る
- 本ディレクトリ内では `e.Use(...)` を直接呼ばないこと — 登録を `httpstack` の外に出すことで、`testkit/testecho` での再利用テストが可能になる
