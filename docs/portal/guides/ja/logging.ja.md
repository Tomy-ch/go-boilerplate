# logging

[English](README.md) | 日本語

`internal/logging` はアプリケーション全体で利用する **構造化ロギング基盤** を提供します。

本パッケージは `zap` をベースにしながら、アプリケーションコードが **zap に直接依存しない形**でログを扱えるようにするための抽象化レイヤです。

主な目的は次の通りです。

- ログフォーマットの統一
- ログフィールドの安全な生成
- observability（trace/span）との統合
- テスト容易性の確保
- フレームワーク非依存のログAPI提供

## パッケージ構成

```txt
internal/logging
├── logger.go
├── logger_core.go
├── stacktrace_core.go
├── field.go
├── field_builder.go
├── const.go
├── test_kit.go
└── mock/
```

各ファイルの役割は次の通りです。

|ファイル|役割|
|---|---|
|`logger.go`|`Logger` interface、その `*logger` 実装、`WithCore`（追加の `LogCore` を Tee）|
|`logger_core.go`|zap ベースの Logger 構築（`NewJSONLogger` / `NewConsoleLogger`、エンコーダ設定）|
|`level.go`|`Level` 型と `LevelDebug/Info/Warn/Error` / `ParseLevel`|
|`stacktrace_core.go`|zap 自動付与の `Entry.Stack` を JSON 出力時に行配列へ変換する zapcore.Core ラッパ|
|`field.go`|ログフィールドの型とフィールド生成関数|
|`field_builder.go`|HTTP / SQL ログフィールド生成|
|`const.go`|ログキー定義|
|`test_kit.go`|テスト用 Logger / FieldBuilder|

## Logger インターフェース

アプリケーションコードは **Logger interface** のみを利用します。

```go
type Logger interface {
    Debug(msg string, fields ...*Field)
    Info(msg string, fields ...*Field)
    Warn(msg string, fields ...*Field)
    Error(msg string, fields ...*Field)

    Named(name string) Logger
    CallerSkip(skip int) Logger
}
```

`Named` は指定名を付与した子ロガーを返し、`CallerSkip` は caller 情報を指定フレーム数スキップして報告するロガーを返します（ラッパ越しにログ出力する場合に有用）。`*Field` から `zap.Field` への変換は非公開の `convertFields` が内部で行い、公開インターフェースには含まれません。

この設計により

- zap 依存を内部に閉じ込める
- テストでモック差し替え可能
- logging 実装を変更してもアプリ層に影響しない

というメリットがあります。

## Logger 生成

Logger は出力方式ごとに生成します。出力レベル（`Level`）と stacktrace を付与し始めるレベルを渡します。

```go
// JSON ロガー（機械可読・本番向け出力方式）
logger := logging.NewJSONLogger(logging.LevelInfo(), logging.LevelError())
// console ロガー（人間可読・開発向け出力方式）
logger := logging.NewConsoleLogger(logging.LevelDebug(), logging.LevelWarn())
```

`LevelDebug` / `LevelInfo` / `LevelWarn` / `LevelError` は対応する `Level` 値を返す関数です。

`Level` は zap のレベルを包む型で、利用側が `zapcore` に直接依存しないようにします。レベル文字列（`debug` / `info` / `warn` / `error`）は `ParseLevel` で変換します。

```go
level, err := logging.ParseLevel("info")
```

実行中のプロセスがどの出力方式・レベルを使うかは、本パッケージではなく DI の合成ルートで決まります。`internal/di/module/logging.go` の `provideLogger` が `APP_MODE` から出力方式を、`APP_LOG_LEVEL` から出力レベルを選択します。

|Mode|出力方式|Stacktrace|
|---|---|---|
|production|JSON logger|Error以上|
|development|console logger|Warn以上|

### 追加 Core の付与（Observability）

`LogCore` は `zapcore.Core` の型エイリアスです。`WithCore` は既存の `Logger` に追加の
core を Tee し（元 Logger と同じ最小レベルでゲート）、同じログエントリをその core からも
出力させます。

```go
logger = logging.WithCore(logger, extraCore)
```

`core` が `nil` の場合は元の `Logger` をそのまま返し、渡された `Logger` が本パッケージの
具象 `*logger` でない場合（テスト用 fake 等）もそのまま返します。

これは OpenTelemetry ログ送出との接続点です。`internal/observability` の `NewLogCore` が、
ログ送出が有効なときに zap ログを OTLP へ橋渡しする `otelzap` core を返します（無効時は `nil`）。
`internal/di/module/logging.go` の `provideLogger` が両者を `WithCore` で結線します。

## Field

ログフィールドは `Field` 型を利用して生成します。

```go
logger.Info(
    "user created",
    logging.String("user_id", "123"),
    logging.Int("age", 20),
)
```

サポートしている型

|関数|型|
|---|---|
|String|string|
|Strings|[]string|
|Int|int|
|Int64|int64|
|Float64|float64|
|Bool|bool|
|Time|time.Time（RFC3339Nano文字列に変換）|
|DurationMs|time.Duration（ミリ秒単位のfloat64に変換）|
|Error|error|
|Stacktrace|error（スタックトレースを行配列 []string に変換）|
|Any|any|

`Stacktrace` はスタックを `[]string`（1 行 1 要素）として保持するため、Grafana / Loki
などの JSON ビューアで改行が可読に表示されます。この分割を行うヘルパ
`SplitStackLines(s string) []string` は公開されており、他所でも再利用されます
（例: recovery ミドルウェアが生のランタイムスタックを `internal_stacktrace` フィールド用に分割）。

この設計の目的

- zap.Field を直接使わせない
- フィールド生成の安全性
- API の統一

## LogFieldBuilder

HTTP / SQL / Observability 用のログフィールド生成をまとめたコンポーネントです。

```go
type LogFieldBuilder interface {
    BuildHTTPRequestFields(req HTTPRequestLogInput) []*Field
    BuildHTTPResponseFields(resp HTTPResponseLogInput) []*Field
    BuildSQLEndFields(sql SQLFieldsEndInput) []*Field
}
```

trace / span フィールドは専用メソッドで生成せず、各 `Build*` メソッドが observability
有効時に自身の出力へ付与します（後述）。

生成

```go
lf := logging.NewLogFields(obsCfg, osCfg)
```

`config.ObservabilityConfig` と `config.OperatingSystemConfig` を受け取り、trace/span フィールドの付与やタイムゾーン情報の付与を制御します。

用途

- HTTPアクセスログ
- SQLログ
- trace/spanログ

などの **構造化ログを自動生成**します。

### 入力構造体

各 Build メソッドは専用の入力構造体を受け取ります。

|構造体|用途|主なフィールド|
|---|---|---|
|`HTTPRequestLogInput`|HTTPリクエストログ|EventType, Method, Path, URI, RemoteIP, Host, Scheme, Proto, UserAgent, ContentType, ContentLength, PathParams, QueryParams|
|`HTTPResponseLogInput`|HTTPレスポンスログ|Method, Path, URI, Status, Latency, RequestID|
|`SQLFieldsEndInput`|SQL終了ログ|Layer, PkgName, FuncName, SpanName, Latency, Query, Args, Err|

すべての入力構造体は `EventAt`（イベント発生時刻）と `TraceID` / `SpanID`（トレース情報）を持ちます。`ParentSpanID` は `SQLFieldsEndInput` のみに存在します。

## HTTP Logging

HTTPリクエスト / レスポンスログは次のフィールドを出力します。

例（Request）

- `event_type=start`
- `method=GET`
- `path=/v1/users`
- `remote_ip=...`
- `trace_id=...`
- `span_id=...`

例（Response）

- `event_type=end`
- `status=200`
- `latency_ms=12`
- `trace_id=...`
- `span_id=...`

## SQL Logging

SQLログは `BuildSQLEndFields` によりクエリの **終了**時点で出力されます。

### SQL End

- `event_type=end`
- `layer=repository`
- `package=...`
- `function=...`
- `span_name=FindUser`
- `latency_ms=4`
- `raw_query=SELECT ...`
- `query_compact=SELECT ...`
- `args_count=2`（引数がある場合のみ）
- `internal_error=...`（クエリが失敗した場合のみ）

クエリは `raw_query`（そのまま）と `query_compact`（改行・タブ・連続空白を 1 行に圧縮した形）の
2 種類が出力されます。

## Observability フィールド

専用の observability ビルダはありません。trace / span フィールドは、observability が有効で
かつ `TraceID` と `SpanID` の両方が存在するときに、HTTP / SQL ログ出力へ付与されます。

- `trace_id`
- `span_id`
- `parent_span_id`（SQL のみ、かつ親スパン ID が存在する場合のみ）

observability が無効な場合（または trace / span ID が空の場合）は出力されません。
`layer` / `package` / `function` は trace 付与ではなく SQL ログ出力（`SQLFieldsEndInput`）の
一部です。

## Test Kit

テストでは `NewTestLogger` を利用します。

```go
logger := logging.NewTestLogger(t)
```

特徴

- `zaptest.NewLogger`
- テストログを `testing.T` に出力
- 副作用なし

出力されたログ（レベル / 出力有無 / caller）を検証したい場合は、`*observer.ObservedLogs` を
`Logger` と併せて返す observed 版を使用します。

```go
logger, observed := logging.NewObservedTestLogger(t)
loggerWithCaller, observed := logging.NewObservedTestLoggerWithCaller(t)
```

LogFieldBuilder のテスト用インスタンス

```go
logging.NewTestLogFieldBuilder(t)
```

## 設計方針

この logging パッケージは次のポリシーで設計されています。

### 1 zap を直接使わせない

アプリケーションコードは `zap.Logger`, `zap.Field` に依存しません。

### 2 Field をラップする

ログフィールドは `Field` 型を利用します。

理由

- フィールド生成APIを固定
- zap 依存の隠蔽

### 3 Observability を統合

trace / span 情報は logging 層で統合します。

- `trace_id`
- `span_id`
- `parent_span_id`

### 4 テスト容易性

Logger は interface のため `mockgen` でモック生成可能です。

## ログキー定数

`const.go` で定義されるログキーの一覧です。

### HTTP系

|定数|キー|
|---|---|
|`EventTypeKey`|`event_type`|
|`EventTypeStart`|`start`|
|`EventTypeEnd`|`end`|
|`EventTypeError`|`error`|
|`EventTypePanic`|`panic`|
|`EventAtKey`|`event_at`|
|`EventTzKey`|`event_tz`|
|`StatusKey`|`status`|
|`MethodKey`|`method`|
|`URIKey`|`uri`|
|`PathKey`|`path`|
|`QueryParamsKey`|`query_params`|
|`PathParamsKey`|`path_params`|
|`UserAgentKey`|`user_agent`|
|`HostKey`|`host`|
|`SchemeKey`|`scheme`|
|`ProtoKey`|`proto`|
|`RemoteIPKey`|`remote_ip`|
|`ContentTypeKey`|`content_type`|
|`ContentLengthKey`|`content_length`|
|`LatencyKey`|`latency_ms`|
|`RequestIDKey`|`request_id`|

### エラー系

|定数|キー|
|---|---|
|`ErrorKey`|`error`|
|`OriginalErrorKey`|`original_error`|
|`ErrorCodeKey`|`error_code`|
|`ErrorMessageKey`|`error_message`|
|`ErrorDetailsKey`|`error_details`|
|`InternalErrorKey`|`internal_error`|
|`InternalStackTraceKey`|`internal_stacktrace`|

### クエリ系

|定数|キー|
|---|---|
|`RawQueryKey`|`raw_query`|
|`QueryCompactKey`|`query_compact`|
|`QueryArgsCountKey`|`args_count`|

### Job系

|定数|キー|
|---|---|
|`JobNameKey`|`job_name`|
|`JobArgsKey`|`job_args`|
|`JobErrorKey`|`job_error`|
|`JobResultKey`|`job_result`|
|`FilterKey`|`filter`|

### worker系

|定数|キー|
|---|---|
|`WorkerNameKey`|`worker_name`|
|`MessageIDKey`|`message_id`|
|`ReceiveCountKey`|`receive_count`|
|`PanicKey`|`panic`|

### 可観測系

|定数|キー|
|---|---|
|`TraceIDKey`|`trace_id`|
|`SpanIDKey`|`span_id`|
|`ParentSpanIDKey`|`parent_span_id`|
|`SpanNameKey`|`span_name`|
|`LayerKey`|`layer`|
|`PackageKey`|`package`|
|`FunctionKey`|`function`|

## セキュリティ注意点

ログには次の情報を **出力しないよう注意してください**

- パスワード
- 認証トークン
- 個人情報

必要な場合は **マスキング処理**を行ってください。
