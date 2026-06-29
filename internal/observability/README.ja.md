# internal/observability

[English](README.md) | 日本語

`internal/observability` は、本プロジェクトの **トレーシング（Tracing）および観測ログ連携**を提供するパッケージです。

このパッケージは **OpenTelemetry をベースとしたトレーシング機構**と、  
`internal/logging` パッケージと連携した **レイヤー単位の可観測性ログ**を提供します。

主な目的：

- OpenTelemetry の初期化と管理
- レイヤー単位の span 生成
- trace / span 情報のログ出力
- Domain / Usecase / Controller の観測統一
- テスト時の軽量トレーサ提供

## アーキテクチャ

observability パッケージは次の構成で設計されています。

```mermaid
flowchart TB

subgraph Observability
    TracerProvider/OTel
    TracerFactory
    LayerTracer
end

TracerProvider/OTel --> TracerFactory
TracerFactory --> LayerTracer
LayerTracer --> ApplicationCode
```

各コンポーネントの役割：

|コンポーネント|役割|
|---|---|
|`TracerProvider`|OpenTelemetry のトレーサープロバイダ|
|`TracerFactory`|レイヤー別トレーサ生成|
|`LayerTracer`|span生成 + observabilityログ|
|`helper.go`|span / trace helper, ShouldLogWithSpan, BuildSpanName|
|`caller.go`|呼び出し元関数名取得|
|`test_kit.go`|テスト用 tracer|

## 提供機能

### 1. TracerProvider

OpenTelemetry のトレーサープロバイダを初期化します。

```go
func TracerProvider(reg lifecycle.Registrar) trace.TracerProvider
```

特徴

- OpenTelemetry TracerProvider を生成
- `otel.SetTracerProvider` に登録
- アプリ終了時に `Shutdown()` を実行

アプリケーションの DI 初期化で利用されます。

### 2. TracerFactory

各レイヤー用の `LayerTracer` を生成するファクトリです。

```go
type TracerFactory interface {
    Controller() LayerTracer
    Usecase() LayerTracer
    Infra() LayerTracer
}
```

この設計により

- Controller
- Usecase
- Infrastructure

ごとに **span namespace を分離**できます。

生成例

```go
tf := observability.NewTracerFactory(tp, logger, logFieldBuilder)

controllerTracer := tf.Controller()
usecaseTracer := tf.Usecase()
infraTracer := tf.Infra()
```

### 3. LayerTracer

`LayerTracer` は **レイヤー単位の span 管理**を行うコンポーネントです。

主な機能

- span生成
- span開始 / 終了ログ
- traceID / spanID ログ出力
- span名の自動生成

#### Start

```go
ctx, end := tracer.Start(ctx)
defer end()
```

span名は `layer.package.function` のルールで自動生成されます。

例

- `usecase.user.CreateUser`
- `controller.user.GetUsers`
- `infrastructure.user.FindByID`

#### StartWithSuffix

span名に追加の接尾辞を付与して span を開始します。

```go
ctx, end := tracer.StartWithSuffix(ctx, "detail")
defer end()
```

生成される span名: `usecase.user.CreateUser.detail`

同一関数内で複数の span を区別したい場合に使用します。

### 4. Span Helper (RunWithSpan)

任意の処理は `RunWithSpan` で簡単に span 計測できます。

この関数はレイヤに依存せず、任意の処理を span + observability logging とともに実行するためのユーティリティです。

```go
ctx, result, err := observability.RunWithSpan(
    ctx,
    tracer,
    observability.Usecase,
    "user",
    "FullName",
    func(ctx context.Context) (string, error) {
        return user.FullName(), nil
    },
)
```

この関数を使うことで、以下を自動で処理します。

- span開始
- span終了
- observabilityログ出力

### 5. ShouldLogWithSpan

o11yモードが有効かつ、現在の Context に有効な Span が存在するかを判定します。

```go
if observability.ShouldLogWithSpan(ctx, obsCfg) {
    // span 前提のログ出力
}
```

`config.ObservabilityConfig` の `Enabled()` と、Context 内の Span の有効性を組み合わせて判定します。

### 6. BuildSpanName

レイヤー名・パッケージ名・関数名からスパン名を構築するヘルパーです。

```go
name := observability.BuildSpanName("usecase", "user", "CreateUser")
// => "usecase.user.CreateUser"
```

### 7. Span Event 定数

span のライフサイクルイベントを表す定数です。

```go
const (
    SpanEventStart = "start"
    SpanEventEnd   = "end"
)
```

ログ出力時の `event_type` フィールドに使用されます。

## Span Logging

span開始 / 終了時には **structured logging** が出力されます。

Start

- `event_type=start`
- `span_name=usecase.user.CreateUser`
- `trace_id=...`
- `span_id=...`

End

- `event_type=end`
- `latency=12ms`
- `trace_id=...`
- `span_id=...`

ログ出力は `internal/logging` の `LogFieldBuilder` を使用します。

## TraceContext

TraceContext は span の識別情報を保持します。

```go
type TraceContext struct {
    traceID
    spanID
    parentSpanID
}
```

取得

```go
tc := observability.ExtractTraceContext(ctx)
```

利用例

```go
tc.TraceID()
tc.SpanID()
tc.ParentSpanID()
```

## Span Helper

### StartSpanWithParent

親 span を引き継いだ子 span を生成します。

```go
tc, ctx, end := observability.StartSpanWithParent(
    ctx,
    tracer,
    "usecase.user.CreateUser",
)
defer end()
```

返却値

|値|説明|
|---|---|
|TraceContext|trace/span 情報|
|context|child context|
|func()|span終了|

## Caller Helper

`caller.go` は **呼び出し元関数名を取得**します。

```go
getCallerFullName()
```

この情報は

- span名生成
- observabilityログ

で使用されます。

## テストサポート

テストでは **Noop tracer** を利用します。

### TracerFactory

```go
tf := observability.NewNoopTracerFactory(t)
```

### LayerTracer

```go
lt := observability.NewMockUsecaseLayerTracer(t)
```

利用可能なテストトレーサ

|関数|説明|
|---|---|
|`NewMockControllerLayerTracer`|Controller用|
|`NewMockUsecaseLayerTracer`|Usecase用|
|`NewMockInfraLayerTracer`|Infra用|
|`NewNoopLayerTracer`|汎用|
|`NewStubSpanContext`|有効な Span を持つ Context 生成|

### StubSpanContext

テストで有効な `trace.Span` を含む Context が必要な場合に使用します。

```go
ctx, cleanup := observability.NewStubSpanContext(t)
defer cleanup()
```

実際の `sdktrace.TracerProvider` を使用して有効な span を生成するため、`ShouldLogWithSpan` のテスト等で利用できます。

## 設計ポリシー

このパッケージは次の設計ポリシーに基づいています。

### 1 Layer単位のトレーシング

span名は必ず `layer.package.function` 形式になります。

理由

- trace可読性向上
- service map 分析容易

### 2 logging と統合

spanイベントは logging パッケージを通じて出力します。

- `trace_id`
- `span_id`
- `parent_span_id`
- `layer`
- `pkg`
- `function`

### 3 アプリケーションコードはOTelに依存しない

アプリケーションコードは `LayerTracer` のみ利用します。

OpenTelemetry SDK は **observability パッケージ内に閉じ込めます。**

### 4 フェールセーフ

observability 機能が失敗しても

- アプリケーション処理
- ビジネスロジック

に影響を与えません。

## メトリクス

トレーシングに加えて、本パッケージはプロセスレベルのメトリクスを公開します。

### ビルド情報 (`app_build_info`)

`internal/observability/metrics/buildinfo` サブパッケージは、アプリケーションのビルド・バージョン・ランタイム情報を Prometheus の info gauge (`app_build_info`、値は常に `1`) として公開します。`/version` エンドポイントと同一の source of truth (`system.BuildInfo`) を用い、全ラベル値を DI 結線時に一度だけ解決します。

詳細は `internal/observability/metrics/buildinfo/README.ja.md` を参照してください。

## セキュリティ注意点

トレース情報には次を含めないでください。

- パスワード
- トークン
- 個人情報
- 秘密鍵

必要な場合は **マスキング処理**を行います。
