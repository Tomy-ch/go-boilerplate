---
status: accepted
date: 2026-07-04
deciders: [maintainers]
tags: [contract, openapi, security]
---

# ADR-0012: リクエスト検証と認証を実行時に仕様から強制する。レスポンスは検証しない

English canonical: [0012-spec-driven-request-validation.md](../../adr/0012-spec-driven-request-validation.md)

## ステータス

accepted

## 背景

[ADR-0009](0009-openapi-first.ja.md) は OpenAPI 仕様をワイヤー契約の唯一の真実のソースとしている。OpenAPI の定義はバックエンドとフロントエンドや他の API との「契約」であるため、これを最優先とし、契約違反がビジネスロジックに到達しないよう処理の中でエラーが出ないように制御することを選んだ。仕様が単にドキュメント化するだけでなく実際にランタイムでサーバーを保護するためには、リクエスト検証とセキュリティスキームの強制が、同一の仕様ドキュメントから自動的にすべての受信リクエストに対して実行されなければならない。同時に、コード生成によって仕様とコードが同期した状態に保たれる場合（[ADR-0011](0011-oapi-codegen-strict-server.ja.md) 参照）、送信レスポンスを実行時に仕様に対して検証することはコストが高く、かつ不要である。

追加の制約として、オペレーションパス（`/health`、`/metrics`、`/ready`、`/healthz`、`/version`）は OpenAPI 仕様に記述されていないため、OpenAPI 検証パイプラインを通過させてはならない。

## 決定

`oapi-codegen/echo-middleware` パッケージの `oapimw.OapiRequestValidatorWithOptions` を Echo ミドルウェアとしてワイヤリングし、解析済みの仕様と `openapi3filter.AuthenticationFunc` を渡す。バリデーターが実行される前に、authn コンテキストスロットを注入して `AuthenticationFunc` がリクエストコンテキストに認証結果を書き込めるようにする。

```go
func Middleware(
    spec *openapi3.T,
    skipper echomw.Skipper,
    authFunc openapi3filter.AuthenticationFunc,
) echo.MiddlewareFunc {
    oapiValidator := oapimw.OapiRequestValidatorWithOptions(spec, &oapimw.Options{
        SilenceServersWarning: true,
        Skipper:               skipper,
        Options: openapi3filter.Options{
            AuthenticationFunc: authFunc,
        },
    })
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            req := c.Request()
            req = req.WithContext(ctxhelper.WithAuthn(req.Context()))
            c.SetRequest(req)
            return oapiValidator(next)(c)
        }
    }
}
```

ミドルウェアはオペレーションパスの検証をバイパスするスキッパー（`internal/controller/httpstack/oapi/skipper`）で設定される。仕様内の `security:` 宣言が `AuthenticationFunc` を駆動する。この関数はセキュリティ要件を宣言したオペレーションに対してのみ発火するため、パブリックエンドポイント（例: `GET /health`）は認証を要求されない。

**レスポンスは実行時に検証しない。** レスポンス契約は構造によって信頼される。ハンドラーコードは同一の仕様から生成されるため（ADR-0011）、型が正しいレスポンスは仕様に違反できない。

## 影響

### ポジティブな影響

- リクエストボディ、クエリパラメーター、パスパラメーターがハンドラーの呼び出し前に仕様に対して検証される。無効な入力はビジネスロジックに到達しない。
- 仕様内で宣言されたセキュリティ要件（`security:` ブロック）が自動的に強制される——設定された `AuthenticationFunc` を通過しないとハンドラーに到達できない。
- オペレーションパスはハンドラーごとのオプトアウトなしに除外される。スキッパーロジックは一元化されている。
- レスポンス検証によるランタイムオーバーヘッドがない。

### ネガティブな影響

- レスポンス値が HTTP パス外で生成された場合（例: レスポンススキーマに違反する値を持つシードされた行）、その違反はサーバー側では見えず、クライアント側にのみ現れる。これを防ぐ方向不変条件については [`openapi/boundary-ownership.md`](../../../openapi/boundary-ownership.md) を参照。
- ミドルウェアはコード生成に使用されるバンドル済み仕様と同じものをワイヤリングした状態に保たれなければならない。起動時に読み込まれた仕様ファイルが生成コードからドリフトした場合、ミドルウェアとハンドラーが静かに不一致になる可能性がある。

## 検討した代替案

### ハンドラーごとの手動検証

各ハンドラーが独自のバインディングおよび検証ロジックを呼び出す。却下：これはコード生成によって置き換えるべき現状の問題であり、煩雑でエラーが発生しやすく、省略が容易である。

### ランタイムレスポンス検証

送信レスポンスボディを仕様に対して検証する。却下：すべてのレスポンスに重大なレイテンシコストがかかり、strict-server の生成コードがすでにコンパイル時にレスポンス型が仕様に一致することを保証している。

## 補足

- ミドルウェア実装: [`internal/controller/httpstack/oapi/oapi.go`](../../../internal/controller/httpstack/oapi/oapi.go)。
- オペレーションパススキッパー: [`internal/controller/httpstack/oapi/skipper/skipper.go`](../../../internal/controller/httpstack/oapi/skipper/skipper.go)。
- `/metrics` の認証例外（このパイプラインからスキップされ、別の BasicAuth ミドルウェアで保護される）は [ADR-0014](0014-metrics-endpoint-auth-exception.ja.md) に記録されている。
- セキュリティと境界の注記: [`openapi/README.md`](../../../openapi/README.md)（§ Security）および [`openapi/boundary-ownership.md`](../../../openapi/boundary-ownership.md)。
- 親の決定: [ADR-0009](0009-openapi-first.ja.md)（OpenAPI ファースト）。
