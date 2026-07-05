---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, http, observability, security]
---

# ADR-0014: /metrics は認証例外 — OpenAPI 検証の外に置き、独立した BasicAuth ミドルウェアで保護する

English canonical: [0014-metrics-endpoint-auth-exception.md](../../adr/0014-metrics-endpoint-auth-exception.md)

## ステータス

accepted

## 背景

[ADR-0012](0012-spec-driven-request-validation.ja.md) は OpenAPI ミドルウェアがすべての受信リクエストを検証し、仕様内で宣言されたセキュリティ要件を強制することを確立している。Prometheus メトリクスエンドポイント（`GET /metrics`）は運用データを公開しており、未認証アクセスから保護されなければならない。しかし、`/metrics` は OpenAPI API 契約の一部ではないオペレーションパスである。パブリックリソースとしてバージョン管理されておらず、API クライアントによって消費されることもなく、そのレスポンスフォーマット（Prometheus テキスト形式）は OpenAPI 仕様に記述されていない。これを OpenAPI 検証パイプラインに通すには仕様に追加してハンドラーインターフェースを生成する必要があるが、それはアーキテクチャ上誤っている。

## 決定

`/metrics` を **OpenAPI 検証パイプラインの外に**登録し、専用の Echo `BasicAuth` ミドルウェアで保護する。

oapi ミドルウェアスキッパーは `/metrics` をオペレーションパスとして分類し、仕様検証を完全にバイパスする：

```go
// internal/controller/httpstack/oapi/skipper/skipper.go
func New() echomw.Skipper {
    return func(c echo.Context) bool {
        return ops.IsOpsPath(c.Request().URL.Path)
    }
}
```

ルートは `echomw.BasicAuth(validator)` をインラインで適用して登録される：

```go
// internal/controller/handler/metrics/metrics_handler.go
func BindHandler(e *echo.Echo, bav echomw.BasicAuthValidator) {
    e.GET("/metrics",
        echo.WrapHandler(promhttp.Handler()),
        echomw.BasicAuth(bav),
    )
}
```

BasicAuth バリデーターはタイミング攻撃に対抗するために定数時間比較を使用する（`internal/controller/httpstack/basicauth/basic.go`）。クレデンシャルは `MetricsConfig` から読み取られる。

OpenAPI 仕様内の `/metrics` への `security:` アノテーションは**ドキュメント目的のみ**——ランタイムの強制を駆動しない。実際の認証はルートに登録された BasicAuth ミドルウェアである。

## 影響

### ポジティブな影響

- `/metrics` は OpenAPI 契約に強制されることなく保護される。仕様はコンシューマーが依存する API リソースのみに制限される。
- メトリクスエンドポイントのクレデンシャルは `MetricsConfig` を通じて独立して設定可能であり、API エンドポイントに使用される JWT/BearerAuth とは分離されている。
- バリデーター内の定数時間比較がタイミング分析によるクレデンシャル漏洩を防ぐ。
- アプリレベルの BasicAuth とインフラレベルのアクセス制御は**排他的ではない**——両方を設定することが多層防御になる。片方が静かに壊れても（ネットワークルールの設定ミス、ゲートウェイポリシーの無効化）、もう片方がこのエンドポイントを守る。

### ネガティブな影響

- `/metrics` のセキュリティ機構は仕様の外に存在するため、OpenAPI ドキュメント単体では発見できない——開発者はハンドラー登録と `MetricsConfig` を確認することを知っていなければならない。
- OpenAPI 仕様が `/metrics` に `security:` アノテーションを含む場合、ランタイムでは静かに無効化される。読者は oapi ミドルウェアがそれを強制していると誤解する可能性がある。

## 検討した代替案

### /metrics を OpenAPI 仕様に含めてハンドラーインターフェースを生成する

セキュリティ強制を他のエンドポイントと一貫させることができる。却下：メトリクスエンドポイントは運用上の関心事であり API リソースではない——仕様に追加するとコンシューマー契約が汚染され、oapi-codegen がモデル化できない Prometheus レスポンスフォーマットの型を生成することが必要になる。

### /metrics をアプリケーションの Authenticator（Authn）経由にする

API エンドポイントと同じリクエスト単位の `Authenticator` を再利用する。却下：頻繁にスクレイプされる運用エンドポイントに対してリクエスト単位のフル認証はオーバーエンジニアリングであり、スクレイプのホットパスにレイテンシを加える。BasicAuth は設定・検証コストが低く運用エンドポイントに適する。トレードオフとして、運用者はメトリクスのユーザー名/パスワードを管理し、ソース管理外に保つ必要がある（`env/.env.*` に実際の値をコミットしない）。

### /metrics の認証をスキップしてネットワークレベルのアクセス制御に委ねる

運用上はシンプルになる。却下：テンプレートはすぐに使える認証機構を提供すべきであり、エンドポイントを開放したままにすることはセキュリティ責任を完全にインフラ設定に転嫁する。

## 補足

- スキッパー実装: [`internal/controller/httpstack/oapi/skipper/skipper.go`](../../../internal/controller/httpstack/oapi/skipper/skipper.go)。
- ルート登録と BasicAuth ワイヤリング: [`internal/controller/handler/metrics/metrics_handler.go`](../../../internal/controller/handler/metrics/metrics_handler.go)。
- バリデーター: [`internal/controller/httpstack/basicauth/basic.go`](../../../internal/controller/httpstack/basicauth/basic.go)。
- オペレーションパス分類: `internal/controller/httpstack/ops/paths.go`。
- [`openapi/README.md`](../../../openapi/README.md)（§ Security）のセキュリティ注記: `/metrics` の `security:` 宣言はドキュメント目的のみ。
- 関連する決定: [ADR-0012](0012-spec-driven-request-validation.ja.md)（仕様駆動リクエスト検証——この ADR はその付随する例外記録）。
- 親の決定: [ADR-0009](0009-openapi-first.ja.md)（OpenAPI ファースト）。
