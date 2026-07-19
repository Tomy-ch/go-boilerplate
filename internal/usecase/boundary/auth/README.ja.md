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
- `WithUserID()` — 内部 UserID を解決した複製を返す（内部ユーザー解決は認証とは別の関心事）
- `HasUserID()` — 内部 UserID が解決済みの場合 true
- `UserID()` — 内部ユーザー UUID を返す（未解決の場合は `ErrUserIDUnresolved`）
- `Issuer()` — トークン発行者を返す（例: `"mock"`、IdP の issuer）
- `Scopes()` — スコープ一覧を返す（任意）
- `Claims()` — クレーム map を返す（任意、認可・UI 制御用）

`New()` は認証結果（subject + issuer）を UserID **未解決**の状態で生成します。内部ユーザーの解決は別の関心事（`IdentityResolver` 相当）であり、`WithUserID()` で行います。

## エラー

|エラー|説明|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|subject が空（`apperror.ErrUnauthenticated` をラップ）|
|`ErrUserIDUnresolved`|内部 UserID が未解決（`apperror.ErrUnauthenticated` をラップ）|
|`ErrTokenMissing`|トークンが空（`apperror.ErrUnauthenticated` をラップ）|

## 設計意図

- 「認証済み」という状態を型で表現する
- トークン解析ロジックを外側（Infrastructure）に押し出す
- 認証（subject/issuer 抽出）と内部ユーザー解決（`WithUserID`）を分離する

## 実装

`internal/infrastructure/auth/` に、`Credential` から `Authn` を生成する `Authenticator` インターフェースの環境別の具体実装が配置されています。
