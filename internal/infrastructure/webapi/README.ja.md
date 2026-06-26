# webapi

[English](README.md) | 日本語

`internal/infrastructure/webapi` は、**外部 Web API gateway の親サブシステム**です。各 leaf は `httpclient` の resilient な substrate 上で usecase の `boundary.Gateway` インターフェースを実装します。

## アーキテクチャ上の位置づけ

```mermaid
flowchart TB
    subgraph "Usecase 層（boundary）"
        IF["exchangerate.Gateway interface"]
    end
    subgraph "Infrastructure 層"
        Impl["webapi/exchangerate gateway"]
        Sub["httpclient.Client substrate"]
    end

    Impl -. implements .-> IF
    Impl --> Sub
```

`webapi/` 配下の各 leaf は `internal/usecase/boundary/<service>` で定義された意味的 gateway インターフェースを実装し、transport は `httpclient` substrate へ委譲します。Usecase / Domain は HTTP の詳細ではなく境界のみに依存します。leaf 実装は独自の README を持ちません（リポジトリの規約）。このサブシステム README が唯一の入り口です。

## 設計方針

- 外部サービスごとに 1 つの leaf パッケージを置き、各 leaf は usecase が定義した `boundary.Gateway` を実装し、境界の出力 DTO（生の HTTP / JSON 形ではない）を返す。
- 各 leaf は `httpclient` substrate をラップし、`DownstreamProfile` を登録する（論理 `Downstream` キーが profile / breaker / metrics / budget を駆動する）。
- 外部サービス向け profile は trace 伝搬を無効化し（`PropagateTrace = false`）、private/loopback 宛て接続を拒否する（`AllowPrivateNetwork = false`）。内部相関 ID の外部漏洩と内部ホストへの SSRF を防ぐ。
- エラーは substrate が写像済みの `apperror` sentinel として返す。JSON デコード / ドメイン形の検証失敗は `apperror.ErrUnavailable` でラップする。
- エンドポイントのベース URL は DI で注入する（サンプルは固定既定値を使用）。各 leaf は `observability.LayerTracer`（`tf.Infra()`）で span を開始する。

## DI 登録

`internal/di/module/webapi.go` の `webapi` モジュールに登録します。各 leaf がコンストラクタ / エンドポイントを provide し、`DownstreamProfile` を `httpclient_profiles` グループへ寄与します。

```go
fx.Module("webapi",
    fx.Provide(
        exchangerateext.NewEndpoint,
        exchangerateext.New,
    ),
    provideHTTPClientProfiles(
        exchangerateext.NewDownstreamProfile,
    ),
)
```
