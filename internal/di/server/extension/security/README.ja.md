# security

[English](README.md) | 日本語

このディレクトリは、HTTP レイヤーにおける **セキュリティ関連ミドルウェア**（CORS・セキュリティヘッダ・Cookie 設定）を DI 経由で Echo に組み込むための fx モジュール群を提供します。

これらは統一された Priority で controller 層のミドルウェアパイプラインに適用されます。

## 役割

このパッケージは、組み合わせ可能なサーバーパイプラインのうち **セキュリティ**の関心事（セキュリティヘッダ・CORS・Cookie 属性）をひとつの DI 単位に分離します。これら横断的な保護を依存性注入の登録の背後にまとめることで、固定された競合のない優先度で適用し、単一の面としてレビューできます。同時に、セキュリティの知識を HTTP レイヤーに留め、domain / usecase 層へ漏らさないことを保証します。

## モジュール一覧

|モジュール|種別|説明|
|---|---|---|
|`Module()`|Use|セキュリティヘッダ（HSTS, X-Frame-Options, Content-Type-Options, Referrer-Policy）|
|`CORSModule()`|Use|CORS 設定（AllowOrigins / AllowMethods / AllowHeaders）|
|`CookieModule()`|Use|Cookie セキュリティ属性（Secure / HttpOnly / SameSite）|

## 注意点

- **Priority 順序**でミドルウェアが適用 — 他のミドルウェアと競合しないよう固定値で管理
- Cookie の `Secure` 属性は HTTPS 接続でのみ有効（ローカル HTTP 環境では挙動が変わる）
- `SecurityConfig` 更新時はミドルウェア設定の見直しを推奨
- セキュリティロジックは HTTP レイヤー専用 — **domain / usecase に知識を漏らさないこと**
