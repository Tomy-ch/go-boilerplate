# auth

[English](README.md) | 日本語

認証に関するインターフェースと値オブジェクトを提供します。

## Authn の詳細

- `Subject()` — 認証主体（例: userID）を返す
- `HasID()` — subject が UUID として解釈できた場合 true
- `ID()` — UUID を返す（解釈できない場合はエラー）
- `Provider()` — 認証プロバイダ名を返す（例: "mock", "google"）
- `Scopes()` — スコープ一覧を返す（任意）
- `Claims()` — クレーム map を返す（任意、認可・UI 制御用）

## エラー

|エラー|説明|
|---|---|
|`ErrUnauthorizedSubjectMissing`|subject が空（`apperror.ErrUnauthenticated` をラップ）|
|`ErrInvalidIDMissing`|subject が UUID として解釈不可（`apperror.ErrValidation` をラップ）|
|`ErrArgumentTokenMissing`|アクセストークンが空（`apperror.ErrInvalidArgument` をラップ）|

## 設計意図

- 「認証済み」という状態を型で表現する
- トークン解析ロジックを外側（Infrastructure）に押し出す
- subject の正規化（trim）と UUID 変換を内部にカプセル化

## 実装

`internal/infrastructure/auth/` に、`Credential` から `Authn` を生成する `Authenticator` インターフェースの環境別の具体実装が配置されています。
