# auth

[English](README.md) | 日本語

> このファイルは canonical な [README.md](README.md) の日本語訳です。直接編集せず、更新は README.md 側から反映してください。

認証に関するインターフェースと値オブジェクトを提供します。

## Credential の詳細

`Credential` は、受信した認証情報をスキームに依存しない形で表現します。

- `Scheme()` — 認証スキームを返します（例: `"Bearer"`。`SchemeBearer` を参照）
- `Token()` — トークンを返します

## Authn の詳細

アイデンティティの中核は Subject（token `sub`）・Issuer（token の発行者）・UserID（内部ユーザー ID）です。

- `Subject()` — 認証された subject（token `sub`）を返します
- `WithUserID()` — 内部 UserID を解決した複製を返します（アイデンティティ解決は認証とは分離されます）
- `HasUserID()` — 内部 UserID が解決済みかどうかを返します
- `UserID()` — 内部ユーザーの UUID を返します（未解決の場合は `ErrUserIDUnresolved`）
- `Issuer()` — token の発行者を返します（例: `"mock"`、IdP の issuer）
- `Scopes()` — スコープ一覧を返します（任意）
- `Claims()` — クレームのマップを返します（任意。認可 / UI 制御用）

`New()` は、UserID が**未解決**の状態で認証結果（subject + issuer）を生成します。内部ユーザーの解決は別の関心事（`IdentityResolver`）であり、`WithUserID()` を通じて適用されます。

## IdentityResolver の詳細

`IdentityResolver` は、認証済みの外部アイデンティティ（`Issuer` + `Subject`）を内部ユーザーへ解決し、UserID を解決した `Authn` の複製を返します。

- `Resolve(ctx, *Authn) (*Authn, error)` — `Issuer` + `Subject` でアイデンティティを検索し、`WithUserID()` を適用した `Authn` を返します
- 該当するアイデンティティが無い場合は `ErrIdentityNotFound`、解決したユーザーが利用不可（例: 論理削除済み）の場合は `ErrUserUnavailable`（いずれも fail-closed）
- 認証成功後に適用されるため、トークン検証の関心事（`Authenticator`）はユーザー検索から独立したままになります

## エラー

|エラー|説明|
|---|---|
|`ErrUnauthenticatedSubjectMissing`|Subject が空（`apperror.ErrUnauthenticated` をラップ）|
|`ErrUserIDUnresolved`|内部 UserID が未解決（`apperror.ErrUnauthenticated` をラップ）|
|`ErrTokenMissing`|トークンが空（`apperror.ErrUnauthenticated` をラップ）|
|`ErrIdentityNotFound`|issuer + subject に一致する内部ユーザーが存在しない（`apperror.ErrUnauthenticated` をラップ）|
|`ErrUserUnavailable`|解決したユーザーが利用不可（例: 論理削除済み）（`apperror.ErrUnauthenticated` をラップ）|

## 設計意図

- 「認証済み」の状態を型で表現する
- トークンのパースロジックは外側の層（Infrastructure）へ押し出す
- 認証（subject/issuer の抽出）と内部ユーザーの解決（`IdentityResolver` / `WithUserID`）を分離する

## 実装

`internal/infrastructure/auth/` が、`Credential` から `Authn` を生成する `Authenticator` インターフェースの環境別実装を提供します。既定の `IdentityResolver`（`internal/infrastructure/auth/identity/`）は UserID を未解決のまま通す passthrough で、内部ユーザーの解決はプロジェクト固有の実装を差し替えて提供します。
