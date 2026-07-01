# oapi

[English](README.md) | 日本語

`oapi` は、Echo HTTP スタックに **リクエストバリデーションと認証を提供する OpenAPI 統合レイヤー**です。

スキーマバリデーション・認証・ルートスキップを1つの Echo ミドルウェアに統合するエントリポイントです。

## アーキテクチャ

```mermaid
flowchart TB
    Request["HTTP リクエスト"]
    Skipper{"Skipper"}
    Validate["OpenAPI スキーマバリデーション"]
    Auth["認証 (auth/)"]
    Authn["Authn → リクエストコンテキストのスロット"]
    Handler["Handler"]

    Request --> Skipper
    Skipper -- スキップ --> Handler
    Skipper -- バリデーション --> Validate
    Validate --> Auth
    Auth --> Authn --> Handler
```

1. **Skipper** がリクエストが ops エンドポイントかを判定 — ops ならバリデーションをバイパス
2. **Validator** がリクエストを OpenAPI 仕様に照合してバリデーション（パス、パラメータ、ボディ、Content-Type）
3. **Auth** がトークンを抽出し、boundary の `Authenticator` 経由で認証し、`Authn` をリクエストコンテキストのスロットに書き込む
4. Handler はバリデーション済み・認証済みのリクエストを受け取る

## サブパッケージ

|パッケージ|説明|詳細|
|---|---|---|
|`auth/`|Cookie / Header からのトークン認証|[README](auth/README.ja.md)|
|`skipper/`|ops エンドポイントのバリデーションスキップ|[README](skipper/README.ja.md)|
|`validator/`|OpenAPI スキーマの読み込みと提供|[README](validator/README.ja.md)|

## 依存関係

|パッケージ|役割|
|---|---|
|`kin-openapi/openapi3`|OpenAPI 3.x スキーマモデル|
|`kin-openapi/openapi3filter`|リクエストバリデーションと認証フィルタ|
|`oapi-codegen/echo-middleware`|OpenAPI バリデーションの Echo アダプタ|
|`ctxhelper`|リクエストコンテキストへの Authn スロット注入と get/set|
|`boundary/auth`|認証インターフェースと `Authn` 値オブジェクト|

## 注意点

- OpenAPI バリデーションはパスパラメータ、クエリパラメータ、リクエストボディ、Content-Type をカバー
- 認証は OpenAPI 仕様に `security` が定義されたエンドポイントでのみトリガーされる
- `Skipper` により ops エンドポイントはバリデーション・認証の対象外
- oapi-codegen バリデータに委譲する前に、`ctxhelper.WithAuthn` で空の `Authn` スロットを `request.Context()` に注入し、認証関数（`context.Context` のみを受け取る）が `ctxhelper.SetAuthn` で認証済みの `Authn` をそのスロットに書き込めるようにする。後段のハンドラは `ctxhelper.GetAuthn` で読み出す
- このレイヤーからのエラーは `errorhandler` で捕捉され、適切な HTTP レスポンスに変換される
