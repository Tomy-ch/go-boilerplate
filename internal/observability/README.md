# internal/observability

概要: アプリケーションのトレース・ログ・メトリクスなどの観測（Observability）機能を提供するユーティリティ群です。

トレーサの生成、レイヤ別トレース、呼び出し元情報取得、テスト用の観測インスタンスなどのヘルパーをまとめています。

## 役割

- トレース生成と管理（トレーサファクトリ、プロバイダ）を行う。
- レイヤ単位のトレーシング（開始・終了・エラーハンドリング）を支援するユーティリティを提供する。
- 呼び出し元情報や補助的な観測用ヘルパーを提供し、ログやトレースの精度を高める。
- テスト用の観測インスタンス／初期化処理を提供して、統合テストでの観測挙動を安定させる。

## 必要度

### 本番運用での必須度

- 必須度: 本番運用で推奨

理由: トレーシングやロギングは運用時の障害解析やパフォーマンス解析に重要です。適切なトレーシングを入れることで、問題発生時の原因特定が容易になります。

### 開発/テスト運用での必須度

- 必須度: 開発/テスト運用で推奨

理由: 開発時にトレースや呼び出し情報があるとデバッグが容易になります。`test_kit.go` を使ってテスト環境向けに軽量化した観測を行えます。

## 利用例（簡易）

### 初期化

```go
// main.go 等の起動箇所で
ctx := context.Background()
tp, err := observability.NewProvider(observability.ProviderConfig{
    ServiceName: "my-service",
    Environment: "production",
    // OTLP エンドポイント等は環境変数で制御
})
if err != nil {
    // ハンドリング
}
defer tp.Shutdown(ctx)

tracer := observability.NewTracerFactory("my-service").Tracer("controller")
// tracer を使って span を開始

// layer_tracer を使った典型例
lt := observability.NewLayerTracer(tracer, "Usecase")
ctx, span := lt.Start(ctx, "HandlePurchase")
defer span.End()

// エラーが発生した場合
lt.RecordError(span, err)
```

### テストでの利用

```go
// internal/observability/test_kit.go を使うと、外部エクスポートを無効化した
// 軽量なプロバイダが取得できます。
nlt := observability.NewNoopLayerTracer()

// go test 実行例
// go test ./internal/observability -v
```

## 実装上の注意

- 外部エクスポーター（OTLP/Jaeger など）への接続失敗がアプリ起動を阻害しないよう、フェールセーフな設計にしてください。
- トレースに埋め込む属性に個人情報や機密情報を含めないでください。
- `caller.go` が提供する呼び出し元情報はパフォーマンスに影響を与える可能性があるため、必要最小限の利用に留めてください。

### 無効化した場合の影響

- トレース情報や呼び出し元情報が取得できなくなり、障害調査や性能分析の効率が低下します。
- テスト環境での観測関連の検証ができなくなり、回帰の検出が遅れる可能性があります。

## 注意点

- トレーサやログに出力する情報に機密データが含まれないよう注意してください（ユーザーデータ、トークン等）。
- 本番では適切なサンプルレートや出力先（OTLP、Jaeger 等）を設定し、コストやパフォーマンスへの影響を制御してください。
- 観測用のミドルウェアやヘルパーは副作用がないように実装し、エラーが発生してもアプリ本体の挙動に影響を与えないようにしてください。
- テスト用インスタンス（`test_kit.go`）はCI環境でのリソース制約を考慮して設定してください。
