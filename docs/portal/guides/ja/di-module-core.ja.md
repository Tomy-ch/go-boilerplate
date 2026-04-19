# core モジュール

[English](README.md) | 日本語

`internal/di/module/core` は、HTTP スタックで共通利用される **コアコンポーネントの DI モジュール群**を提供するパッケージです。

各モジュールは `fx.Option` を返し、対応するコンポーネントを DI コンテナに登録します。

## モジュール一覧

|関数|ファイル|提供するコンポーネント|
|---|---|---|
|`AuthnModule()`|`auth.go`|認証（Authenticator + Auth コントローラ）|
|`BasicAuthModule()`|`basicauth.go`|メトリクスエンドポイント用 Basic 認証バリデータ|
|`IPRateLimiterModule()`|`ip_rate_limiter.go`|IP ベースのレートリミッター|
|`SecurityCookieModule()`|`security_cookie.go`|Cookie セキュリティ属性の設定|
|`SkipperModule()`|`skipper.go`|OpenAPI バリデーションの ops エンドポイントスキップ|
|`ValidatorModule()`|`validator.go`|OpenAPI スキーマバリデータ|

## 設計方針

- 各モジュールは1ファイル = 1モジュールで分離
- 内部実装は `internal/controller/httpstack` 等のコンストラクタを `fx.Provide` でラップするだけ
- ビジネスロジックを含まない

## 注意点

- モジュールの追加・削除は `internal/di/module` の上位モジュールから参照を変更する必要がある
- テストでは各モジュールが正しくコンポーネントを提供できることを検証している
