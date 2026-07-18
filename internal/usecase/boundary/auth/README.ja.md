# auth

[English](README.md) | 日本語

認証に関するインターフェースと値オブジェクトを提供します。

## Credential の詳細

`Credential` は、受け取った認証情報をスキームに依存しない中立表現で保持します。

- `Scheme()` — 認証スキームを返す（例: `"Bearer"`。`SchemeBearer` を参照）
- `Token()` — トークンを返す

## Authn の詳細

識別の中核は Subject（token の `sub`）・Issuer（トークン発行者）・UserID（内部ユーザー ID）です。

- `Subject()` — 認証主体（token の `sub`）を返す
- `HasUserID()` — subject を UUID として解釈できた場合 true
- `UserID()` — 内部ユーザー UUID を返す（解釈できない場合はエラー）
- `Issuer()` — トークン発行者を返す（例: `"mock"`、IdP の issuer）
- `Scopes()` — スコープ一覧を返す（任意）
- `Claims()` — クレーム map を返す（任意、認可・UI 制御用）

## エラー

|エラー|説明|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|subject が空（`apperror.ErrUnauthenticated` をラップ）|
|`ErrSubjectNotUUID`|subject が UUID として解釈不可（`apperror.ErrValidation` をラップ）|
|`ErrTokenMissing`|トークンが空（`apperror.ErrInvalidArgument` をラップ）|

## 設計意図

- 「認証済み」という状態を型で表現する
- トークン解析ロジックを外側（Infrastructure）に押し出す
- subject の正規化（trim）と UUID 変換を内部にカプセル化

## 実装

`internal/infrastructure/auth/` に、`Credential` から `Authn` を生成する `Authenticator` インターフェースの環境別の具体実装が配置されています。
