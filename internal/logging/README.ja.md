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
├── field.go
├── field_builder.go
├── const.go
├── test_kit.go
└── mock/
```

各ファイルの役割は次の通りです。

|ファイル|役割|
|---|---|
|`logger.go`|アプリケーションが利用する Logger interface|
|`logger_core.go`|zap.Logger の実装|
|`field.go`|ログフィールドの型|
|`field_builder.go`|HTTP / SQL / Observability ログフィールド生成|
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
    ConvertFields(fields []*Field) []zap.Field
}
```

`ConvertFields` は `*Field` スライスを `zap.Field` スライスに変換します。主にフレームワーク連携など、内部的に zap.Field が必要な場面で使用します。

この設計により

- zap 依存を内部に閉じ込める
- テストでモック差し替え可能
- logging 実装を変更してもアプリ層に影響しない

というメリットがあります。

## Logger 生成

Logger はアプリケーションの実行モードに応じて生成されます。

```go
logger, err := logging.New(appCfg)
```

`New` は `config.ApplicationConfig` のモードに応じて適切なロガーを選択します。

個別に生成する場合は以下の関数も利用できます。

```go
logger, err := logging.NewProductionLogger()
logger, err := logging.NewDevelopmentLogger()
```

内部では次のロガーが使用されます。

|Mode|Logger|
|---|---|
|production|JSON logger|
|development|console logger|

### Production Logger

- Encoding: JSON
- Level: Info
- Stacktrace: Error以上

### Development Logger

- Encoding: Console
- Level: Debug
- Stacktrace: Warn以上

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
|Stacktrace|error（スタックトレース文字列に変換）|
|Any|any|

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
    BuildSQLStartFields(sql SQLFieldsStartInput) []*Field
    BuildSQLEndFields(sql SQLFieldsEndInput) []*Field
    BuildObservabilityFields(obs ObservabilityFieldsInput) []*Field
}
```

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
|`HTTPRequestLogInput`|HTTPリクエストログ|Method, Path, URI, RemoteIP, Host, Scheme, Proto, UserAgent, ContentType, ContentLength, PathParams, QueryParams|
|`HTTPResponseLogInput`|HTTPレスポンスログ|Method, Path, URI, Status, Latency, RequestID|
|`SQLFieldsStartInput`|SQL開始ログ|Layer, PkgName, FuncName, SpanName|
|`SQLFieldsEndInput`|SQL終了ログ|Layer, PkgName, FuncName, SpanName, Latency, Query, Args, Err|
|`ObservabilityFieldsInput`|Observabilityログ|Layer, PkgName, FuncName, SpanName, EventType, Latency|

すべての入力構造体は `EventAt`（イベント発生時刻）と `TraceID` / `SpanID` / `ParentSpanID`（トレース情報）を共通で持ちます。

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

SQLログは **開始 / 終了**の2イベントで出力されます。

### SQL Start

- `event_type=start`
- `layer=repository`
- `span_name=FindUser`

### SQL End

- `event_type=end`
- `latency_ms=4`
- `query=SELECT ...`
- `args=[...]`

クエリは

- `raw_query`
- `query_compact`

の2種類が出力されます。

## Observability Logging

Observabilityログは trace/span 情報を含みます。

- `trace_id`
- `span_id`
- `parent_span_id`
- `layer`
- `package`
- `function`

Observabilityが無効な場合は出力されません。

## Test Kit

テストでは `NewTestLogger` を利用します。

```go
logger := logging.NewTestLogger(t)
```

特徴

- `zaptest.NewLogger`
- テストログを `testing.T` に出力
- 副作用なし

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
|`ErrorCodeKey`|`error_code`|
|`ErrorMessageKey`|`error_message`|
|`ErrorDetails`|`error_details`|
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
